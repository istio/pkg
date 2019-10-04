/**
 *  @file
 *  @copyright defined in aergo/LICENSE.txt
 */

package ledger

import (
	"bytes"
	"fmt"
	"istio.io/pkg/cache"
	"math/rand"
	"runtime"
	"sort"
	"testing"
	"time"
)

func TestSmtEmptyTrie(t *testing.T) {
	smt := NewSMT(nil, Hasher, nil, time.Minute)
	if !bytes.Equal([]byte{}, smt.Root) {
		t.Fatal("empty trie root hash not correct")
	}
}

func TestSmtUpdateAndGet(t *testing.T) {
	smt := NewSMT(nil, Hasher, nil, time.Minute)
	smt.atomicUpdate = false

	// Add data to empty trie
	keys := getFreshData(10, 8)
	values := getFreshData(10, 8)
	ch := make(chan result, 1)
	smt.update(smt.Root, keys, values, nil, 0, smt.TrieHeight, false, true, ch)
	res := <-ch
	root := res.update

	// Check all keys have been stored
	for i, key := range keys {
		value, _ := smt.get(root, key, nil, 0, smt.TrieHeight)
		if !bytes.Equal(values[i], value) {
			t.Fatal("value not updated")
		}
	}

	// Append to the trie
	newKeys := getFreshData(5, 8)
	newValues := getFreshData(5, 8)
	ch = make(chan result, 1)
	smt.update(root, newKeys, newValues, nil, 0, smt.TrieHeight, false, true, ch)
	res = <-ch
	newRoot := res.update
	if bytes.Equal(root, newRoot) {
		t.Fatal("trie not updated")
	}
	for i, newKey := range newKeys {
		newValue, _ := smt.get(newRoot, newKey, nil, 0, smt.TrieHeight)
		if !bytes.Equal(newValues[i], newValue) {
			t.Fatal("failed to get value")
		}
	}
	// Check old keys are still stored
	for i, key := range keys {
		value, _ := smt.get(newRoot, key, nil, 0, smt.TrieHeight)
		if !bytes.Equal(values[i], value) {
			t.Fatal("failed to get value")
		}
	}
}

func TestTrieAtomicUpdate(t *testing.T) {
	smt := NewSMT(nil, Hasher, nil, time.Minute)
	smt.CacheHeightLimit = 0
	keys := getFreshData(10, 8)
	values := getFreshData(10, 8)
	root, _ := smt.Update(keys, values)
	//updatedNb := len(smt.db.updatedNodes.Items())
	//newvalues := getFreshData(10, 8)
	//smt.Update(keys, newvalues)
	//if len(smt.db.updatedNodes.Items()) != 2*updatedNb {
	//	tree.Fatal("Atomic update doesnt store all tries")
	//}

	// check keys of previous atomic update are accessible in
	// updated nodes with root.
	smt.atomicUpdate = false
	for i, key := range keys {
		value, _ := smt.get(root, key, nil, 0, smt.TrieHeight)
		if !bytes.Equal(values[i], value) {
			t.Fatal("failed to get value")
		}
	}
}

func TestSmtPublicUpdateAndGet(t *testing.T) {
	smt := NewSMT(nil, Hasher, nil, time.Minute)
	smt.CacheHeightLimit = 0
	// Add data to empty trie
	keys := getFreshData(5, 8)
	values := getFreshData(5, 8)
	root, _ := smt.Update(keys, values)
	//cacheNb := len(smt.db.liveCache)

	// Check all keys have been stored
	for i, key := range keys {
		value, _ := smt.Get(key)
		if !bytes.Equal(values[i], value) {
			t.Fatal("trie not updated")
		}
	}
	if !bytes.Equal(root, smt.Root) {
		t.Fatal("Root not stored")
	}

	newValues := getFreshData(5, 8)
	smt.Update(keys, newValues)

	// Check all keys have been modified
	for i, key := range keys {
		value, _ := smt.Get(key)
		if !bytes.Equal(newValues[i], value) {
			t.Fatal("trie not updated")
		}
	}

	newKeys := getFreshData(5, 8)
	newValues = getFreshData(5, 8)
	smt.Update(newKeys, newValues)
	for i, key := range newKeys {
		value, _ := smt.Get(key)
		if !bytes.Equal(newValues[i], value) {
			t.Fatal("trie not updated")
		}
	}
}

