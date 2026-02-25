package cmd

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/bitfootco/kyper-cli/internal/api"
	"github.com/bitfootco/kyper-cli/internal/archive"
	"github.com/bitfootco/kyper-cli/internal/config"
	"github.com/bitfootco/kyper-cli/internal/kyperfile"
	"github.com/bitfootco/kyper-cli/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(pushCmd)
}

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Validate, archive, upload, and tail build log",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Require auth
		_, client, err := requireAuth()
		if err != nil {
			return err
		}

		// 2. Read + validate kyper.yml
		kf, raw, err := loadKyperYML()
		if err != nil {
			return err
		}

		result := kyperfile.Validate(kf, true)
		if !result.Valid {
			if jsonOutput {
				return ui.PrintJSON(result)
			}
			for _, e := range result.Errors {
				ui.PrintError(e)
			}
			return fmt.Errorf("kyper.yml validation failed — run 'kyper validate' for details")
		}
		for _, w := range result.Warnings {
			ui.PrintWarning(w)
		}

		slug := slugFromName(kf.Name)

		// 3. Build archive
		tmpDir := os.TempDir()
		zipPath := filepath.Join(tmpDir, slug+"-source.zip")
		defer os.Remove(zipPath)

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

		// 4. Sync app (create or update)
		err = ui.RunWithSpinner("Syncing app...", jsonOutput, func() error {
			_, statusErr := client.GetAppStatus(slug)
			if statusErr != nil {
				if api.IsNotFound(statusErr) {
					// Create new app
					params := buildAppParams(kf)
					_, createErr := client.CreateApp(params)
					return createErr
				}
				return statusErr
			}
			// Update existing app
			params := buildUpdateParams(kf)
			_, updateErr := client.UpdateApp(slug, params)
			return updateErr
		})
		if err != nil {
			return fmt.Errorf("syncing app: %w", err)
		}

		// 5. Upload version
		var vr *api.VersionResponse
		err = ui.RunWithSpinner("Uploading...", jsonOutput, func() error {
			var uploadErr error
			vr, uploadErr = client.CreateVersion(slug, string(raw), zipPath)
			return uploadErr
		})
		if err != nil {
			return fmt.Errorf("uploading version: %w", err)
		}

		if !jsonOutput {
			ui.PrintSuccess(fmt.Sprintf("Version %s uploaded (ID: %d)", vr.Version, vr.ID))
			fmt.Println()
		}

		// 6. Tail build log
		if err := tailLog(client, vr.ID, 0); err != nil {
			return err
		}

		// 7. Check final status and prompt retry on failure
		finalStatus, err := client.GetAppStatus(slug)
		if err == nil && finalStatus.LatestVersion != nil && finalStatus.LatestVersion.Status == "build_failed" {
			if !jsonOutput {
				var retry bool
				if err := huh.NewConfirm().
					Title("Build failed. Retry?").
					Affirmative("Yes").
					Negative("No").
					Value(&retry).
					Run(); err != nil {
					return err
				}
				if retry {
					if _, err := client.RetryVersion(vr.ID); err != nil {
						return fmt.Errorf("retrying build: %w", err)
					}
					return tailLog(client, vr.ID, 0)
				}
			}
		}

		return nil
	},
}

func buildAppParams(kf *config.KyperFile) map[string]interface{} {
	params := map[string]interface{}{
		"title":       kf.Name,
		"description": kf.Description,
		"category":    kf.Category,
	}
	if kf.Tagline != "" {
		params["tagline"] = kf.Tagline
	}

	applyPricingParams(kf, params)

	// Build tech_stack string from processes and deps
	var stacks []string
	for name := range kf.Processes {
		stacks = append(stacks, name)
	}
	if len(stacks) > 0 {
		params["tech_stack"] = fmt.Sprintf("processes: %v", stacks)
	}

	return params
}

func buildUpdateParams(kf *config.KyperFile) map[string]interface{} {
	params := map[string]interface{}{
		"description": kf.Description,
		"category":    kf.Category,
	}
	if kf.Tagline != "" {
		params["tagline"] = kf.Tagline
	}

	applyPricingParams(kf, params)

	return params
}

func applyPricingParams(kf *config.KyperFile, params map[string]interface{}) {
	params["pricing_type"] = derivePricingType(kf)
	if kf.Pricing.OneTime != nil {
		params["one_time_price_cents"] = int(math.Round(*kf.Pricing.OneTime * 100))
	}
	if kf.Pricing.Subscription != nil {
		params["subscription_price_cents"] = int(math.Round(*kf.Pricing.Subscription * 100))
	}
}

func derivePricingType(kf *config.KyperFile) string {
	hasOneTime := kf.Pricing.OneTime != nil
	hasSub := kf.Pricing.Subscription != nil
	if hasOneTime && hasSub {
		return "both"
	}
	if hasOneTime {
		return "one_time"
	}
	if hasSub {
		return "subscription"
	}
	return ""
}

func humanizeBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
