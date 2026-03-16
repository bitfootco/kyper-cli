package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(validateCmd)
}

var validateCmd = &cobra.Command{
	Use:        "validate",
	Short:      "Validate kyper.yml locally",
	Deprecated: "use 'kyper check' instead (use 'kyper check --quick' to skip Dockerfile check)",
	Args:       cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCheck(true)
	},
}
