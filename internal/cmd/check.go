package cmd

import (
	"fmt"
	"os"

	"github.com/bitfootco/kyper-cli/internal/kyperfile"
	"github.com/bitfootco/kyper-cli/internal/ui"
	"github.com/spf13/cobra"
)

var checkQuick bool

func init() {
	checkCmd.Flags().BoolVar(&checkQuick, "quick", false, "Skip Dockerfile existence check (validation only)")
	rootCmd.AddCommand(checkCmd)
}

var checkCmd = &cobra.Command{
	Use:     "check",
	Short:   "Validate kyper.yml and confirm Dockerfile exists",
	GroupID: "project",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCheck(checkQuick)
	},
}

func runCheck(quick bool) error {
	kf, _, err := loadKyperYML()
	if err != nil {
		return err
	}

	result := kyperfile.Validate(kf, true)

	// Also verify Dockerfile exists on disk unless --quick
	dockerfileExists := true
	if !quick && kf.Docker.Dockerfile != "" {
		if _, statErr := os.Stat(kf.Docker.Dockerfile); os.IsNotExist(statErr) {
			dockerfileExists = false
		}
	}

	if jsonOutput {
		out := map[string]interface{}{
			"valid":    result.Valid && (quick || dockerfileExists),
			"errors":   result.Errors,
			"warnings": result.Warnings,
		}
		if !quick {
			out["dockerfile_exists"] = dockerfileExists
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
	if !quick {
		if dockerfileExists {
			ui.PrintSuccess("Dockerfile exists")
		} else {
			fmt.Println()
			ui.PrintError("Dockerfile not found: " + kf.Docker.Dockerfile)
			return fmt.Errorf("check failed")
		}
	}
	fmt.Println()
	ui.PrintSuccess("All checks passed")
	return nil
}
