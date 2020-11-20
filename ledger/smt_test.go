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
	"math/rand"
	"runtime"
	"sort"
	"testing"
	"time"

	"gotest.tools/assert"

	"istio.io/pkg/cache"
)

func TestSmtEmptyTrie(t *testing.T) {
	smt := newSMT(hasher, nil)
	if !bytes.Equal([]byte{}, smt.root) {
		t.Fatal("empty trie root hash not correct")
	}
}

func TestSmtUpdateAndGet(t *testing.T) {
	smt := newSMT(hasher, nil)

	// Add data to empty trie
	keys := getFreshData(10)
	values := getFreshData(10)
	root, err := smt.Update(keys, values)
	assert.NilError(t, err)

	// Check all keys have been stored
	for i, key := range keys {
		value, err := smt.Get(key)
		assert.NilError(t, err)
		if !bytes.Equal(values[i], value) {
			t.Fatal("value not updated")
		}
	}

	// Append to the trie
	newKeys := getFreshData(5)
	newValues := getFreshData(5)
	newRoot, err := smt.Update(newKeys, newValues)
	assert.NilError(t, err)
	if bytes.Equal(root, newRoot) {
		t.Fatal("trie not updated")
	}
	for i, newKey := range newKeys {
		newValue, err := smt.Get(newKey)
		assert.NilError(t, err)
		if !bytes.Equal(newValues[i], newValue) {
			t.Fatal("failed to get value")
		}
	}
	// Check old keys are still stored
	for i, key := range keys {
		value, err := smt.Get(key)
		assert.NilError(t, err)
		if !bytes.Equal(values[i], value) {
			t.Fatal("failed to get value")
		}
	}
}

func TestTrieAtomicUpdate(t *testing.T) {
	smt := newSMT(hasher, nil)
	keys := getFreshData(10)
	values := getFreshData(10)
	_, err := smt.Update(keys, values)
	assert.NilError(t, err)

	// check keys of previous atomic update are accessible in
	// updated nodes with root.
	for i, key := range keys {
		value, err := smt.Get(key)
		assert.NilError(t, err)
		if !bytes.Equal(values[i], value) {
			t.Fatal("failed to get value")
		}
	}
}

func TestSmtPublicUpdateAndGet(t *testing.T) {
	smt := newSMT(hasher, nil)
	// Add data to empty trie
	keys := getFreshData(5)
	values := getFreshData(5)
	root, _ := smt.Update(keys, values)

	// Check all keys have been stored
	for i, key := range keys {
		value, _ := smt.Get(key)
		if !bytes.Equal(values[i], value) {
			t.Fatal("trie not updated")
		}
	}
	if !bytes.Equal(root, smt.root) {
		t.Fatal("root not stored")
	}

	newValues := getFreshData(5)
	_, err := smt.Update(keys, newValues)
	assert.NilError(t, err)

	// Check all keys have been modified
	for i, key := range keys {
		value, _ := smt.Get(key)
		if !bytes.Equal(newValues[i], value) {
			t.Fatal("trie not updated")
		}
	}

	newKeys := getFreshData(5)
	newValues = getFreshData(5)
	_, err = smt.Update(newKeys, newValues)
	assert.NilError(t, err)
	for i, key := range newKeys {
		value, _ := smt.Get(key)
		if !bytes.Equal(newValues[i], value) {
			t.Fatal("trie not updated")
		}
	}
}

func TestSmtDelete(t *testing.T) {
	smt := newSMT(hasher, nil)
	// Add data to empty trie
	keys := getFreshData(10)
	values := getFreshData(10)
	_, err := smt.Update(keys, values)
	assert.NilError(t, err)
	value, err := smt.Get(keys[0])
	assert.NilError(t, err)
	if !bytes.Equal(values[0], value) {
		t.Fatal("trie not updated")
	}

	// Delete from trie
	// To delete a key, just set it's value to Default leaf hash.
	newRoot, err := smt.Delete(keys[0])
	assert.NilError(t, err)
	validate(t, smt)
	newValue, err := smt.Get(keys[0])
	assert.NilError(t, err)
	if len(newValue) != 0 {
		t.Fatal("Failed to delete from trie")
	}
	// Remove deleted key from keys and check root with a clean trie.
	smt2 := newSMT(hasher, nil)
	cleanRoot, err := smt2.Update(keys[1:], values[1:])
	assert.NilError(t, err)
	keys1, values1, err := smt.GetAll()
	assert.NilError(t, err)
	keys2, values2, err := smt2.GetAll()
	equalByteArrays(t, keys1, keys2)
	equalByteArrays(t, values1, values2)
	assert.NilError(t, err)
	// this assertion is probably failing because deleting doesn't restructure dangling shortcuts.  Shouldn't hash(nil, x) = x?
	assert.Assert(t, bytes.Equal(newRoot, cleanRoot),
		"identical trees produced different roots! [%v] [%v]", newRoot, cleanRoot)

	//Empty the trie
	var root []byte
	for _, k := range keys {
		root, err = smt.Delete(k)
		assert.NilError(t, err)
	}
	if len(root) != 0 {
		t.Fatal("empty trie root hash not correct")
	}
	// Test deleting an already empty key
	smt = newSMT(hasher, nil)
	keys = getFreshData(2)
	values = getFreshData(2)
	root, err = smt.Update(keys, values)
	assert.NilError(t, err)
	key0 := make([]byte, 8)
	key1 := make([]byte, 8)

	_, err = smt.Delete(key0)
	assert.NilError(t, err)
	_, err = smt.Delete(key1)
	assert.NilError(t, err)
	if !bytes.Equal(root, smt.root) {
		// this is failing due to some sort of interaction between the shortcut and the delete
		t.Fatal("deleting a default key shouldn't modify the tree")
	}
}

