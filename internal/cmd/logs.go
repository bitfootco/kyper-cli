package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(logsCmd)
}

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Stream build logs for the latest version",
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

		if status.LatestVersion == nil {
			return fmt.Errorf("no versions found — run 'kyper push' first")
		}

		return tailLog(client, status.LatestVersion.ID, 0)
	},
}
