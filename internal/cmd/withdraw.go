package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(withdrawCmd)
}

var withdrawCmd = &cobra.Command{
	Use:        "withdraw",
	Short:      "Withdraw a version from review",
	Deprecated: "use 'kyper push withdraw' instead",
	Args:       cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPushWithdraw()
	},
}
