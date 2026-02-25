package cmd

import (
	"fmt"

	"github.com/bitfootco/kyper-cli/internal/kyperfile"
	"github.com/bitfootco/kyper-cli/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(validateCmd)
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate kyper.yml locally",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		kf, _, err := loadKyperYML()
		if err != nil {
			return err
		}

		result := kyperfile.Validate(kf, true)

		if jsonOutput {
			return ui.PrintJSON(result)
		}

		fmt.Println(ui.Bold.Render("Validating kyper.yml"))
		fmt.Println()

		if len(result.Errors) == 0 && len(result.Warnings) == 0 {
			ui.PrintSuccess("All checks passed")
			return nil
		}

		for _, e := range result.Errors {
			fmt.Println(ui.Error.Render("  FAIL") + "  " + e)
		}
		for _, w := range result.Warnings {
			fmt.Println(ui.Warning.Render("  WARN") + "  " + w)
		}
		fmt.Println()

		if result.Valid {
			ui.PrintSuccess(fmt.Sprintf("Valid with %d warning(s)", len(result.Warnings)))
			return nil
		}

		ui.PrintError(fmt.Sprintf("%d error(s), %d warning(s)", len(result.Errors), len(result.Warnings)))
		return fmt.Errorf("kyper.yml validation failed")
	},
}
