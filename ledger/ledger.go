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

// Package ledger implements a modified map with three unique characteristics:
// 1. every unique state of the map is given a unique hash
// 2. prior states of the map are retained for a fixed period of time
// 2. given a previous hash, we can retrieve a previous state from the map, if it is still retained.
package ledger

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/spaolacci/murmur3"
)

// Ledger exposes a modified map with three unique characteristics:
// 1. every unique state of the map is given a unique hash
// 2. prior states of the map are retained for a fixed period of time
// 2. given a previous hash, we can retrieve a previous state from the map, if it is still retained.
type Ledger interface {
	// Put adds or overwrites a key in the Ledger
	Put(key, value string) (string, error)
	// Delete removes a key from the Ledger, which may still be read using GetPreviousValue
	Delete(key string) error
	// Get returns a the value of the key from the Ledger's current state
	Get(key string) (string, error)
	// RootHash is the hash of all keys and values currently in the Ledger
	RootHash() string
	// GetPreviousValue executes a get against a previous version of the ledger, using that version's root hash.
	GetPreviousValue(previousRootHash, key string) (result string, err error)
	// EraseRootHash re-claims any memory used by this version of history, preserving bits shared with other versions.
	EraseRootHash(rootHash string) error
}

type smtLedger struct {
	tree    *smt
	history *history
}

// Make returns a Ledger which will retain previous nodes after they are deleted.
// the retention parameter has been removed in favor of EraseRootHash, but is left
// here for backwards compatibility
func Make(_ time.Duration) Ledger {
	return smtLedger{tree: newSMT(hasher, nil), history: newHistory()}
}

func (s smtLedger) EraseRootHash(rootHash string) error {
	// occurences is a list of every time in (underased) history when this hash has been observed
	occurences := s.history.Get(rootHash)
	if occurences == nil || len(occurences) == 0 {
		return fmt.Errorf("rootHash %s is not present in ledger history", rootHash)
	}
	var adjacentRoots [][]byte
	for _, o := range occurences {
		adjacentRoots = append(adjacentRoots, o.Prev().Value.([]byte), o.Next().Value.([]byte))
	}
	err := s.tree.Erase(occurences[0].Value.([]byte), adjacentRoots)
	if err != nil {
		return err
	}
	for _, o := range occurences {
		s.history.Remove(o)
	}
	s.history.lock.Lock()
	defer s.history.lock.Unlock()
	delete(s.history.index, rootHash)
	return nil
}

// Put adds a key value pair to the ledger, overwriting previous values and marking them for
// removal after the retention specified in Make()
func (s smtLedger) Put(key, value string) (result string, err error) {
	b, err := s.tree.Update([][]byte{coerceKeyToHashLen(key)}, [][]byte{coerceToHashLen(value)})
	s.history.Put(b)
	result = s.RootHash()
	return
}

// Delete removes a key value pair from the ledger, marking it for removal after the retention specified in Make()
func (s smtLedger) Delete(key string) error {
	b, err := s.tree.Delete(coerceKeyToHashLen(key))
	if err != nil {
		return err
	}
	s.history.Put(b)
	return nil
}

// GetPreviousValue returns the value of key when the ledger's RootHash was previousHash, if it is still retained.
func (s smtLedger) GetPreviousValue(previousRootHash, key string) (result string, err error) {
	prevBytes, err := base64.StdEncoding.DecodeString(previousRootHash)
	if err != nil {
		return "", err
	}
	b, err := s.tree.GetPreviousValue(prevBytes, coerceKeyToHashLen(key))
	var i int
	// trim leading 0's from b
	for i = range b {
		if b[i] != 0 {
			break
		}
	}
	result = string(b[i:])
	return
}

// Get returns the current value of key.
func (s smtLedger) Get(key string) (result string, err error) {
	return s.GetPreviousValue(s.RootHash(), key)
}

// RootHash represents the hash of the current state of the ledger.
func (s smtLedger) RootHash() string {
	return base64.StdEncoding.EncodeToString(s.tree.Root())
}

func coerceKeyToHashLen(val string) []byte {
	hasher := murmur3.New64()
	_, _ = hasher.Write([]byte(val))
	return hasher.Sum(nil)
}

func coerceToHashLen(val string) []byte {
	// hash length is fixed at 64 bits until generic support is added
	const hashLen = 64
	byteVal := []byte(val)
	if len(byteVal) < hashLen/8 {
		// zero fill the left side of the slice
		zerofill := make([]byte, hashLen/8)
		byteVal = append(zerofill[:hashLen/8-len(byteVal)], byteVal...)
	}
	return byteVal[:hashLen/8]
}
