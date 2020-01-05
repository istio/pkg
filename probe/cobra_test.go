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
