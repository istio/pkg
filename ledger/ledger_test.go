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
	"encoding/base64"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/spaolacci/murmur3"
	"golang.org/x/sync/errgroup"

	"istio.io/pkg/cache"
)

type testLedger struct {
	s *smtLedger
	g *GomegaWithT
}

type validTreeMatcher struct{}

func (matcher *validTreeMatcher) FailureMessage(actual interface{}) (message string) {
	panic("implement me")
}

func (matcher *validTreeMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	panic("implement me")
}

func (matcher *validTreeMatcher) Match(actual interface{}) (success bool, err error) {
	var tree *smt
	if sl, ok := actual.(*smtLedger); !ok {
		if tree, ok = actual.(*smt); !ok {
			return false, fmt.Errorf("validTreeMatch matcher expects an smtLedger or smt")
		}
	} else {
		tree = sl.tree
	}
	return validate(tree)
}

func beValidTree() types.GomegaMatcher {
	return &validTreeMatcher{}
}

func (tl *testLedger) Delete(key string) (string, error) {
	res, err := tl.s.Delete(key)
	tl.g.Expect(tl.s).To(beValidTree())
	return res, err
}

func (tl *testLedger) Get(key string) (string, error) {
	return tl.s.Get(key)
}

func (tl *testLedger) RootHash() string {
	return tl.s.RootHash()
}

func (tl *testLedger) GetPreviousValue(previousRootHash, key string) (result string, err error) {
	return tl.s.GetPreviousValue(previousRootHash, key)
}

func (tl *testLedger) EraseRootHash(rootHash string) error {
	err := tl.s.EraseRootHash(rootHash)
	tl.g.Expect(tl.s).To(beValidTree())
	return err
}

func (tl *testLedger) Stats() cache.Stats {
	return tl.s.Stats()
}

func (tl *testLedger) GetAll() (map[string]string, error) {
	return tl.s.GetAll()
}

func (tl *testLedger) GetAllPrevious(s string) (map[string]string, error) {
	return tl.s.GetAllPrevious(s)
}

func (tl *testLedger) Put(key, value string) (result string, err error) {
	result, err = tl.s.Put(key, value)
	tl.g.Expect(tl.s).To(beValidTree())
	return
}

func MakeTest(g *GomegaWithT) Ledger {
	s := makeOld(1).(*smtLedger)
	RegisterFailHandler(func(message string, callerSkip ...int) {
		fmt.Printf("Failure detected.  Graphviz of failing ledger:\n%s", s.tree.DumpToDOT())
	})
	return &testLedger{
		s: s,
		g: g,
	}
}

