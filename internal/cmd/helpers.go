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

func slugFromName(name string) string {
	s := strings.ToLower(name)
	s = slugRegexp.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func tailLog(client *api.Client, versionID int, startCursor int) error {
	cursor := startCursor
	timeout := time.After(30 * time.Minute)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("build log tailing timed out after 30 minutes")
		default:
		}

		log, err := client.GetBuildLog(versionID, cursor)
		if err != nil {
			return fmt.Errorf("fetching build log: %w", err)
		}

		if log.Log != "" {
			fmt.Print(log.Log)
		}
		cursor = log.Cursor

		if log.Complete {
			fmt.Println()
			printBuildStatus(log.Status)
			return nil
		}

		time.Sleep(2 * time.Second)
	}
}

func printBuildStatus(status string) {
	switch status {
	case "published", "built":
		fmt.Println(ui.SuccessBanner.Render("BUILD SUCCEEDED"))
	case "build_failed":
		fmt.Println(ui.ErrorBanner.Render("BUILD FAILED"))
	case "cancelled":
		fmt.Println(ui.WarningBanner.Render("BUILD CANCELLED"))
	default:
		fmt.Println(ui.Label.Render("Status: " + status))
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
