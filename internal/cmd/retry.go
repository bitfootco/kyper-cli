package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(retryCmd)
}

var retryCmd = &cobra.Command{
	Use:        "retry",
	Short:      "Retry a failed build",
	Deprecated: "use 'kyper push retry' instead",
	Args:       cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPushRetry()
	},
}
