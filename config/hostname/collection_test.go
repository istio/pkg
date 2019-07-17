// Copyright 2017 Istio Authors
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

package hostname_test

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"istio.io/pkg/config/hostname"
)

func TestNewCollection(t *testing.T) {
	cases := []struct {
		name     string
		input    []string
		expected hostname.Collection
	}{
		{
			name:     "nil yields empty collection",
			input:    nil,
			expected: make(hostname.Collection, 0),
		},
		{
			name:     "values copied",
			input:    []string{"a", "b"},
			expected: hostname.Collection{hostname.Instance("a"), hostname.Instance("b")},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if actual := hostname.NewCollection(c.input); !reflect.DeepEqual(actual, c.expected) {
				t.Fatalf("expected %v, got %v", c.expected, actual)
			}
		})
	}
}

func TestCollectionIntersection(t *testing.T) {
	tests := []struct {
		a, b, intersection hostname.Collection
	}{
		{
			hostname.Collection{"foo,com"},
			hostname.Collection{"bar.com"},
			hostname.Collection{},
		},
		{
			hostname.Collection{"foo.com", "bar.com"},
			hostname.Collection{"bar.com"},
			hostname.Collection{"bar.com"},
		},
		{
			hostname.Collection{"foo.com", "bar.com"},
			hostname.Collection{"*.com"},
			hostname.Collection{"foo.com", "bar.com"},
		},
		{
			hostname.Collection{"*.com"},
			hostname.Collection{"foo.com", "bar.com"},
			hostname.Collection{"foo.com", "bar.com"},
		},
		{
			hostname.Collection{"foo.com", "*.net"},
			hostname.Collection{"*.com", "bar.net"},
			hostname.Collection{"foo.com", "bar.net"},
		},
		{
			hostname.Collection{"foo.com", "*.net"},
			hostname.Collection{"*.bar.net"},
			hostname.Collection{"*.bar.net"},
		},
		{
			hostname.Collection{"foo.com", "bar.net"},
			hostname.Collection{"*"},
			hostname.Collection{"foo.com", "bar.net"},
		},
		{
			hostname.Collection{"foo.com"},
			hostname.Collection{},
			hostname.Collection{},
		},
		{
			hostname.Collection{},
			hostname.Collection{"bar.com"},
			hostname.Collection{},
		},
		{
			hostname.Collection{"*", "foo.com"},
			hostname.Collection{"foo.com"},
			hostname.Collection{"foo.com"},
		},
	}

	for idx, tt := range tests {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			result := tt.a.Intersection(tt.b)
			if !reflect.DeepEqual(result, tt.intersection) {
				t.Fatalf("%v.Intersection(%v) = %v, want %v", tt.a, tt.b, result, tt.intersection)
			}
		})
	}
}

func TestCollectionForNamespace(t *testing.T) {
	tests := []struct {
		hosts     []string
		namespace string
		want      hostname.Collection
	}{
		{
			[]string{"ns1/foo.com", "ns2/bar.com"},
			"ns1",
			hostname.Collection{"foo.com"},
		},
		{
			[]string{"ns1/foo.com", "ns2/bar.com"},
			"ns3",
			hostname.Collection{},
		},
		{
			[]string{"ns1/foo.com", "*/bar.com"},
			"ns1",
			hostname.Collection{"foo.com", "bar.com"},
		},
		{
			[]string{"ns1/foo.com", "*/bar.com"},
			"ns3",
			hostname.Collection{"bar.com"},
		},
		{
			[]string{"foo.com", "ns2/bar.com"},
			"ns2",
			hostname.Collection{"foo.com", "bar.com"},
		},
		{
			[]string{"foo.com", "ns2/bar.com"},
			"ns3",
			hostname.Collection{"foo.com"},
		},
	}

	for idx, tt := range tests {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			result := hostname.CollectionForNamespace(tt.hosts, tt.namespace)
			if !reflect.DeepEqual(result, tt.want) {
				t.Fatalf("CollectionForNamespace(%v, %v) = %v, want %v", tt.hosts, tt.namespace, result, tt.want)
			}
		})
	}
}

func TestCollectionSortOrder(t *testing.T) {
	tests := []struct {
		in, want hostname.Collection
	}{
		// Prove we sort alphabetically:
		{
			hostname.Collection{"b", "a"},
			hostname.Collection{"a", "b"},
		},
		{
			hostname.Collection{"bb", "cc", "aa"},
			hostname.Collection{"aa", "bb", "cc"},
		},
		// Prove we sort longest first, alphabetically:
		{
			hostname.Collection{"b", "a", "aa"},
			hostname.Collection{"aa", "a", "b"},
		},
		{
			hostname.Collection{"foo.com", "bar.com", "foo.bar.com"},
			hostname.Collection{"foo.bar.com", "bar.com", "foo.com"},
		},
		// We sort wildcards last, always
		{
			hostname.Collection{"a", "*", "z"},
			hostname.Collection{"a", "z", "*"},
		},
		{
			hostname.Collection{"foo.com", "bar.com", "*.com"},
			hostname.Collection{"bar.com", "foo.com", "*.com"},
		},
		{
			hostname.Collection{"foo.com", "bar.com", "*.com", "*.foo.com", "*", "baz.bar.com"},
			hostname.Collection{"baz.bar.com", "bar.com", "foo.com", "*.foo.com", "*.com", "*"},
		},
	}

	for idx, tt := range tests {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			// Save a copy to report errors with
			tmp := make(hostname.Collection, len(tt.in))
			copy(tmp, tt.in)

			sort.Sort(tt.in)
			if !reflect.DeepEqual(tt.in, tt.want) {
				t.Fatalf("sort.Sort(%v) = %v, want %v", tmp, tt.in, tt.want)
			}
		})
	}
}

func BenchmarkSort(b *testing.B) {
	unsorted := hostname.Collection{"foo.com", "bar.com", "*.com", "*.foo.com", "*", "baz.bar.com"}

	for n := 0; n < b.N; n++ {
		given := make(hostname.Collection, len(unsorted))
		copy(given, unsorted)
		sort.Sort(given)
	}
}
