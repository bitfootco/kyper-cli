package cmd

import (
	"fmt"
	"os"

	"github.com/bitfootco/kyper-cli/internal/kyperfile"
	"github.com/bitfootco/kyper-cli/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(checkCmd)
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate kyper.yml and confirm Dockerfile exists",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		kf, _, err := loadKyperYML()
		if err != nil {
			return err
		}

		result := kyperfile.Validate(kf, true)

		// Also verify Dockerfile exists on disk (validate already checks this,
		// but we surface it explicitly for the check summary).
		dockerfileExists := true
		if kf.Docker.Dockerfile != "" {
			if _, statErr := os.Stat(kf.Docker.Dockerfile); os.IsNotExist(statErr) {
				dockerfileExists = false
			}
		}

		if jsonOutput {
			out := map[string]interface{}{
				"valid":               result.Valid && dockerfileExists,
				"errors":              result.Errors,
				"warnings":            result.Warnings,
				"dockerfile_exists":   dockerfileExists,
			}
			return ui.PrintJSON(out)
		}

		fmt.Println(ui.Bold.Render("Checking project"))
		fmt.Println()

		for _, e := range result.Errors {
			fmt.Println(ui.Error.Render("  FAIL") + "  " + e)
		}
		for _, w := range result.Warnings {
			fmt.Println(ui.Warning.Render("  WARN") + "  " + w)
		}

		if !result.Valid {
			fmt.Println()
			ui.PrintError(fmt.Sprintf("%d error(s) found", len(result.Errors)))
			return fmt.Errorf("check failed")
		}

		ui.PrintSuccess("kyper.yml is valid")
		if dockerfileExists {
			ui.PrintSuccess("Dockerfile exists")
		}
		fmt.Println()
		ui.PrintSuccess("All checks passed")
		return nil
	},
}
