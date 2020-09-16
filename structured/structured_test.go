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

package structured

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSerialize(t *testing.T) {
	tests := []struct {
		desc   string
		in     *Error
		prefix string
	}{
		{
			desc: "nil",
		},
		{
			desc: "empty",
			in:   &Error{},
		},
		{
			desc: "empty fields",
			in: &Error{
				MoreInfo:    "",
				Impact:      "",
				Action:      "",
				LikelyCause: "",
			},
		},
		{
			desc:   "all fields",
			prefix: "prefix: ",
			in: &Error{
				MoreInfo:    "MoreInfo",
				Impact:      "Impact",
				Action:      "Action",
				LikelyCause: "LikelyCause",
				Err:         errors.New("err"),
			},
		},
		{
			desc:   "some fields",
			prefix: "prefix: ",
			in: &Error{
				MoreInfo:    "MoreInfo",
				LikelyCause: "LikelyCause",
				Err:         errors.New("err"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			prefix, se := Parse(fmt.Errorf("%s%s", tt.prefix, tt.in).Error())
			if got, want := prefix, tt.prefix; got != want {
				t.Fatalf("got: %s, want %s", got, want)
			}
			if !reflect.DeepEqual(tt.in, se) {
				t.Fatalf("deserialized Error differs: %s", cmp.Diff(tt.in, se))
			}
		})
	}
}
