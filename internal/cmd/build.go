package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/bitfootco/kyper-cli/internal/kyperfile"
	"github.com/bitfootco/kyper-cli/internal/ui"
	"github.com/spf13/cobra"
)

var buildNoCache bool

func init() {
	buildCmd.Flags().BoolVar(&buildNoCache, "no-cache", false, "Build without Docker layer caching")
	rootCmd.AddCommand(buildCmd)
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the Docker image locally",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Check docker is available
		if _, err := exec.LookPath("docker"); err != nil {
			return fmt.Errorf("docker not found in $PATH — install Docker: https://docs.docker.com/get-docker/")
		}

		// 2. Load + validate kyper.yml
		kf, _, err := loadKyperYML()
		if err != nil {
			return err
		}

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

		if !jsonOutput {
			ui.PrintSuccess("kyper.yml is valid")
		}

		// 3. Confirm Dockerfile exists
		dockerfile := kf.Docker.Dockerfile
		if _, err := os.Stat(dockerfile); os.IsNotExist(err) {
			return fmt.Errorf("dockerfile not found: %s", dockerfile)
		}

		// 4. Build image
		imageTag := fmt.Sprintf("kyper-local/%s:%s", slugFromTitle(kf.Name), kf.Version)

		if !jsonOutput {
			fmt.Printf("Building %s from %s\n\n", ui.Bold.Render(imageTag), dockerfile)
		}

		buildArgs := []string{"build", "-f", dockerfile, "-t", imageTag}
		if buildNoCache {
			buildArgs = append(buildArgs, "--no-cache")
		}
		buildArgs = append(buildArgs, ".")

		dockerCmd := exec.Command("docker", buildArgs...)
		dockerCmd.Stdout = os.Stdout
		dockerCmd.Stderr = os.Stderr
		dockerCmd.Stdin = os.Stdin

		if err := dockerCmd.Run(); err != nil {
			if !jsonOutput {
				fmt.Println()
				ui.PrintError("Build failed. Fix the issue above, then retry.")
			}
			return fmt.Errorf("docker build failed")
		}

		// 5. Success
		if jsonOutput {
			return ui.PrintJSON(map[string]string{
				"image":  imageTag,
				"status": "success",
			})
		}

		fmt.Println()
		ui.PrintSuccess(fmt.Sprintf("Build succeeded: %s", imageTag))
		fmt.Println()
		ui.PrintInfo("Run: kyper push")

		return nil
	},
}