func equalByteArrays(t *testing.T, left, right [][]byte) {
	assert.Equal(t, len(left), len(right), "byte arrays are not of equal length")
	for i, l := range left {
		assert.Assert(t, bytes.Equal(l, right[i]), "byte array index %d is not equal", i)
	}
}

func validate(t *testing.T, s *smt) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	n, err := buildRootNode(s.root, s.trieHeight, s.db)
	assert.NilError(t, err)
	validateRecursive(t, n, s.hash, s.defaultHashes)
}

func validateRecursive(t *testing.T, n *node, hasher func(data ...[]byte) []byte, defaultHashes [][]byte) {
	nilChildren := func(n *node, expect bool) error {
		actLeft := n.left() == nil
		actRight := n.right() == nil
		if actLeft != expect {
			return fmt.Errorf("left node == nil should be %v, but is %v", expect, actLeft)
		}
		if actRight != expect {
			return fmt.Errorf("left node == nil should be %v, but is %v", expect, actRight)
		}
		return nil
	}
	correctVal := n.calculateHash(hasher, defaultHashes)
	assert.Assert(t, bytes.Equal(n.val, correctVal), "incorrect node value")
	if n.isShortcut() {
		assert.NilError(t, nilChildren(n, false), "shortcut children cannot be nil")
		assert.NilError(t, nilChildren(n.left(), true), "shortcut cannot have grandchildren")
		assert.NilError(t, nilChildren(n.right(), true), "shortcut cannot have grandchildren")
	} else {
		if n.left() != nil {
			validateRecursive(t, n.left(), hasher, defaultHashes)
		}
		if n.right() != nil {
			validateRecursive(t, n.right(), hasher, defaultHashes)
		}
	}
}

// test updating and deleting at the same time
func TestTrieUpdateAndDelete(t *testing.T) {
	smt := newSMT(hasher, nil)
	key0 := make([]byte, 8)
	values := getFreshData(1)
	root, _ := smt.Update([][]byte{key0}, values)
	node, err := buildRootNode(root, smt.trieHeight, smt.db)
	assert.NilError(t, err)
	if !node.isShortcut() || !bytes.Equal(node.left().val[:hashLength], key0) || !bytes.Equal(node.right().val[:hashLength], values[0]) {
		t.Fatal("leaf shortcut didn'tree move up to root")
	}

	key1 := make([]byte, 8)
	// set the last bit
	bitSet(key1, 63)
	_, err = smt.Delete(key0)
	assert.NilError(t, err)
	_, err = smt.Update([][]byte{key1}, getFreshData(1))
	assert.NilError(t, err)
}
func bitSet(bits []byte, i int) {
	bits[i/8] |= 1 << uint(7-i%8)
}

func TestSmtRaisesError(t *testing.T) {

	smt := newSMT(hasher, nil)
	// Add data to empty trie
	keys := getFreshData(10)
	values := getFreshData(10)
	_, err := smt.Update(keys, values)
	assert.NilError(t, err)
	smt.db.updatedNodes = byteCache{cache: cache.NewTTL(forever, time.Minute)}
	smt.loadDefaultHashes()

	// Check errors are raised is a key is not in cache nor db
	for _, key := range keys {
		_, err := smt.Get(key)
		assert.ErrorContains(t, err, "is unknown",
			"Error not created if database doesnt have a node")
	}

}

func getFreshData(size int) [][]byte {
	length := 8
	var data [][]byte
	for i := 0; i < size; i++ {
		key := make([]byte, 8)
		_, err := rand.Read(key)
		if err != nil {
			panic(err)
		}
		data = append(data, hasher(key)[:length])
	}
	sort.Sort(dataArray(data))
	return data
}

func benchmark10MAccounts10Ktps(smt *smt, b *testing.B) {
	fmt.Println("\nLoading b.N x 1000 accounts")
	for index := 0; index < b.N; index++ {
		newkeys := getFreshData(1000)
		newvalues := getFreshData(1000)
		start := time.Now()
		smt.Update(newkeys, newvalues)
		end := time.Now()
		end2 := time.Now()
		for i, key := range newkeys {
			val, _ := smt.Get(key)
			if !bytes.Equal(val, newvalues[i]) {
				b.Fatal("new key not included")
			}
		}
		end3 := time.Now()
		elapsed := end.Sub(start)
		elapsed2 := end2.Sub(end)
		elapsed3 := end3.Sub(end2)
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		fmt.Println(index, " : update time : ", elapsed, "commit time : ", elapsed2,
			"\n1000 Get time : ", elapsed3,
			"\nRAM : ", m.Sys/1024/1024, " MiB")
	}
}

//go test -run=xxx -bench=. -benchmem -test.benchtime=20s
func BenchmarkCacheHeightLimit(b *testing.B) {
	smt := newSMT(hasher, cache.NewTTL(forever, time.Minute))
	benchmark10MAccounts10Ktps(smt, b)
}