/*
// Because of the batching, variable sized keys are no longer available
func TestSmtDifferentKeySize(tree *testing.T) {
	keySize := 20
	smt := NewSMT(uint64(keySize), hash, nil)
	// Add data to empty trie
	keys := getFreshData(10, keySize)
	values := getFreshData(10, 8)
	smt.Update(keys, values)

	// Check all keys have been stored
	for i, key := range keys {
		value, _ := smt.Get(key)
		if !bytes.Equal(values[i], value) {
			tree.Fatal("trie not updated")
		}
	}
	newValues := getFreshData(10, 8)
	smt.Update(keys, newValues)
	// Check all keys have been modified
	for i, key := range keys {
		value, _ := smt.Get(key)
		if !bytes.Equal(newValues[i], value) {
			tree.Fatal("trie not updated")
		}
	}
	smt.Update(keys[0:1], [][]byte{DefaultLeaf})
	newValue, _ := smt.Get(keys[0])
	if len(newValue) != 0 {
		tree.Fatal("Failed to delete from trie")
	}
	newValue, _ = smt.Get(make([]byte, keySize))
	if len(newValue) != 0 {
		tree.Fatal("Failed to delete from trie")
	}
	ap, _ := smt.MerkleProof(keys[8])
	if !smt.VerifyMerkleProof(ap, keys[8], newValues[8]) {
		tree.Fatalf("failed to verify inclusion proof")
	}
}
*/

func TestSmtDelete(t *testing.T) {
	smt := NewSMT(nil, Hasher, nil, time.Minute)
	// Add data to empty trie
	keys := getFreshData(10, 8)
	values := getFreshData(10, 8)
	ch := make(chan result, 1)
	smt.update(smt.Root, keys, values, nil, 0, smt.TrieHeight, false, true, ch)
	res := <-ch
	root := res.update
	value, _ := smt.get(root, keys[0], nil, 0, smt.TrieHeight)
	if !bytes.Equal(values[0], value) {
		t.Fatal("trie not updated")
	}

	// Delete from trie
	// To delete a key, just set it's value to Default leaf hash.
	ch = make(chan result, 1)
	smt.update(root, keys[0:1], [][]byte{DefaultLeaf}, nil, 0, smt.TrieHeight, false, true, ch)
	res = <-ch
	newRoot := res.update
	newValue, _ := smt.get(newRoot, keys[0], nil, 0, smt.TrieHeight)
	if len(newValue) != 0 {
		t.Fatal("Failed to delete from trie")
	}
	// Remove deleted key from keys and check root with a clean trie.
	smt2 := NewSMT(nil, Hasher, nil, time.Minute)
	ch = make(chan result, 1)
	smt2.update(smt2.Root, keys[1:], values[1:], nil, 0, smt.TrieHeight, false, true, ch)
	res = <-ch
	cleanRoot := res.update
	if !bytes.Equal(newRoot, cleanRoot) {
		t.Fatal("roots mismatch")
	}

	//Empty the trie
	var newValues [][]byte
	for i := 0; i < 10; i++ {
		newValues = append(newValues, DefaultLeaf)
	}
	ch = make(chan result, 1)
	smt.update(root, keys, newValues, nil, 0, smt.TrieHeight, false, true, ch)
	res = <-ch
	root = res.update
	if len(root) != 0 {
		t.Fatal("empty trie root hash not correct")
	}
	// Test deleting an already empty key
	smt = NewSMT(nil, Hasher, nil, time.Minute)
	keys = getFreshData(2, 8)
	values = getFreshData(2, 8)
	root, _ = smt.Update(keys, values)
	key0 := make([]byte, 8, 8)
	key1 := make([]byte, 8, 8)
	smt.Update([][]byte{key0, key1}, [][]byte{DefaultLeaf, DefaultLeaf})
	if !bytes.Equal(root, smt.Root) {
		t.Fatal("deleting a default key shouldnt' modify the tree")
	}
}

