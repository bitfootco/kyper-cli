package cmd

import (
	"fmt"

	"github.com/bitfootco/kyper-cli/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(retryCmd)
}

var retryCmd = &cobra.Command{
	Use:   "retry",
	Short: "Retry a failed build",
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
		if v.Status != "build_failed" {
			return fmt.Errorf("latest version is %q — can only retry failed builds", v.Status)
		}

		resp, err := client.RetryVersion(v.ID)
		if err != nil {
			return fmt.Errorf("retrying build: %w", err)
		}

		if jsonOutput {
			return ui.PrintJSON(resp)
		}

		ui.PrintSuccess(resp.Message)
		fmt.Println()

		buildStatus, buildLog, waitErr := waitForBuild(client, v.ID, false)
		if waitErr != nil {
			return waitErr
		}
		printBuildStatus(buildStatus)
		if buildStatus == "build_failed" && buildLog != "" {
			fmt.Println()
			fmt.Print(buildLog)
		}
		return nil
	},
}
