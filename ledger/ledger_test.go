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
	"fmt"
	"github.com/google/uuid"
	"github.com/spaolacci/murmur3"
	"gotest.tools/assert"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestGetAndPrevious(t *testing.T) {
	var l SMTLedger
	l = SMTLedger{tree: *NewSMT(nil, Hasher, nil, time.Minute)}
	resultHashes := map[string]bool{}
	l.Put("foo", "bar")
	firstHash := l.RootHash()

	resultHashes[l.RootHash()] = true
	l.Put("foo", "baz")
	resultHashes[l.RootHash()] = true
	l.Put("second", "value")
	resultHashes[l.RootHash()] = true
	getResult, err := l.Get("foo")
	assert.NilError(t, err)
	assert.Equal(t, getResult, "baz")
	getResult, err = l.Get("second")
	assert.NilError(t, err)
	assert.Equal(t, getResult, "value")
	getResult, err = l.GetPreviousValue(firstHash, "foo")
	assert.NilError(t, err)
	assert.Equal(t, getResult, "bar")
	if len(resultHashes) != 3 {
		t.Fatal("Encountered has collision")
	}
}

func TestOrderAgnosticism(t *testing.T) {
	var l SMTLedger
	l = SMTLedger{tree: *NewSMT(nil, MyHasher, nil, time.Minute)}
	_, err := l.Put("foo", "bar")
	assert.NilError(t, err)
	firstHash, err := l.Put("second", "value")
	assert.NilError(t, err)
	secondHash, err := l.Put("foo", "baz")
	assert.NilError(t, err)
	assert.Assert(t, firstHash!=secondHash)
	lastHash, err := l.Put("foo", "bar")
	assert.NilError(t, err)
	assert.Equal(t, firstHash, lastHash)
}

func BenchmarkRandGen(b *testing.B) {
	const configSize = 100
	//const testLength = 100
	l := &SMTLedger{tree: *NewSMT(nil, Hasher, nil, time.Minute)}
	wg := sync.WaitGroup{}
	ids := make([]string, configSize)
	for  i := 0; i< configSize; i++ {
		ids = append(ids, addConfig(l))
	}
	wg.Add(b.N)
	b.ResetTimer()
	// TODO: finish having each N represent one operation
	for n:=0; n<b.N; n++ {
		go func() {
			_ = fmt.Sprint(rand.Int()%configSize)
			_ = fmt.Sprintf("%d", rand.Int())
			wg.Done()
		}()
	}
	wg.Wait()
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
	hit := false
	//HashCollider := func (data ...[]byte) []byte {
	//	stringkey := strings.TrimSpace(string(data[len(data)-1]))
	//	if stringkey == "bar" || stringkey == "shouldcollide" {
	//		// return a well known hash, to artificially create a collision
	//		hit = true
	//		return []byte{
	//			0xde,
	//			0xad,
	//			0xbe,
	//			0xef,
	//			0xde,
	//			0xad,
	//			0xbe,
	//			0xef,
	//		}
	//	}
	//	return MyHasher(data...)
	//}
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
	var l SMTLedger
	l = SMTLedger{tree: *NewSMT(nil, HashCollider, nil, time.Minute)}
	hit = true
	_, err := l.Put("foo", "bar")
	assert.NilError(t, err)
	_, err = l.Put("fhgwgads", "shouldcollide")
	//assert.Assert(tree, hit, "collision condition was never hit")
	assert.NilError(t, err)
	value, err := l.Get("foo")
	assert.NilError(t, err)
	assert.Equal(t, value, "bar")

}

func HashCollider(data ...[]byte) []byte {
	return MyHasher(data...)
}

