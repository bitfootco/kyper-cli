package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/bitfootco/kyper-cli/internal/ui"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var bumpFlag string

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Bump the version in kyper.yml",
	Long:  "Interactively select a patch, minor, or major version bump and write it to kyper.yml.",
	Args:  cobra.NoArgs,
	RunE:  runTag,
}

func init() {
	tagCmd.Flags().StringVar(&bumpFlag, "bump", "", "Version bump type: patch, minor, or major")
	rootCmd.AddCommand(tagCmd)
}

func runTag(cmd *cobra.Command, args []string) error {
	kf, raw, err := loadKyperYML()
	if err != nil {
		return err
	}

	major, minor, patch, err := parseVersion(kf.Version)
	if err != nil {
		return fmt.Errorf("invalid version in kyper.yml: %w", err)
	}

	patchVer := fmt.Sprintf("%d.%d.%d", major, minor, patch+1)
	minorVer := fmt.Sprintf("%d.%d.%d", major, minor+1, 0)
	majorVer := fmt.Sprintf("%d.%d.%d", major+1, 0, 0)

	bump := bumpFlag
	if bump == "" {
		if jsonOutput {
			return fmt.Errorf("use --bump with --json")
		}
		var selected string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select version bump").
					Options(
						huh.NewOption(fmt.Sprintf("patch  %s → %s", kf.Version, patchVer), "patch"),
						huh.NewOption(fmt.Sprintf("minor  %s → %s", kf.Version, minorVer), "minor"),
						huh.NewOption(fmt.Sprintf("major  %s → %s", kf.Version, majorVer), "major"),
					).
					Value(&selected),
			),
		)
		if err := form.Run(); err != nil {
			return err
		}
		bump = selected
	}

	var newVersion string
	switch bump {
	case "patch":
		newVersion = patchVer
	case "minor":
		newVersion = minorVer
	case "major":
		newVersion = majorVer
	default:
		return fmt.Errorf("invalid --bump value %q: must be patch, minor, or major", bump)
	}

	updated, err := replaceVersion(raw, kf.Version, newVersion)
	if err != nil {
		return fmt.Errorf("updating version: %w", err)
	}

	if err := os.WriteFile("kyper.yml", updated, 0644); err != nil {
		return fmt.Errorf("writing kyper.yml: %w", err)
	}

	if jsonOutput {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{
			"previous_version": kf.Version,
			"new_version":      newVersion,
		})
	}

	ui.PrintSuccess(fmt.Sprintf("Version bumped %s → %s", kf.Version, newVersion))
	return nil
}

func parseVersion(v string) (major, minor, patch int, err error) {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("expected MAJOR.MINOR.PATCH, got %q", v)
	}
	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major: %w", err)
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor: %w", err)
	}
	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid patch: %w", err)
	}
	return major, minor, patch, nil
}

func replaceVersion(raw []byte, oldVersion, newVersion string) ([]byte, error) {
	pattern := fmt.Sprintf(`(?m)^version:\s*%s\s*$`, regexp.QuoteMeta(oldVersion))
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	if !re.Match(raw) {
		return nil, fmt.Errorf("version %q not found in kyper.yml", oldVersion)
	}
	return re.ReplaceAllLiteral(raw, []byte("version: "+newVersion)), nil
}
