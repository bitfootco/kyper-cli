package cmd

import (
	"fmt"
	"time"

	"github.com/bitfootco/kyper-cli/internal/api"
	"github.com/bitfootco/kyper-cli/internal/config"
	"github.com/bitfootco/kyper-cli/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate via browser (device auth flow)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURL(), "")

		// Step 1: Request device code
		var grant *api.DeviceGrant
		err := ui.RunWithSpinner("Requesting device code...", jsonOutput, func() error {
			var e error
			grant, e = client.DeviceAuthorize()
			return e
		})
		if err != nil {
			return fmt.Errorf("requesting device code: %w", err)
		}

		// Step 2: Show verification URL
		fmt.Println()
		fmt.Println(ui.Bold.Render("Open this URL in your browser to authenticate:"))
		fmt.Println()
		fmt.Println("  " + ui.InfoStyle.Render(grant.VerificationURI))
		fmt.Println()

		// Try to open browser
		if err := openBrowser(grant.VerificationURI); err != nil {
			ui.PrintWarning("Could not open browser automatically")
		}

		// Step 3: Poll for token
		var token string
		err = ui.RunWithSpinner("Waiting for authorization...", jsonOutput, func() error {
			deadline := time.After(5 * time.Minute)
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-deadline:
					return fmt.Errorf("authorization timed out (5 minutes)")
				case <-ticker.C:
					resp, err := client.DeviceToken(grant.Code)
					if err != nil {
						if api.IsNotFound(err) {
							return fmt.Errorf("device code expired — run 'kyper login' again")
						}
						return err
					}
					if resp.Pending {
						continue
					}
					if resp.APIToken != "" {
						token = resp.APIToken
						return nil
					}
				}
			}
		})
		if err != nil {
			return err
		}

		// Step 4: Save token
		cfg := &config.Config{APIToken: token}
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		// Step 5: Verify identity
		authedClient := api.NewClient(baseURL(), token)
		user, err := authedClient.GetMe()
		if err != nil {
			return fmt.Errorf("verifying identity: %w", err)
		}

		if jsonOutput {
			return ui.PrintJSON(map[string]string{
				"email": user.Email,
				"role":  user.Role,
			})
		}

		fmt.Println()
		ui.PrintSuccess(fmt.Sprintf("Logged in as %s (%s)", user.Email, user.Role))
		return nil
	},
}
