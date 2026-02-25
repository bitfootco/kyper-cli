package cmd

import (
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	hostFlag   string
)

var rootCmd = &cobra.Command{
	Use:   "kyper",
	Short: "Kyper CLI — push, validate, and manage apps on the Kyper marketplace",
	Long:  "The official CLI for the Kyper marketplace. Build, deploy, and manage your apps.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output raw JSON (for scripting)")
	rootCmd.PersistentFlags().StringVar(&hostFlag, "host", "", "Override API host URL")
}

func Execute() error {
	return rootCmd.Execute()
}
