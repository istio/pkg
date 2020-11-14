// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ledger

import (
	"bytes"
	"sync"
	"time"

	"istio.io/pkg/cache"
)

// The smt is derived from https://github.com/aergoio/SMT with modifications
// to remove unneeded features, and to support retention of old nodes for a fixed time.
// The aergoio smt license is as follows:
/*
MIT License

Copyright (c) 2018 aergo

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
© 2019 GitHub, Inc.
*/

// TODO when using the smt, make sure keys and values are same length as hash

// smt is a sparse Merkle tree.
type smt struct {
	rootMu sync.RWMutex
	// root is the current root of the smt.
	root []byte
	// defaultHashes are the default values of empty trees
	defaultHashes [][]byte
	// db holds the cache and related locks
	db *cacheDB
	// hash is the hash function used in the trie
	hash func(data ...[]byte) []byte
	// trieHeight is the number if bits in a key
	trieHeight byte
	// the minimum length of time old nodes will be retained.
	retentionDuration time.Duration
	// lock is for the whole struct
	lock sync.RWMutex
	// atomicUpdate, commit all the changes made by intermediate update calls
	atomicUpdate bool
}

// this is the closest time.Duration comes to Forever, with a duration of ~145 years
// we can'tree use int64 max because the duration gets added to Now(), and the ints
// rollover, causing an immediate expiration (ironic, eh?)
const forever time.Duration = 1<<(63-1) - 1

// newSMT creates a new smt given a keySize, hash function, cache (nil will be defaulted to TTLCache), and retention
// duration for old nodes.
func newSMT(hash func(data ...[]byte) []byte, updateCache cache.ExpiringCache, retentionDuration time.Duration) *smt {
	if updateCache == nil {
		updateCache = cache.NewTTL(forever, time.Second)
	}
	s := &smt{
		hash:              hash,
		trieHeight:        byte(len(hash([]byte("height"))) * 8), // hash any string to get output length
		retentionDuration: retentionDuration,
	}
	s.db = &cacheDB{
		updatedNodes: byteCache{cache: updateCache},
	}
	s.loadDefaultHashes()
	return s
}

func (s *smt) Root() []byte {
	s.rootMu.RLock()
	defer s.rootMu.RUnlock()
	return s.root
}

// loadDefaultHashes creates the default hashes
func (s *smt) loadDefaultHashes() {
	s.defaultHashes = make([][]byte, s.trieHeight+1)
	s.defaultHashes[0] = defaultLeaf
	var h []byte
	for i := byte(1); i <= s.trieHeight; i++ {
		h = s.hash(s.defaultHashes[i-1], s.defaultHashes[i-1])
		s.defaultHashes[i] = h
	}
}

// Update adds a sorted list of keys and their values to the trie
// If Update is called multiple times, only the state after the last update
// is committed.
// When calling Update multiple times without commit, make sure the
// values of different keys are unique(hash contains the key for example)
// otherwise some subtree may get overwritten with the wrong hash.
func (s *smt) Update(keys, values [][]byte) ([]byte, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.atomicUpdate = true
	ch := make(chan result, 1)
	//s.update(s.Root(), keys, values, nil, 0, s.trieHeight, false, true, ch)
	n, err := buildRootNode(s.Root(), s.trieHeight, s.db)
	if err != nil {
		return nil, err
	}
	s.update(n, keys, values, ch)
	result := <-ch
	if result.err != nil {
		return nil, result.err
	}
	s.rootMu.Lock()
	defer s.rootMu.Unlock()
	if len(result.update) != 0 {
		s.root = result.update[:hashLength]
	} else {
		s.root = nil
	}

	return s.root, nil
}

// result is used to contain the result of goroutines and is sent through a channel.
type result struct {
	update []byte
	err    error
}

