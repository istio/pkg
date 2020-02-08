// Copyright 2020 Istio Authors

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

package probe

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestOpts(t *testing.T) {
	fp, err := ioutil.TempFile(os.TempDir(), t.Name())
	if err != nil {
		t.Fatalf("failed to create temp file")
	}
	cases := []struct {
		args       string
		expectFail bool
	}{
		{
			"probe",
			true,
		},
		{
			"extra-args",
			true,
		},
		{
			fmt.Sprintf("--probe-path=%s.invalid --interval=1s", fp.Name()),
			true,
		},
		{
			fmt.Sprintf("--probe-path=%s --interval=1s", fp.Name()),
			false,
		},
	}

	for _, v := range cases {
		t.Run(v.args, func(t *testing.T) {
			cmd := CobraCommand()
			cmd.SetArgs(strings.Split(v.args, " "))
			err := cmd.Execute()

			if !v.expectFail && err != nil {
				t.Errorf("Got %v, expecting success", err)
			}
			if v.expectFail && err == nil {
				t.Errorf("Expected failure, got success")
			}
		})
	}
}
