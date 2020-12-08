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
	"fmt"
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
	// lock is for the whole struct
	lock sync.RWMutex
}

// this is the closest time.Duration comes to Forever, with a duration of ~145 years
// we can'tree use int64 max because the duration gets added to Now(), and the ints
// rollover, causing an immediate expiration (ironic, eh?)
const forever time.Duration = 1<<(63-1) - 1

// newSMT creates a new smt given a keySize, hash function, cache (nil will be defaulted to TTLCache), and retention
// duration for old nodes.
func newSMT(hash func(data ...[]byte) []byte, updateCache cache.ExpiringCache) *smt {
	if updateCache == nil {
		updateCache = cache.NewTTL(forever, 0)
	}
	s := &smt{
		hash:       hash,
		trieHeight: byte(len(hash([]byte("height"))) * 8), // hash any string to get output length
	}
	s.db = &cacheDB{
		updatedNodes: byteCache{cache: updateCache},
	}
	s.loadDefaultHashes()
	return s
}

// loadDefaultHashes creates the default hashes
func (s *smt) loadDefaultHashes() {
	s.defaultHashes = make([][]byte, s.trieHeight+1)
	s.defaultHashes[0] = hasher([]byte{0x0})
	var h []byte
	for i := byte(1); i <= s.trieHeight; i++ {
		h = s.hash(s.defaultHashes[i-1], s.defaultHashes[i-1])
		s.defaultHashes[i] = h
	}
}

func (s *smt) Delete(key []byte) ([]byte, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	n, err := buildRootNode(s.Root(), s.trieHeight, s.db)
	if err != nil {
		return nil, err
	}
	newRoot, _, _ := s.delete(n, key)
	s.rootMu.Lock()
	defer s.rootMu.Unlock()
	if len(newRoot) != 0 {
		s.root = newRoot
	} else {
		s.root = nil
	}
	return s.root, nil
}

// Update adds a sorted list of keys and their values to the trie
func (s *smt) Update(keys, values [][]byte) ([]byte, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	ch := make(chan result, 1)
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
		s.root = result.update
	} else {
		s.root = nil
	}

	return s.root, nil
}

func (s *smt) delete(n *node, key []byte) (newVal, reloKey, reloValue []byte) {
	defer n.store()
	if n.height() == 0 {
		n.val = nil
		return nil, nil, nil
	}
	if n.isShortcut() {
		if n.left() != nil && bytes.Equal(key, n.left().val[:hashLength]) {
			n.removeShortcut()
			n.val = nil
			return nil, nil, nil
		}
		return n.val, nil, nil
	}
	var keyChild, altChild *node // keyChild is the child n traversed in search of the key
	// altChild is the n not traveled, important for relocating shortcuts
	if bitIsSet(key, s.trieHeight-n.height()) {
		// recurse right
		keyChild = n.right()
		altChild = n.left()
	} else {
		//recurse left
		keyChild = n.left()
		altChild = n.right()
	}
	var childVal []byte
	if keyChild != nil {
		childVal, reloKey, reloValue = s.delete(keyChild, key)
	}
	if reloKey != nil && altChild != nil {
		keyChild.makeShortcut(reloKey, reloValue)
		keyChild.val = keyChild.calculateHash(s.hash, s.defaultHashes)
		reloKey, reloValue = nil, nil
		childVal = keyChild.val
		keyChild.store()
	}
	if childVal == nil && altChild != nil && altChild.isShortcut() {
		reloKey = altChild.left().val
		reloValue = altChild.right().val
		altChild.removeShortcut()
		altChild.val = nil
	}
	if childVal == nil && altChild == nil {
		n.val = nil
	} else {
		n.val = n.calculateHash(s.hash, s.defaultHashes)
	}
	newVal = n.val
	return
}

// result is used to contain the result of goroutines and is sent through a channel.
type result struct {
	update []byte
	err    error
}

func (s *smt) update(node *node, keys, values [][]byte, ch chan<- result) {
	if node.height() == 0 {
		// update this value
		node.val = values[0]
		ch <- result{update: values[0]}
		return
	}
	// if node a shortcut, it needs to be relocated further down the tree with one of our updated keys
	if node.isShortcut() {
		keys, values = s.maybeAddShortcutToKV(keys, values, node.left().val[:hashLength], node.right().val)
		// remove shortcut notation
		node.removeShortcut()
	}
	// Split the keys array so each branch can be updated in parallel
	// Does this require that keys are sorted?  Yes, see Update()
	lkeys, rkeys := s.splitKeys(keys, s.trieHeight-node.height()) //off by one?
	splitIndex := len(lkeys)
	lvalues, rvalues := values[:splitIndex], values[splitIndex:]

	if node.left() == nil && node.right() == nil && len(keys) == 1 {
		// we can store this as a shortcut
		node.makeShortcut(keys[0], values[0])
	} else {
		switch {
		case len(lkeys) == 0 && len(rkeys) > 0:
			node.initRight()
			newch := make(chan result, 1)
			s.update(node.right(), rkeys, rvalues, newch)
			res := <-newch
			if res.err != nil {
				ch <- result{err: res.err}
				return
			}
		case len(lkeys) > 0 && len(rkeys) == 0:
			node.initLeft()
			newch := make(chan result, 1)
			s.update(node.left(), lkeys, lvalues, newch)
			res := <-newch
			if res.err != nil {
				ch <- result{err: res.err}
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
				ch <- result{err: lresult.err}
				return
			}
			if rresult.err != nil {
				ch <- result{err: rresult.err}
				return
			}
		}
	}
	node.val = node.calculateHash(s.hash, s.defaultHashes)
	node.store()
	ch <- result{update: node.val}
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

// Erase will remove from the cache any pages which do not exist in the next or previous trie.
func (s *smt) Erase(rootHash []byte, adjacents [][]byte) error {
	rootNode, err := buildRootNode(rootHash, s.trieHeight, s.db)
	if err != nil {
		return fmt.Errorf("failed to retrieve root node: %s", err)
	}
	var adjacentNodes []*node
	for _, n := range adjacents {
		nextNode, err := buildRootNode(n, s.trieHeight, s.db)
		if err != nil {
			return fmt.Errorf("failed to retrieve next node: %s", err)
		}
		adjacentNodes = append(adjacentNodes, nextNode)
	}
	s.eraseRecursive(rootNode, adjacentNodes)
	return nil
}

func (s *smt) eraseRecursive(rootHash *node, adjacentNodes []*node) {
	if rootHash == nil {
		return
	}
	var anyMatch bool
	var lefts, rights []*node
	for _, n := range adjacentNodes {
		if n != nil {
			lefts = append(lefts, n.left())
			rights = append(rights, n.right())
			if bytes.Equal(n.val, rootHash.val) {
				anyMatch = true
			}
		}
	}
	if !anyMatch {
		// erase this rootHash if it's the root of a page
		if rootHash.isLeaf() {
			// populate next page before deleting from the cache
			rootHash.getNextPage().delete()
		} else if rootHash.isShortcut() {
			return
		}
		// maybe make this parallel?
		s.eraseRecursive(rootHash.left(), lefts)
		s.eraseRecursive(rootHash.right(), rights)
	}
}

const batchLen int = 31
const batchHeight int = 4 // this is log2(batchLen+1)-1
