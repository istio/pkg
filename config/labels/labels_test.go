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

package labels_test

import (
	"testing"

	"istio.io/pkg/config/labels"
)

func TestSubsetOf(t *testing.T) {
	cases := []struct {
		name     string
		left     labels.Instance
		right    labels.Instance
		expected bool
	}{
		{
			name:     "nil is subset",
			left:     nil,
			right:    labels.Instance{"app": "a"},
			expected: true,
		},
		{
			name:     "not subset of nil",
			left:     labels.Instance{"app": "a"},
			right:    nil,
			expected: false,
		},
		{
			name:     "valid subset",
			left:     labels.Instance{"app": "a"},
			right:    labels.Instance{"app": "a", "prod": "env"},
			expected: true,
		},
		{
			name:     "not subset",
			left:     labels.Instance{"app": "a", "prod": "env"},
			right:    labels.Instance{"app": "a"},
			expected: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if actual := c.left.SubsetOf(c.right); actual != c.expected {
				t.Fatalf("expected %t got %t", c.expected, actual)
			}
		})
	}
}

func TestEquals(t *testing.T) {
	cases := []struct {
		a, b labels.Instance
		want bool
	}{
		{
			a: nil,
			b: labels.Instance{"a": "b"},
		},
		{
			a: labels.Instance{"a": "b"},
			b: nil,
		},
		{
			a:    labels.Instance{"a": "b"},
			b:    labels.Instance{"a": "b"},
			want: true,
		},
	}
	for _, c := range cases {
		if got := c.a.Equals(c.b); got != c.want {
			t.Errorf("Failed: got eq=%v want=%v for %q ?= %q", got, c.want, c.a, c.b)
		}
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name   string
		labels labels.Instance
		valid  bool
	}{
		{
			name:  "NoLabels",
			valid: true,
		},
		{
			name:   "BadKeyAndValue",
			labels: labels.Instance{"^": "^"},
		},
		{
			name:   "Good",
			labels: labels.Instance{"key": "value"},
			valid:  true,
		},
		{
			name:   "EmptyValue",
			labels: labels.Instance{"key": ""},
			valid:  true,
		},
		{
			name:   "EmptyKey",
			labels: labels.Instance{"": "value"},
		},
		{
			name:   "BadKey1",
			labels: labels.Instance{".key": "value"},
		},
		{
			name:   "BadKey2",
			labels: labels.Instance{"key_": "value"},
		},
		{
			name:   "BadKey3",
			labels: labels.Instance{"key$": "value"},
		},
		{
			name:   "BadValue1",
			labels: labels.Instance{"key": ".value"},
		},
		{
			name:   "BadValue2",
			labels: labels.Instance{"key": "value_"},
		},
		{
			name:   "BadValue3",
			labels: labels.Instance{"key": "value$"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.labels.Validate(); (got == nil) != c.valid {
				t.Fatalf("got valid=%v but wanted valid=%v: %v", got == nil, c.valid, got)
			}
		})
	}
}
