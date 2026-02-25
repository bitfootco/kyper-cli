package cmd

import (
	"fmt"

	"github.com/bitfootco/kyper-cli/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show app and latest version status",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		kf, _, err := loadKyperYML()
		if err != nil {
			return err
		}

		_, client, err := requireAuth()
		if err != nil {
			return err
		}

		slug := slugFromName(kf.Name)
		status, err := client.GetAppStatus(slug)
		if err != nil {
			return fmt.Errorf("fetching status: %w", err)
		}

		if jsonOutput {
			return ui.PrintJSON(status)
		}

		fmt.Println(ui.Bold.Render("App: ") + status.App.Title)
		fmt.Println(ui.Bold.Render("Slug: ") + status.App.Slug)
		fmt.Println(ui.Bold.Render("Status: ") + formatStatus(status.Status))
		fmt.Println()

		if status.LatestVersion != nil {
			v := status.LatestVersion
			fmt.Println(ui.Bold.Render("Latest Version"))
			fmt.Println("  Version: " + v.Version)
			fmt.Println("  Status:  " + formatStatus(v.Status))
			if v.ReviewNotes != "" {
				fmt.Println("  Notes:   " + v.ReviewNotes)
			}
		} else {
			fmt.Println(ui.DimStyle.Render("No versions pushed yet"))
		}

		return nil
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
