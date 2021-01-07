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
	"sync"

	"istio.io/pkg/cache"
)

// gcledger automatically erases any versions which did not have RootHash() called on them.
// this prevents memory leaks when a given version was never used for communication with envoy.
type gcledger struct {
	inner           *smtLedger
	lock            sync.RWMutex
	currentHashUsed bool
}

func (g *gcledger) Put(key, value string) (result string, err error) {
	g.lock.Lock()
	defer g.lock.Unlock()
	prior := g.inner.RootHash()
	result, err = g.inner.Put(key, value)
	if err == nil {
		err = g.hanldemutation(prior)
	}
	return
}

func (g *gcledger) Delete(key string) (result string, err error) {
	g.lock.Lock()
	defer g.lock.Unlock()
	prior := g.inner.RootHash()
	result, err = g.inner.Delete(key)
	if err == nil {
		err = g.hanldemutation(prior)
	}
	return
}

func (g *gcledger) Get(key string) (result string, err error) {
	g.lock.RLock()
	defer g.lock.RUnlock()
	return g.inner.Get(key)
}

func (g *gcledger) RootHash() string {
	g.lock.Lock()
	defer g.lock.Unlock()
	g.currentHashUsed = true
	return g.inner.RootHash()
}

func (g *gcledger) GetPreviousValue(previousRootHash, key string) (result string, err error) {
	g.lock.RLock()
	defer g.lock.RUnlock()
	return g.inner.GetPreviousValue(previousRootHash, key)
}

func (g *gcledger) EraseRootHash(rootHash string) error {
	g.lock.Lock()
	defer g.lock.Unlock()
	return g.inner.EraseRootHash(rootHash)
}

func (g *gcledger) Stats() cache.Stats {
	return g.inner.Stats()
}

func (g *gcledger) GetAll() (map[string]string, error) {
	g.lock.RLock()
	defer g.lock.RUnlock()
	return g.inner.GetAll()
}

func (g *gcledger) GetAllPrevious(s string) (map[string]string, error) {
	g.lock.RLock()
	defer g.lock.RUnlock()
	return g.inner.GetAllPrevious(s)
}

func (g *gcledger) hanldemutation(priorRoot string) (err error) {
	if len(priorRoot) == 0 {
		return
	}
	if !g.currentHashUsed {
		err = g.inner.EraseRootHash(priorRoot)
	}
	g.currentHashUsed = false
	return
}
