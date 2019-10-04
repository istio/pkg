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
	"istio.io/pkg/cache"
	"time"

	//"fmt"
	"sync"
)

type CacheDB struct {
	// liveCache contains the first levels of the trie (nodes that have 2 non default children)
	//liveCache CacheWrapper
	// liveMux is a lock for liveCache
	liveMux sync.RWMutex
	// updatedNodes that have will be flushed to disk
	updatedNodes ByteCache
	// updatedMux is a lock for updatedNodes
	updatedMux sync.RWMutex
	// nodesToRevert will be deleted from db
	nodesToRevert [][]byte
	// revertMux is a lock for updatedNodes
	revertMux sync.RWMutex
	// lock for CacheDB
	lock sync.RWMutex
}

// ByteCache implements a modified ExpiringCache interface, returning byte arrays
// for ease of integration with SMT calls.
type ByteCache struct {
	cache cache.ExpiringCache
}
// Set inserts an entry in the cache. This will replace any entry with
// the same key that is already in the cache. The entry may be automatically
// expunged from the cache at some point, depending on the eviction policies
// of the cache and the options specified when the cache was created.
func (b *ByteCache) Set(key Hash, value [][]byte) {
	b.cache.Set(key, value)
}

// Get retrieves the value associated with the supplied key if the key
// is present in the cache.
func (b *ByteCache) Get(key Hash) (value [][]byte, ok bool){
	ivalue, ok := b.cache.Get(key)
	if ok {
		value, _ = ivalue.([][]byte)
	}
	return
}

// SetWithExpiration inserts an entry in the cache with a requested expiration time.
// This will replace any entry with the same key that is already in the cache.
// The entry will be automatically expunged from the cache at or slightly after the
// requested expiration time.
func (b *ByteCache) SetWithExpiration(key Hash, value [][]byte, expiration time.Duration) {
	b.cache.SetWithExpiration(key, value, expiration)
}


func (db *CacheDB) serializeBatch(batch [][]byte) []byte {
	serialized := make([]byte, 4) //, 30*33)
	if batch[0][0] == 1 {
		// the batch node is a shortcut
		bitSet(serialized, 31)
	}
	for i := 1; i < 31; i++ {
		if len(batch[i]) != 0 {
			bitSet(serialized, i-1)
			serialized = append(serialized, batch[i]...)
		}
	}
	return serialized
}
