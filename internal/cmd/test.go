package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bitfootco/kyper-cli/internal/api"
	"github.com/bitfootco/kyper-cli/internal/archive"
	"github.com/bitfootco/kyper-cli/internal/kyperfile"
	"github.com/bitfootco/kyper-cli/internal/ui"
	"github.com/spf13/cobra"
)

var testEnvFile string

// maxNilDeploymentPolls is how many times tailProvisionLog will retry when the
// deployment record is still null right after build completes (~20s window).
const maxNilDeploymentPolls = 10

var (
	testStatus  bool
	testDestroy bool
)

func init() {
	rootCmd.AddCommand(testCmd)
	testCmd.Flags().BoolVar(&testStatus, "status", false, "Show current test deploy status")
	testCmd.Flags().BoolVar(&testDestroy, "destroy", false, "Tear down the active test deploy")
	testCmd.Flags().StringVar(&testEnvFile, "env-file", ".env", "Path to .env file to load for the test deployment")
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Build and ephemerally deploy your app for testing",
	Long: `Build and deploy your app on Kyper's real K8s infrastructure for testing.
The deployment includes your declared deps (Postgres, Redis, etc.) and
auto-destroys after 1 hour.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, client, err := requireAuth()
		if err != nil {
			return err
		}

		if testStatus && testDestroy {
			return fmt.Errorf("--status and --destroy are mutually exclusive")
		}

		kf, raw, err := loadKyperYML()
		if err != nil {
			return err
		}

		slug := slugFromTitle(kf.Name)

		// --status: show current test deploy
		if testStatus {
			return runTestStatus(client, slug)
		}

		// --destroy: tear down active test deploy
		if testDestroy {
			return runTestDestroy(client, slug)
		}

		// Main flow: build + deploy
		result := kyperfile.Validate(kf, true)
		if !result.Valid {
			if jsonOutput {
				_ = ui.PrintJSON(result)
				return fmt.Errorf("kyper.yml validation failed")
			}
			for _, e := range result.Errors {
				ui.PrintError(e)
			}
			return fmt.Errorf("kyper.yml validation failed — run 'kyper validate' for details")
		}
		for _, w := range result.Warnings {
			ui.PrintWarning(w)
		}

		// Build archive
		tmpDir := os.TempDir()
		zipPath := filepath.Join(tmpDir, slug+"-test-source.zip")
		defer func() { _ = os.Remove(zipPath) }()

		err = ui.RunWithSpinner("Building archive...", jsonOutput, func() error {
			return archive.Create(".", zipPath)
		})
		if err != nil {
			return fmt.Errorf("building archive: %w", err)
		}

		info, _ := os.Stat(zipPath)
		if !jsonOutput && info != nil {
			fmt.Printf("Archive: %s\n", humanizeBytes(info.Size()))
		}

		// Sync app (create or update)
		if err = syncApp(client, slug, kf); err != nil {
			return fmt.Errorf("syncing app: %w", err)
		}

		// Load env vars from file (silently skip if missing)
		envVars := parseEnvFile(testEnvFile)
		if !jsonOutput && len(envVars) > 0 {
			fmt.Printf("Loaded %d env var(s) from %s\n", len(envVars), testEnvFile)
		}

		// Submit test deploy
		apiYAML := slugifyYAMLName(raw, slug)
		var tr *api.TestDeployResponse
		err = ui.RunWithSpinner("Queuing test deploy...", jsonOutput, func() error {
			var uploadErr error
			tr, uploadErr = client.CreateTestDeploy(slug, string(apiYAML), zipPath, envVars)
			return uploadErr
		})
		if err != nil {
			return fmt.Errorf("queuing test deploy: %w", err)
		}
		if tr == nil {
			return fmt.Errorf("queuing test deploy: no response from server")
		}

		if !jsonOutput {
			ui.PrintSuccess(tr.Message)
			for _, w := range tr.Warnings {
				ui.PrintWarning(w)
			}
			fmt.Println()
		}

		// Phase 1: stream build log
		if !jsonOutput {
			fmt.Println(ui.Bold.Render("— Build phase —"))
		}

		var buildStatus, buildLog string
		if jsonOutput {
			buildStatus, _, err = waitForBuild(client, tr.VersionID, true)
		} else {
			buildStatus, buildLog, err = waitForBuild(client, tr.VersionID, false)
		}
		if err != nil {
			return err
		}

		if !jsonOutput {
			printBuildStatus(buildStatus)
		}

		if buildStatus == "build_failed" {
			if !jsonOutput && buildLog != "" {
				fmt.Println()
				fmt.Print(buildLog)
			}
			if jsonOutput {
				_ = ui.PrintJSON(map[string]interface{}{
					"version_id":   tr.VersionID,
					"build_status": buildStatus,
				})
			}
			return fmt.Errorf("build failed — run 'kyper build' locally to debug")
		}

		// Phase 2: poll provision log
		if !jsonOutput {
			fmt.Println()
			fmt.Println(ui.Bold.Render("— Provision phase —"))
			fmt.Println(ui.DimStyle.Render("Note: provisioning with deps (Postgres, Redis) can take 3–5 minutes."))
			fmt.Println()
		}

		deployment, err := tailProvisionLog(client, slug)
		if err != nil {
			return err
		}

		if deployment.Status != "running" {
			return fmt.Errorf("test deployment provisioning failed (status: %s)", deployment.Status)
		}

		// Success
		expiresIn := formatExpiresIn(deployment.ExpiresAt)
		if jsonOutput {
			_ = ui.PrintJSON(map[string]interface{}{
				"url":        deployment.URL,
				"expires_at": deployment.ExpiresAt,
				"status":     deployment.Status,
			})
		} else {
			fmt.Println()
			fmt.Println(ui.SuccessBanner.Render("✓ Test deploy is live!"))
			fmt.Printf("  %s\n", deployment.URL)
			fmt.Printf("  Auto-destroys %s. Run 'kyper test --destroy' to tear it down early.\n", expiresIn)
		}

		return nil
	},
}

func runTestStatus(client *api.Client, slug string) error {
	status, err := client.GetTestDeploy(slug, 0)
	if err != nil {
		if api.IsNotFound(err) {
			if jsonOutput {
				_ = ui.PrintJSON(map[string]interface{}{"active": false})
			} else {
				ui.PrintWarning("No active test deploy for this app.")
			}
			return nil
		}
		return fmt.Errorf("fetching test deploy status: %w", err)
	}

	if jsonOutput {
		_ = ui.PrintJSON(status)
		return nil
	}

	fmt.Printf("%s %s\n", ui.Label.Render("Build status:"), status.BuildStatus)
	if status.Deployment != nil {
		d := status.Deployment
		fmt.Printf("%s %s\n", ui.Label.Render("Deploy status:"), d.Status)
		if d.URL != "" {
			fmt.Printf("%s %s\n", ui.Label.Render("URL:"), d.URL)
		}
		if d.ExpiresAt != "" {
			fmt.Printf("%s %s\n", ui.Label.Render("Expires:"), formatExpiresIn(d.ExpiresAt))
		}
	} else {
		ui.PrintWarning("No deployment found (build may still be running).")
	}
	return nil
}

func runTestDestroy(client *api.Client, slug string) error {
	var resp *api.MessageResponse
	err := ui.RunWithSpinner("Tearing down test deploy...", jsonOutput, func() error {
		var destroyErr error
		resp, destroyErr = client.DeleteTestDeploy(slug)
		return destroyErr
	})
	if err != nil {
		if api.IsNotFound(err) {
			if jsonOutput {
				_ = ui.PrintJSON(map[string]interface{}{"active": false})
			} else {
				ui.PrintWarning("No active test deploy to destroy.")
			}
			return nil
		}
		return fmt.Errorf("destroying test deploy: %w", err)
	}
	if jsonOutput {
		_ = ui.PrintJSON(map[string]interface{}{"message": resp.Message})
	} else {
		ui.PrintSuccess(resp.Message)
	}
	return nil
}

// tailProvisionLog polls GET /api/v1/apps/:slug/test_deploy until the deployment
// reaches a terminal state, streaming provision_log content incrementally.
// Returns the final deployment (always non-nil on nil error).
func tailProvisionLog(client *api.Client, slug string) (*api.TestDeployment, error) {
	cursor := 0
	nilRetries := maxNilDeploymentPolls

	for {
		status, err := client.GetTestDeploy(slug, cursor)
		if err != nil {
			if api.IsNotFound(err) {
				return nil, fmt.Errorf("test deploy not found — may have been cancelled")
			}
			return nil, fmt.Errorf("polling provision status: %w", err)
		}

		if status.Deployment == nil {
			nilRetries--
			if nilRetries <= 0 {
				return nil, fmt.Errorf("provision deployment record never appeared")
			}
			time.Sleep(2 * time.Second)
			continue
		}

		d := status.Deployment

		if d.ProvisionLog != "" {
			fmt.Print(d.ProvisionLog)
		}
		cursor = d.ProvisionLogCursor

		switch d.Status {
		case "running", "failed", "terminated", "destroying":
			return d, nil
		}

		time.Sleep(2 * time.Second)
	}
}

func formatExpiresIn(expiresAt string) string {
	if expiresAt == "" {
		return "in ~1 hour"
	}
	t, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return "at " + expiresAt
	}
	remaining := time.Until(t)
	if remaining <= 0 {
		return "soon (expired)"
	}
	mins := int(remaining.Minutes())
	if mins < 60 {
		return fmt.Sprintf("in %d minute(s)", mins)
	}
	hours := mins / 60
	mins = mins % 60
	if mins == 0 {
		return fmt.Sprintf("in %d hour(s)", hours)
	}
	return fmt.Sprintf("in %dh %dm", hours, mins)
}
