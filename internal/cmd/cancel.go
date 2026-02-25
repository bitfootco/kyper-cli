package cmd

import (
	"fmt"

	"github.com/bitfootco/kyper-cli/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cancelCmd)
}

var cancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel a pending or building version",
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
			return fmt.Errorf("no versions found")
		}

		v := status.LatestVersion
		if v.Status != "pending" && v.Status != "building" {
			return fmt.Errorf("latest version is %q — can only cancel pending or building versions", v.Status)
		}

		resp, err := client.CancelVersion(v.ID)
		if err != nil {
			return fmt.Errorf("cancelling version: %w", err)
		}

		if jsonOutput {
			return ui.PrintJSON(resp)
		}

		ui.PrintSuccess(resp.Message)
		return nil
	},
}