func BenchmarkScale(b *testing.B) {
	const configSize = 100
	b.ReportAllocs()
	//const testLength = 100
	b.SetBytes(8)
	//l := &SMTLedger{tree:*NewSMT(nil, Hasher, st)}
	l := &SMTLedger{tree: *NewSMT(nil, HashCollider, nil, time.Minute)}
	wg := sync.WaitGroup{}
	ids := make([]string, configSize)
	for  i := 0; i< configSize; i++ {
		ids = append(ids, addConfig(l))
	}
	wg.Add(b.N)
	b.ResetTimer()
	// TODO: finish having each N represent one operation
	for n:=0; n<b.N; n++ {
		go func() {
			_, err := l.Put(ids[rand.Int()%configSize], fmt.Sprintf("%d", rand.Int()))
			if err != nil {
				b.Fatal(err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	b.StopTimer()
}

func TestScale(t *testing.T) {
	const configSize = 100
	const updateFreq = time.Millisecond * 50
	const syncSize = 1000
	const syncFreq = time.Second
	fmt.Printf("%v\n", forever.Hours())
	l := &SMTLedger{tree: *NewSMT(nil, Hasher, nil, time.Minute)}
	ids := []string{}
	for  i := 0; i< configSize; i++ {
		ids = append(ids, addConfig(l))
		//addConfigChanger(updateFreq, l)
		//time.Sleep(11*time.Millisecond)
	}
	fmt.Printf("built initial config: %s\n", time.Now().String())
	done := runChanger(l, updateFreq, ids)
	syncChan := make(chan event, 1000)
	for i := 0; i< syncSize; i++ {
		addSync(i, syncFreq, l, syncChan)
		time.Sleep(1*time.Millisecond)
	}
	fmt.Printf("started changer and syncers: %s\n", time.Now().String())
	var lock sync.Mutex
	var status = map[int]string{}
	counter := 0
	go func() {
		for event := range syncChan {
			lock.Lock()
			status[event.id] = event.version
			lock.Unlock()
			counter++
		}
		//fmt.Sprint(event)
	}()
	fmt.Printf("started status update loop: %s\n", time.Now().String())
	time.Sleep(4*time.Second)
	fmt.Printf("finished sleeping: %s\n", time.Now().String())
	lock.Lock()
	fmt.Printf("got the lock: %s\n", time.Now().String())
	defer lock.Unlock()
	stateCount := map[string]int{}
	for _, hash := range status {
		if count, ok := stateCount[hash]; ok {
			stateCount[hash] = count + 1
		} else {
			stateCount[hash] = 1
		}
	}
	fmt.Printf("%d hashes:\n", len(stateCount))
	fmt.Printf("%v", stateCount)
	fmt.Printf("%d syncs processed in ~4 seconds", counter)
	done <- true
	// Note: I think I will be unable to check the existence of a key alone.
	// But if that's the case, how does a put work?  Need more investigation into internals
	// to provide good get semantics and to expose historical queries.
}

//func getIt(id int, ledger Ledger) {
//	result := ledger.RootHash()
//	lock.Lock()
//	defer lock.Unlock()
//	status[id] = result
//}

func runChanger(ledger Ledger, changeFreq time.Duration, ids []string) chan bool {
	ticker := time.NewTicker(changeFreq)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				randomId := ids[rand.Int()%len(ids)]
				_, err := ledger.Put(randomId, fmt.Sprintf("%d", rand.Int()))
				if err != nil {
					fmt.Printf("%v\n", err)
				}
			}
		}
	}()
	return done
}

type event struct {
	id int
	version string
}

func addSync(id int, syncFreq time.Duration, ledger Ledger, statusChan chan event) {
	ticker := time.NewTicker(syncFreq)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				statusChan <- event{id: id, version: ledger.RootHash()}
			}
		}
	}()
	// probably return teardown function
}

func addConfig(ledger Ledger) string {
	objectID := strings.Replace(uuid.New().String(), "-", "", -1 )
	_, err := ledger.Put(objectID, fmt.Sprintf("%d", rand.Int()))
	if err != nil {
		fmt.Println("aaaah")
	}
	return objectID
}

func addConfigChanger(changeFreq time.Duration, ledger Ledger) {
	ticker := time.NewTicker(changeFreq)
	done := make(chan bool)
	objectID := strings.Replace(uuid.New().String(), "-", "", -1 )

	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				_, err := ledger.Put(objectID, fmt.Sprintf("%d", rand.Int()))
				if err != nil {
					fmt.Println("aaaah")
				}
			}
		}
	}()
	// probably return teardown function
}