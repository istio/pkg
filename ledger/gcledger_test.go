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
	"testing"

	. "github.com/onsi/gomega"
)

func TestGC(t *testing.T) {
	g := NewGomegaWithT(t)
	l := Make().(*gcledger)
	size := 20
	k1, v1 := getFreshEntries(size)
	k2, v2 := getFreshEntries(size)
	wg := sync.WaitGroup{}
	wg.Add(size)
	for i := 0; i < size; i++ {
		key := k1[i]
		value := v1[i]
		go func() {
			defer wg.Done()
			_, err := l.Put(key, value)
			g.Expect(err).NotTo(HaveOccurred())
		}()
	}
	wg.Wait()
	g.Expect(l.inner.history.Len()).To(Equal(1))
	_ = l.RootHash()
	wg = sync.WaitGroup{}
	wg.Add(size)
	for i := 0; i < size; i++ {
		key := k2[i]
		value := v2[i]
		go func() {
			defer wg.Done()
			_, err := l.Put(key, value)
			g.Expect(err).NotTo(HaveOccurred())
		}()
	}
	wg.Wait()
	g.Expect(l.inner.history.Len()).To(Equal(2))
	all, err := l.GetAll()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(all).To(HaveLen(size * 2))
	for i := range k2 {
		g.Expect(all).To(HaveKeyWithValue(k2[i], v2[i]))
	}
}