// test updating and deleting at the same time
func TestTrieUpdateAndDelete(t *testing.T) {
	smt := NewSMT(nil, Hasher, nil, time.Minute)
	smt.CacheHeightLimit = 0
	key0 := make([]byte, 8, 8)
	values := getFreshData(1, 8)
	root, _ := smt.Update([][]byte{key0}, values)
	smt.atomicUpdate = false
	_, _, k, v, isShortcut, _ := smt.loadChildren(root, smt.TrieHeight, 0, nil)
	if !isShortcut || !bytes.Equal(k[:HashLength], key0) || !bytes.Equal(v[:HashLength], values[0]) {
		t.Fatal("leaf shortcut didn'tree move up to root")
	}

	key1 := make([]byte, 8, 8)
	// set the last bit
	bitSet(key1, 63)
	keys := [][]byte{key0, key1}
	values = [][]byte{DefaultLeaf, getFreshData(1, 8)[0]}
	smt.Update(keys, values)

	// shortcut nodes don'tree move up so size is 16+1 instead of 1.
	//x := len(smt.db.updatedNodes)
	//if x != 17 {
	//	tree.Fatalf("number of cache nodes not correct after delete: %d", x)
	//}
}

func TestSmtRaisesError(t *testing.T) {

	smt := NewSMT(nil, Hasher, nil, time.Minute)
	// Add data to empty trie
	keys := getFreshData(10, 8)
	values := getFreshData(10, 8)
	smt.Update(keys, values)
	//smt.db.liveCache = make(map[Hash][][]byte)
	smt.db.updatedNodes = ByteCache{cache: cache.NewTTL(forever, time.Minute)}
	smt.loadDefaultHashes()

	// Check errors are raised is a keys is not in cache nor db
	for _, key := range keys {
		_, err := smt.Get(key)
		if err == nil {
			t.Fatal("Error not created if database doesnt have a node")
		}
	}
	// TODO: Shouldn'tree this succeed, failing the test?
	_, err := smt.Update(keys, values)
	if err == nil {
		t.Fatal("Error not created if database doesnt have a node")
	}
}

func getFreshData(size, length int) [][]byte {
	var data [][]byte
	for i := 0; i < size; i++ {
		key := make([]byte, 8)
		_, err := rand.Read(key)
		if err != nil {
			panic(err)
		}
		data = append(data, Hasher(key)[:length])
	}
	sort.Sort(DataArray(data))
	return data
}

func benchmark10MAccounts10Ktps(smt *SMT, b *testing.B) {
	//b.ReportAllocs()
	fmt.Println("\nLoading b.N x 1000 accounts")
	for index := 0; index < b.N; index++ {
		newkeys := getFreshData(1000, 8)
		newvalues := getFreshData(1000, 8)
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
			"\ndb read : ", smt.LoadDbCounter, "    cache read : ", smt.LoadCacheCounter,
			//"\ncache size : ", len(smt.db.liveCache.Items()),
			"\nRAM : ", m.Sys/1024/1024, " MiB")
	}
}

//go test -run=xxx -bench=. -benchmem -test.benchtime=20s
func BenchmarkCacheHeightLimit233(b *testing.B) {
	smt := NewSMT(nil, Hasher, cache.NewTTL(forever, time.Minute), time.Minute)
	smt.CacheHeightLimit = 233
	benchmark10MAccounts10Ktps(smt, b)
}
func BenchmarkCacheHeightLimit238(b *testing.B) {
	smt := NewSMT(nil, Hasher, cache.NewTTL(forever, time.Minute), time.Minute)
	smt.CacheHeightLimit = 238
	benchmark10MAccounts10Ktps(smt, b)
}
func BenchmarkCacheHeightLimit245(b *testing.B) {
	smt := NewSMT(nil, Hasher, cache.NewTTL(forever, time.Minute), time.Minute)
	smt.CacheHeightLimit = 245
	benchmark10MAccounts10Ktps(smt, b)
}