package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/bitfootco/kyper-cli/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	pushCmd.AddCommand(pushStatusCmd, pushLogsCmd, pushRetryCmd, pushCancelCmd, pushWithdrawCmd)
}

var pushStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show app and latest version status",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPushStatus()
	},
}

var pushLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Stream build logs for the latest version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPushLogs()
	},
}

var pushRetryCmd = &cobra.Command{
	Use:   "retry",
	Short: "Retry a failed build",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPushRetry()
	},
}

var pushCancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel a pending or building version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPushCancel()
	},
}

var pushWithdrawCmd = &cobra.Command{
	Use:   "withdraw",
	Short: "Withdraw a version from review",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPushWithdraw()
	},
}

// runPushStatus is the shared implementation for `kyper push status` and the
// deprecated `kyper status`.
func runPushStatus() error {
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

	if jsonOutput {
		return ui.PrintJSON(status)
	}

	fmt.Println(ui.Bold.Render("App: ") + kf.Name)
	fmt.Println(ui.Bold.Render("Slug: ") + slug)
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
}

// runPushLogs is the shared implementation for `kyper push logs` and the
// deprecated `kyper logs`.
func runPushLogs() error {
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
		return fmt.Errorf("no versions found — run 'kyper push' first")
	}

	_, err = tailLog(client, status.LatestVersion.ID, 0)
	return err
}

// runPushRetry is the shared implementation for `kyper push retry` and the
// deprecated `kyper retry`.
func runPushRetry() error {
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
}

// runPushCancel is the shared implementation for `kyper push cancel` and the
// deprecated `kyper cancel`.
func runPushCancel() error {
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
}

// runPushWithdraw is the shared implementation for `kyper push withdraw` and
// the deprecated `kyper withdraw`.
func runPushWithdraw() error {
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
		promptErr := huh.NewConfirm().
			Title(fmt.Sprintf("Withdraw version %s?", v.Version)).
			Description("This will remove the version from review.").
			Affirmative("Yes, withdraw").
			Negative("No, keep it").
			Value(&confirm).
			Run()
		if promptErr != nil {
			return promptErr
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
}

