package cmd

import (
	"fmt"

	"github.com/bitfootco/kyper-cli/internal/ui"
	"github.com/bitfootco/kyper-cli/internal/version"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if jsonOutput {
			return ui.PrintJSON(map[string]string{
				"version": version.Version,
				"commit":  version.Commit,
				"date":    version.Date,
			})
		}
		fmt.Printf("kyper %s (%s, %s)\n", version.Version, version.Commit, version.Date)
		return nil
	},
}
