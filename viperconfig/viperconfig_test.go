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

package viperconfig

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestViperConfig(t *testing.T) {
	v := viper.GetViper()
	var foo string
	hasRun := false
	c := cobra.Command{
		Run: func(c *cobra.Command, args []string) {
			assert.Equal(t, foo, "somethingelse")
			hasRun = true
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			v.BindPFlags(cmd.Flags())
		},
	}
	c.PersistentFlags().StringVar(&foo, "foo", "notempty", "foo is a fake flag")
	ViperizeRootCmdDefault(&c)

	c.SetArgs([]string{"--config", "testconfig.yaml"})
	c.Execute()
	if !hasRun {
		assert.True(t, hasRun, "command never ran")
	}
}

func TestDuplicate(t *testing.T) {
	var configv []string
	var cmdLine []string
	var defaultv []string
	hasRunRoot := false
	c := cobra.Command{Run: func(c *cobra.Command, args []string) {
		assert.Equal(t, configv, []string{"third", "fourth"})
		assert.Equal(t, cmdLine, []string{"first", "second"})
		assert.Equal(t, defaultv, []string{"notempty"})
		hasRunRoot = true
	}}
	c.PersistentFlags().StringSliceVar(&configv, "configv", []string{"notempty"}, "configv is a fake flag")
	c.PersistentFlags().StringSliceVar(&cmdLine, "cmdLine", []string{"notempty"}, "cmdLine is a fake flag")
	c.PersistentFlags().StringSliceVar(&defaultv, "defaultv", []string{"notempty"}, "defaultv is a fake flag")
	ViperizeRootCmdDefault(&c)
	// run root and check
	c.SetArgs([]string{"--config", "./testlistconfig.yaml", "--cmdLine", "first", "--cmdLine", "second"})
	c.Execute()
	assert.True(t, hasRunRoot, "root command never ran")
}

func TestViperize(t *testing.T) {
	var foo string
	var bar string
	hasRunRoot := false
	hasRunPre := false
	hasRunSub := false
	c := cobra.Command{Run: func(c *cobra.Command, args []string) {
		assert.Equal(t, foo, "somethingelse")
		hasRunRoot = true
	}, PreRun: func(c *cobra.Command, args []string) {
		hasRunPre = true
	}}
	c.PersistentFlags().StringVar(&foo, "foo", "notempty", "foo is a fake flag")
	sub := cobra.Command{Use: "sub", Run: func(c *cobra.Command, args []string) {
		// this is not getting set, probably because it's not in sub's persistent flag set...
		assert.Equal(t, foo, "expected")
		assert.Equal(t, bar, "alsoexpected")
		hasRunSub = true
	}}
	sub.PersistentFlags().StringVar(&bar, "bar", "notepmty", "bar is also a fake flag")
	c.AddCommand(&sub)
	ViperizeRootCmdDefault(&c)
	// run root and check
	c.SetArgs([]string{"--config", "testconfig.yaml"})
	c.Execute()
	assert.True(t, hasRunRoot, "root command never ran")
	assert.True(t, hasRunPre, "root command never preran")
	assert.False(t, hasRunSub, "subcommand ran unintentionally")
	// reset the test
	foo = ""
	bar = ""
	hasRunRoot = false
	hasRunSub = false
	hasRunPre = false
	// run sub and check
	c.SetArgs([]string{"sub", "--config", "testsubconfig.yaml"})
	c.Execute()
	assert.True(t, hasRunSub, "subcommand never ran")
	assert.False(t, hasRunRoot, "root command ran unintentionally")
	assert.False(t, hasRunPre, "root command preran unintentionally")
}