func (s *smt) update(node *node, keys, values [][]byte, ch chan<- result) {
	if node.height() == 0 {
		if bytes.Equal(values[0], defaultLeaf) {
			ch <- result{nil, nil}
		} else {
			node.val = values[0]
			ch <- result{values[0], nil}
		}
		return
	}
	if node.isShortcut() {
		// if this is a delete operation, we have either arrived at the key to delete, or there is nothing to delete
		deletes := getDeleteIndices(values)
		if len(deletes) > 0 {
			// we are about to mutate keys and values
			// copy to avoid side-effects
			keys = copy2d(keys)
			values = copy2d(values)
			for i := range deletes {
				if node.left() != nil && bytes.Equal(keys[i], node.left().val[:hashLength]) {
					node.removeShortcut()
					node.val = node.calculateHash(s.hash, s.defaultHashes)
				}
				keys[i] = nil
				values[i] = nil
			}
			keys = removeNils(keys)
			values = removeNils(values)
			if len(keys) == 0 {
				ch <- result{node.val, nil}
				return
			}
		}
	}
	// if node is still a shortcut, proceed as normal
	if node.isShortcut() {
		keys, values = s.maybeAddShortcutToKV(keys, values, node.left().val[:hashLength], node.right().val[:hashLength])
		// remove shortcut notation
		node.removeShortcut()
	}
	// Split the keys array so each branch can be updated in parallel
	// Does this require that keys are sorted?  Yes, see Update()
	lkeys, rkeys := s.splitKeys(keys, s.trieHeight-node.height()) //off by one?
	splitIndex := len(lkeys)
	lvalues, rvalues := values[:splitIndex], values[splitIndex:]

	if node.left() == nil && node.right() == nil && len(keys) == 1 {
		if !bytes.Equal(values[0], defaultLeaf) {
			// we can store this as a shortcut
			node.makeShortcut(keys[0], values[0])
		} else {
			// if the subtree contains only one key, store the key/value in a shortcut node
			// TODO: this
			//store = false
		}
	} else {
		switch {
		case len(lkeys) == 0 && len(rkeys) > 0:
			node.initRight()
			newch := make(chan result, 1)
			s.update(node.right(), rkeys, rvalues, newch)
			res := <-newch
			if res.err != nil {
				ch <- result{nil, res.err}
				return
			}
		case len(lkeys) > 0 && len(rkeys) == 0:
			node.initLeft()
			newch := make(chan result, 1)
			s.update(node.left(), lkeys, lvalues, newch)
			res := <-newch
			if res.err != nil {
				ch <- result{nil, res.err}
				return
			}
		default:
			lch := make(chan result, 1)
			rch := make(chan result, 1)
			node.initRight()
			node.initLeft()
			go s.update(node.left(), lkeys, lvalues, lch)
			go s.update(node.right(), rkeys, rvalues, rch)
			lresult := <-lch
			rresult := <-rch
			if lresult.err != nil {
				ch <- result{nil, lresult.err}
				return
			}
			if rresult.err != nil {
				ch <- result{nil, rresult.err}
				return
			}
		}
	}
	node.val = node.calculateHash(s.hash, s.defaultHashes)
	node.store()
	ch <- result{node.val, nil}
	return
}

func removeNils(keys [][]byte) (result [][]byte) {
	for _, k := range keys {
		if k != nil {
			result = append(result, k)
		}
	}
	return
}

func getDeleteIndices(values [][]byte) (result []int) {
	for i, v := range values {
		if bytes.Equal(v, defaultLeaf) {
			result = append(result, i)
		}
	}
	return
}

// splitKeys divides the array of keys into 2 so they can update left and right branches in parallel
func (s *smt) splitKeys(keys [][]byte, height byte) ([][]byte, [][]byte) {
	for i, key := range keys {
		if bitIsSet(key, height) {
			return keys[:i], keys[i:]
		}
	}
	return keys, nil
}

// maybeAddShortcutToKV adds a shortcut key to the keys array to be updated.
// this is used when a subtree containing a shortcut node is being updated
func (s *smt) maybeAddShortcutToKV(keys, values [][]byte, shortcutKey, shortcutVal []byte) ([][]byte, [][]byte) {
	newKeys := make([][]byte, 0, len(keys)+1)
	newVals := make([][]byte, 0, len(keys)+1)

	if bytes.Compare(shortcutKey, keys[0]) < 0 {
		newKeys = append(newKeys, shortcutKey)
		newKeys = append(newKeys, keys...)
		newVals = append(newVals, shortcutVal)
		newVals = append(newVals, values...)
	} else if bytes.Compare(shortcutKey, keys[len(keys)-1]) > 0 {
		newKeys = append(newKeys, keys...)
		newKeys = append(newKeys, shortcutKey)
		newVals = append(newVals, values...)
		newVals = append(newVals, shortcutVal)
	} else {
		higher := false
		for i, key := range keys {
			if bytes.Equal(shortcutKey, key) {
				// the shortcut keys is being updated
				return keys, values
			}
			if !higher && bytes.Compare(shortcutKey, key) > 0 {
				higher = true
				continue
			}
			if higher && bytes.Compare(shortcutKey, key) < 0 {
				// insert shortcut in slices
				newKeys = append(newKeys, keys[:i]...)
				newKeys = append(newKeys, shortcutKey)
				newKeys = append(newKeys, keys[i:]...)
				newVals = append(newVals, values[:i]...)
				newVals = append(newVals, shortcutVal)
				newVals = append(newVals, values[i:]...)
				break
			}
		}
	}
	return newKeys, newVals
}

const batchLen int = 31
const batchHeight int = 4 // this is log2(batchLen+1)-1
