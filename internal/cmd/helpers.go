package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/bitfootco/kyper-cli/internal/api"
	"github.com/bitfootco/kyper-cli/internal/config"
	"github.com/bitfootco/kyper-cli/internal/ui"
)

const defaultBaseURL = "https://kyper.shop"

func baseURL() string {
	if hostFlag != "" {
		return hostFlag
	}
	if env := os.Getenv("KYPER_HOST"); env != "" {
		return env
	}
	return defaultBaseURL
}

func requireAuth() (*config.Config, *api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("loading config: %w", err)
	}
	if cfg.APIToken == "" {
		return nil, nil, fmt.Errorf("not logged in — run 'kyper login' first")
	}
	client := api.NewClient(baseURL(), cfg.APIToken)
	return cfg, client, nil
}

func loadKyperYML() (*config.KyperFile, []byte, error) {
	kf, raw, err := config.LoadKyperFile("kyper.yml")
	if err != nil {
		return nil, nil, fmt.Errorf("reading kyper.yml: %w\nRun 'kyper init' to create one", err)
	}
	return kf, raw, nil
}

var slugRegexp = regexp.MustCompile(`[^a-z0-9]+`)

func slugFromTitle(title string) string {
	s := strings.ToLower(title)
	s = slugRegexp.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

var yamlNameRegexp = regexp.MustCompile(`(?m)^name:\s*.*$`)

func slugifyYAMLName(raw []byte, slug string) []byte {
	return yamlNameRegexp.ReplaceAll(raw, []byte("name: "+slug))
}

// tailLog streams the build log for a version, printing output as it arrives.
// It returns the final build status (e.g. "built", "build_failed", "in_review").
func tailLog(client *api.Client, versionID int, startCursor int) (string, error) {
	cursor := startCursor
	timeout := time.After(30 * time.Minute)

	for {
		select {
		case <-timeout:
			return "", fmt.Errorf("build log tailing timed out after 30 minutes")
		default:
		}

		log, err := client.GetBuildLog(versionID, cursor)
		if err != nil {
			return "", fmt.Errorf("fetching build log: %w", err)
		}

		if log.Log != "" {
			fmt.Print(log.Log)
		}
		cursor = log.Cursor

		if log.Complete {
			fmt.Println()
			printBuildStatus(log.Status)
			return log.Status, nil
		}

		time.Sleep(2 * time.Second)
	}
}

// waitForBuild polls until the build completes, showing a spinner.
// Returns the final status string and, on build_failed, the full build log.
func waitForBuild(client *api.Client, versionID int, jsonMode bool) (status string, buildLog string, err error) {
	spinErr := ui.RunWithSpinner("Building...", jsonMode, func() error {
		cursor := 0
		timeout := time.After(30 * time.Minute)
		for {
			select {
			case <-timeout:
				return fmt.Errorf("build timed out after 30 minutes")
			default:
			}
			bl, pollErr := client.GetBuildLog(versionID, cursor)
			if pollErr != nil {
				return fmt.Errorf("fetching build log: %w", pollErr)
			}
			cursor = bl.Cursor
			if bl.Complete {
				status = bl.Status
				return nil
			}
			time.Sleep(2 * time.Second)
		}
	})
	if spinErr != nil {
		return "", "", spinErr
	}
	if status == "build_failed" {
		full, logErr := client.GetBuildLog(versionID, 0)
		if logErr == nil && full != nil {
			buildLog = full.Log
		}
	}
	return status, buildLog, nil
}

func printBuildStatus(status string) {
	switch status {
	case "published", "built":
		fmt.Println(ui.SuccessBanner.Render("✓ Build succeeded"))
	case "in_review":
		fmt.Println(ui.SuccessBanner.Render("✓ Build succeeded — submitted for review"))
	case "build_failed":
		fmt.Println(ui.ErrorBanner.Render("✗ Build failed"))
	case "cancelled":
		fmt.Println(ui.WarningBanner.Render("! Build cancelled"))
	default:
		fmt.Println(ui.Bold.Render("Status: ") + status)
	}
}

func openBrowser(url string) error {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux":
		cmd = "xdg-open"
	default:
		return fmt.Errorf("unsupported platform — open %s manually", url)
	}
	return exec.Command(cmd, url).Start()
}
