package probe

import (
	"fmt"

	"github.com/spf13/cobra"
)

func CobraCommand() *cobra.Command {
	var (
		probeOptions Options
	)

	prb := &cobra.Command{
		Use:   "probe",
		Short: "Check the liveness or readiness of a locally-running server",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !probeOptions.IsValid() {
				return fmt.Errorf("some options are not valid")
			}
			if err := NewFileClient(&probeOptions).GetStatus(); err != nil {
				return fmt.Errorf("fail on inspecting path %s: %v", probeOptions.Path, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "OK")
			return nil
		},
	}
	prb.PersistentFlags().StringVar(&probeOptions.Path, "probe-path", "",
		"Path of the file for checking the availability.")
	prb.PersistentFlags().DurationVar(&probeOptions.UpdateInterval, "interval", 0,
		"Duration used for checking the target file's last modified time.")

	return prb
}
