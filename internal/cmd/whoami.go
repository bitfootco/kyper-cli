package cmd

import (
	"fmt"

	"github.com/bitfootco/kyper-cli/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(whoamiCmd)
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show authenticated user",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, client, err := requireAuth()
		if err != nil {
			return err
		}

		user, err := client.GetMe()
		if err != nil {
			return fmt.Errorf("fetching user: %w", err)
		}

		if jsonOutput {
			return ui.PrintJSON(user)
		}

		fmt.Printf("%s (%s)\n", user.Email, user.Role)
		return nil
	},
}