func TestLongKeys(t *testing.T) {
	g := NewGomegaWithT(t)
	longKey := "virtual-service/frontend/default"
	l := MakeTest(g)
	_, err := l.Put(longKey+"1", "1")
	g.Expect(err).NotTo(HaveOccurred())
	_, err = l.Put(longKey+"2", "2")
	g.Expect(err).NotTo(HaveOccurred())
	res, err := l.Get(longKey + "1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(res).To(Equal("1"))
	res, err = l.Get(longKey + "2")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(res).To(Equal("2"))
	res, err = l.Get(longKey)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(res).To(Equal(""))

}

func TestGetAndPrevious(t *testing.T) {
	g := NewGomegaWithT(t)
	l := MakeTest(g)
	resultHashes := map[string]bool{}
	l.Put("foo", "bar")
	firstHash := l.RootHash()

	resultHashes[l.RootHash()] = true
	l.Put("foo", "baz")
	resultHashes[l.RootHash()] = true
	l.Put("second", "value")
	resultHashes[l.RootHash()] = true
	getResult, err := l.Get("foo")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(getResult).To(Equal("baz"))
	getResult, err = l.Get("second")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(getResult).To(Equal("value"))
	getResult, err = l.GetPreviousValue(firstHash, "foo")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(getResult).To(Equal("bar"))
	g.Expect(resultHashes).To(HaveLen(3))
}

func TestOrderAgnosticism(t *testing.T) {
	g := NewGomegaWithT(t)
	l := MakeTest(g)
	_, err := l.Put("foo", "bar")
	g.Expect(err).NotTo(HaveOccurred())
	firstHash, err := l.Put("second", "value")
	g.Expect(err).NotTo(HaveOccurred())
	secondHash, err := l.Put("foo", "baz")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(firstHash).NotTo(Equal(secondHash))
	lastHash, err := l.Put("foo", "bar")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(firstHash).To(Equal(lastHash))
}

func MyHasher(data ...[]byte) (result []byte) {
	var hasher = murmur3.New64()
	for i := 0; i < len(data); i++ {
		hasher.Write(data[i])
	}
	result = hasher.Sum(nil)
	hasher.Reset()
	return
}

func TestCollision(t *testing.T) {
	g := NewGomegaWithT(t)
	hit := false
	HashCollider := func(data ...[]byte) []byte {
		if hit {
			return []byte{
				0xde,
				0xad,
				0xbe,
				0xef,
				0xde,
				0xad,
				0xbe,
				0xef,
			}
		}
		return MyHasher(data...)
	}
	l := MakeTest(g)
	l.(*testLedger).s.tree.hash = HashCollider
	hit = true
	_, err := l.Put("foo", "bar")
	g.Expect(err).NotTo(HaveOccurred())
	_, err = l.Put("fhgwgads", "shouldcollide")
	g.Expect(err).NotTo(HaveOccurred())
	value, err := l.Get("foo")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(value).To(Equal("bar"))

}

func HashCollider(data ...[]byte) []byte {
	return MyHasher(data...)
}

func BenchmarkScale(b *testing.B) {
	g := NewGomegaWithT(b)
	const configSize = 100
	b.ReportAllocs()
	b.SetBytes(8)
	l := makeOld(1)
	var eg errgroup.Group
	ids := make([]string, configSize)
	for i := 0; i < configSize; i++ {
		ids = append(ids, addConfig(l, g))
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		eg.Go(func() error {
			_, err := l.Put(ids[rand.Int()%configSize], strconv.Itoa(rand.Int()))
			_ = l.RootHash()
			return err
		})
	}
	if err := eg.Wait(); err != nil {
		b.Fatalf("An error occurred putting new data on the ledger: %v", err)
	}
	b.StopTimer()
}
func addConfig(ledger Ledger, g *GomegaWithT) string {
	objectID := strings.Replace(uuid.New().String(), "-", "", -1)
	_, err := ledger.Put(objectID, fmt.Sprintf("%d", rand.Int()))
	g.Expect(err).NotTo(HaveOccurred())
	return objectID
}

func TestParallel(t *testing.T) {
	g := NewGomegaWithT(t)
	l := MakeTest(g)
	size := 100
	k1, v1 := getFreshEntries(size)
	k2, v2 := getFreshEntries(size)
	versions := make(chan string, 1)
	oldversions := make(chan string, size*3)
	// write version to old version once it's non-current
	go func() {
		var prev string
		for v := range versions {
			if len(prev) > 0 {
				oldversions <- prev
			}
			prev = v
		}
		close(oldversions)
	}()
	wg := sync.WaitGroup{}
	wg.Add(size)
	for i := 0; i < size; i++ {
		key := k1[i]
		value := v1[i]
		go func() {
			v, err := l.Put(key, value)
			versions <- v
			g.Expect(err).NotTo(HaveOccurred())
			wg.Done()
		}()
	}
	wg.Wait()
	wg = sync.WaitGroup{}
	wg.Add(size)
	for i := 0; i < size; i++ {
		key := k2[i]
		value := v2[i]
		del := k1[i]
		go func() {
			defer wg.Done()
			_, err := l.Delete(del)
			g.Expect(err).NotTo(HaveOccurred())
			x, err := l.Put(key, value)
			g.Expect(err).NotTo(HaveOccurred())
			versions <- x
		}()
	}
	// when the above loop completes, close the channels
	go func() {
		wg.Wait()
		close(versions)
	}()
	wg2 := sync.WaitGroup{}
	wg2.Add(size*2 - 1)
	for v := range oldversions {
		go func(b string) {
			if rand.Intn(10) == 1 {
				err := l.EraseRootHash(b)
				g.Expect(err).NotTo(HaveOccurred())
			} else {
				_, err := l.GetAllPrevious(b)
				g.Expect(err).NotTo(HaveOccurred())
			}
			wg2.Done()
		}(v)
	}
	wg2.Wait()
	all, err := l.GetAll()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(all).To(HaveLen(size))
	for i := range k2 {
		g.Expect(all).To(HaveKeyWithValue(k2[i], v2[i]))
	}
}

func getFreshEntries(size int) (keys []string, values []string) {
	length := 8
	for i := 0; i < size; i++ {
		r := make([]byte, 8)
		_, err := rand.Read(r)
		if err != nil {
			panic(err)
		}
		keys = append(keys, base64.StdEncoding.EncodeToString(r[:length]))
		_, err = rand.Read(r)
		if err != nil {
			panic(err)
		}
		values = append(values, base64.StdEncoding.EncodeToString(r[:length]))
	}
	return
}

func TestEraseRootHash(t *testing.T) {
	g := NewGomegaWithT(t)
	l := MakeTest(g)
	_, err := l.Put("One", "1")
	g.Expect(err).NotTo(HaveOccurred())
	_, err = l.Put("Two", "2")
	g.Expect(err).NotTo(HaveOccurred())
	_, err = l.Put("Three", "3")
	g.Expect(err).NotTo(HaveOccurred())
	_, err = l.Put("Four", "4")
	g.Expect(err).NotTo(HaveOccurred())
	_, err = l.Put("Five", "5")
	g.Expect(err).NotTo(HaveOccurred())
	six, err := l.Put("Six", "6")
	g.Expect(err).NotTo(HaveOccurred())
	seven, err := l.Put("Seven", "7")
	g.Expect(err).NotTo(HaveOccurred())
	_, err = l.Delete("Six")
	g.Expect(err).NotTo(HaveOccurred())
	_, err = l.Delete("Six")
	g.Expect(err).NotTo(HaveOccurred())
	_, err = l.Put("Eight", "8")
	g.Expect(err).NotTo(HaveOccurred())
	_, err = l.Put("Nine", "9")
	g.Expect(err).NotTo(HaveOccurred())
	_, err = l.Put("Ten", "10")
	g.Expect(err).NotTo(HaveOccurred())
	err = l.EraseRootHash(seven)
	g.Expect(err).NotTo(HaveOccurred())
	val, err := l.GetPreviousValue(six, "Six")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(val).To(Equal("6"))
	_, err = l.GetPreviousValue(seven, "Six")
	g.Expect(err).To(MatchError(ContainSubstring("root node")))
	err = l.EraseRootHash(six)
	g.Expect(err).NotTo(HaveOccurred())
	_, err = l.GetPreviousValue(six, "Six")
	g.Expect(err).To(MatchError(ContainSubstring("root node")))
	err = l.EraseRootHash(seven)
	g.Expect(err).To(MatchError(ContainSubstring("rootHash")))
	// cache misses now occur on every non-duplicat write, so they are less meaningful...
	all, err := l.GetAll()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(all).To(HaveKeyWithValue("One", "1"))
}
