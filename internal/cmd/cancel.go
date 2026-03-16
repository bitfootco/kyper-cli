package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cancelCmd)
}

var cancelCmd = &cobra.Command{
	Use:        "cancel",
	Short:      "Cancel a pending or building version",
	Deprecated: "use 'kyper push cancel' instead",
	Args:       cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPushCancel()
	},
}
