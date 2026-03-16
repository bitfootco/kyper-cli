package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(logsCmd)
}

var logsCmd = &cobra.Command{
	Use:        "logs",
	Short:      "Stream build logs for the latest version",
	Deprecated: "use 'kyper push logs' instead",
	Args:       cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPushLogs()
	},
}
