package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/bitfootco/kyper-cli/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(withdrawCmd)
}

var withdrawCmd = &cobra.Command{
	Use:   "withdraw",
	Short: "Withdraw a version from review",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, client, err := requireAuth()
		if err != nil {
			return err
		}

		kf, _, err := loadKyperYML()
		if err != nil {
			return err
		}

		slug := slugFromTitle(kf.Name)
		status, err := client.GetAppStatus(slug)
		if err != nil {
			return fmt.Errorf("fetching status: %w", err)
		}

		if status.LatestVersion == nil {
			return fmt.Errorf("no versions found")
		}

		v := status.LatestVersion
		if v.Status == "published" || v.Status == "building" {
			return fmt.Errorf("latest version is %q — cannot withdraw", v.Status)
		}

		if !jsonOutput {
			var confirm bool
			err := huh.NewConfirm().
				Title(fmt.Sprintf("Withdraw version %s?", v.Version)).
				Description("This will remove the version from review.").
				Affirmative("Yes, withdraw").
				Negative("No, keep it").
				Value(&confirm).
				Run()
			if err != nil {
				return err
			}
			if !confirm {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		resp, err := client.DeleteVersion(v.ID)
		if err != nil {
			return fmt.Errorf("withdrawing version: %w", err)
		}

		if jsonOutput {
			return ui.PrintJSON(resp)
		}

		ui.PrintSuccess(resp.Message)
		return nil
	},
}
