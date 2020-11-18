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
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/spaolacci/murmur3"
	"golang.org/x/sync/errgroup"
	"gotest.tools/assert"
)

func TestLongKeys(t *testing.T) {
	longKey := "virtual-service/frontend/default"
	l := smtLedger{tree: newSMT(hasher, nil, time.Minute), history: NewHistory()}
	_, err := l.Put(longKey+"1", "1")
	assert.NilError(t, err)
	_, err = l.Put(longKey+"2", "2")
	assert.NilError(t, err)
	res, err := l.Get(longKey + "1")
	assert.NilError(t, err)
	assert.Equal(t, res, "1")
	res, err = l.Get(longKey + "2")
	assert.NilError(t, err)
	assert.Equal(t, res, "2")
	res, err = l.Get(longKey)
	assert.NilError(t, err)
	assert.Equal(t, res, "")
}

func TestGetAndPrevious(t *testing.T) {
	l := smtLedger{tree: newSMT(hasher, nil, time.Minute), history: NewHistory()}
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
	l := smtLedger{tree: newSMT(MyHasher, nil, time.Minute), history: NewHistory()}
	_, err := l.Put("foo", "bar")
	assert.NilError(t, err)
	firstHash, err := l.Put("second", "value")
	assert.NilError(t, err)
	secondHash, err := l.Put("foo", "baz")
	assert.NilError(t, err)
	assert.Assert(t, firstHash != secondHash)
	lastHash, err := l.Put("foo", "bar")
	assert.NilError(t, err)
	assert.Equal(t, firstHash, lastHash)
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
	l := smtLedger{tree: newSMT(HashCollider, nil, time.Minute), history: NewHistory()}
	hit = true
	_, err := l.Put("foo", "bar")
	assert.NilError(t, err)
	_, err = l.Put("fhgwgads", "shouldcollide")
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
	b.SetBytes(8)
	l := Make(time.Minute)
	var eg errgroup.Group
	ids := make([]string, configSize)
	for i := 0; i < configSize; i++ {
		ids = append(ids, addConfig(l, b))
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
func addConfig(ledger Ledger, b *testing.B) string {
	objectID := strings.Replace(uuid.New().String(), "-", "", -1)
	_, err := ledger.Put(objectID, fmt.Sprintf("%d", rand.Int()))
	assert.NilError(b, err)
	return objectID
}

func TestParallel(t *testing.T) {
	l := Make(time.Minute)
	size := 5000
	k1, v1 := getFreshEntries(size)
	k2, v2 := getFreshEntries(size)
	for i := 0; i < size; i++ {
		key := k1[i]
		value := v1[i]
		go func() {
			_, err := l.Put(key, value)
			assert.NilError(t, err)
		}()
	}
	for i := 0; i < size; i++ {
		key := k2[i]
		value := v2[i]
		del := k1[i]
		go func() {
			_, err := l.Put(key, value)
			assert.NilError(t, err)
			err = l.Delete(del)
			assert.NilError(t, err)
		}()
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
	l := Make(time.Minute)
	_, err := l.Put("One", "1")
	assert.NilError(t, err)
	_, err = l.Put("Two", "2")
	assert.NilError(t, err)
	_, err = l.Put("Three", "3")
	assert.NilError(t, err)
	_, err = l.Put("Four", "4")
	assert.NilError(t, err)
	_, err = l.Put("Five", "5")
	assert.NilError(t, err)
	six, err := l.Put("Six", "6")
	assert.NilError(t, err)
	seven, err := l.Put("Seven", "7")
	assert.NilError(t, err)
	err = l.Delete("Six")
	assert.NilError(t, err)
	_, err = l.Put("Eight", "8")
	assert.NilError(t, err)
	_, err = l.Put("Nine", "9")
	assert.NilError(t, err)
	_, err = l.Put("Ten", "10")
	assert.NilError(t, err)
	err = l.EraseRootHash(seven)
	assert.NilError(t, err)
	val, err := l.GetPreviousValue(six, "Six")
	assert.NilError(t, err)
	assert.Equal(t, val, "6")
	_, err = l.GetPreviousValue(seven, "Six")
	assert.ErrorContains(t, err, "root node")
	err = l.EraseRootHash(six)
	assert.NilError(t, err)
	_, err = l.GetPreviousValue(six, "Six")
	assert.ErrorContains(t, err, "root node")
	err = l.EraseRootHash(seven)
	assert.ErrorContains(t, err, "rootHash")
}
