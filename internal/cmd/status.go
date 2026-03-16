package cmd

import (
	"github.com/bitfootco/kyper-cli/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:        "status",
	Short:      "Show app and latest version status",
	Deprecated: "use 'kyper push status' instead",
	Args:       cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPushStatus()
	},
}

func formatStatus(s string) string {
	switch s {
	case "published", "active", "built":
		return ui.Success.Render(s)
	case "build_failed", "rejected":
		return ui.Error.Render(s)
	case "pending", "building", "in_review":
		return ui.Warning.Render(s)
	default:
		return s
	}
}
