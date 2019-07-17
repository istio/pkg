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

func TestGetLocality(t *testing.T) {
	defaultValue := "defaultValue"
	cases := []struct {
		name     string
		labels   labels.Instance
		expected string
	}{
		{
			name:     "nil labels",
			labels:   nil,
			expected: defaultValue,
		},
		{
			name:     "empty labels",
			labels:   labels.Instance{},
			expected: defaultValue,
		},
		{
			name: "empty locality",
			labels: labels.Instance{
				labels.IstioLocality: "",
			},
			expected: defaultValue,
		},
		{
			name: "value unaltered",
			labels: labels.Instance{
				labels.IstioLocality: "region/zone/subzone-2",
			},
			expected: "region/zone/subzone-2",
		},
		{
			name: "k8s separator replaced by slash",
			labels: labels.Instance{
				labels.IstioLocality: "region.zone.subzone-2",
			},
			expected: "region/zone/subzone-2",
		},
		{
			name: "label with both k8s label separators and slashes",
			labels: labels.Instance{
				labels.IstioLocality: "region/zone/subzone.2",
			},
			expected: "region/zone/subzone.2",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := labels.GetLocalityOrDefault(defaultValue, testCase.labels)
			if got != testCase.expected {
				t.Errorf("expected %s, but got %s", testCase.expected, got)
			}
		})
	}
}
