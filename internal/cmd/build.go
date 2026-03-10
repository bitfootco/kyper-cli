package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/bitfootco/kyper-cli/internal/kyperfile"
	"github.com/bitfootco/kyper-cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	buildNoCache bool
	buildScan    bool
	buildFixOnly bool
)

func init() {
	buildCmd.Flags().BoolVar(&buildNoCache, "no-cache", false, "Build without Docker layer caching")
	buildCmd.Flags().BoolVar(&buildScan, "scan", false, "Run a Trivy security scan after building (same gate as the remote pipeline)")
	buildCmd.Flags().BoolVar(&buildFixOnly, "fix", false, "With --scan, show only blocking (fixable) vulnerabilities")
	rootCmd.AddCommand(buildCmd)
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the Docker image locally (with optional security scan)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 0. Flag validation
		if buildFixOnly && !buildScan {
			return fmt.Errorf("--fix requires --scan")
		}

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

		// 5. Security scan (if --scan)
		if buildScan {
			ignoredCVEs := make(map[string]bool)
			for _, entry := range kf.Security.IgnoreCVEs {
				if entry.ID != "" {
					ignoredCVEs[entry.ID] = true
				}
			}
			return runBuildScan(imageTag, ignoredCVEs)
		}

		// 6. Success (no scan)
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

func runBuildScan(imageTag string, ignoredCVEs map[string]bool) error {
	// Check trivy is available
	if _, err := exec.LookPath("trivy"); err != nil {
		return fmt.Errorf("trivy not found in $PATH — required for --scan\n\nInstall:\n  brew install trivy          # macOS\n  apt install trivy           # Debian/Ubuntu\n  https://github.com/aquasecurity/trivy/releases  # other")
	}

	if !jsonOutput {
		fmt.Println()
		ui.PrintInfo("Running Trivy security scan...")
		fmt.Println()
	}

	trivyCmd := exec.Command("trivy", "image", "--format", "json",
		"--exit-code", "1", "--severity", "HIGH,CRITICAL", "--no-progress", imageTag)
	trivyOut, trivyErr := trivyCmd.Output()
	// exit code 1 is expected when vulns are found; only fail on empty output
	if len(trivyOut) == 0 && trivyErr != nil {
		return fmt.Errorf("trivy failed to scan the image: %w", trivyErr)
	}

	sr, err := parseTrivyJSON(trivyOut, ignoredCVEs)
	if err != nil {
		return fmt.Errorf("failed to parse Trivy output: %w", err)
	}
	sr.Image = imageTag

	if jsonOutput {
		return ui.PrintJSON(sr)
	}

	printScanResult(sr, buildFixOnly)

	if sr.Status == "failed" {
		return fmt.Errorf("security scan failed")
	}
	return nil
}

// Trivy JSON output types
type trivyOutput struct {
	Results []trivyResult `json:"Results"`
}

type trivyResult struct {
	Vulnerabilities []trivyVuln `json:"Vulnerabilities"`
}

type trivyVuln struct {
	VulnerabilityID string `json:"VulnerabilityID"`
	PkgName         string `json:"PkgName"`
	Severity        string `json:"Severity"`
	FixedVersion    string `json:"FixedVersion"`
}

// Scan result types
type scanVuln struct {
	ID       string `json:"id"`
	Pkg      string `json:"pkg"`
	Severity string `json:"severity"`
	Fixed    string `json:"fixed,omitempty"`
}

type scanResult struct {
	Image           string     `json:"image"`
	Status          string     `json:"status"`
	FixableCritical int        `json:"fixable_critical"`
	FixableHigh     int        `json:"fixable_high"`
	UnfixedCount    int        `json:"unfixed_count"`
	IgnoredCount    int        `json:"ignored_count"`
	Blocking        []scanVuln `json:"blocking"`
	Advisory        []scanVuln `json:"advisory"`
	Ignored         []scanVuln `json:"ignored"`
}

func parseTrivyJSON(raw []byte, ignoredCVEs map[string]bool) (*scanResult, error) {
	var output trivyOutput
	if err := json.Unmarshal(raw, &output); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Collect all vulns, CRITICAL first
	var criticals, highs []scanVuln
	for _, r := range output.Results {
		for _, v := range r.Vulnerabilities {
			sv := scanVuln{
				ID:       v.VulnerabilityID,
				Pkg:      v.PkgName,
				Severity: v.Severity,
				Fixed:    v.FixedVersion,
			}
			if v.Severity == "CRITICAL" {
				criticals = append(criticals, sv)
			} else {
				highs = append(highs, sv)
			}
		}
	}
	all := append(criticals, highs...)

	// Split: ignored → fixable (blocking) / unfixed (advisory)
	var blocking, advisory, ignored []scanVuln
	for _, v := range all {
		if ignoredCVEs[v.ID] {
			ignored = append(ignored, v)
		} else if v.Fixed != "" {
			blocking = append(blocking, v)
		} else {
			advisory = append(advisory, v)
		}
	}

	fixableCritical := 0
	fixableHigh := 0
	for _, v := range blocking {
		if v.Severity == "CRITICAL" {
			fixableCritical++
		} else {
			fixableHigh++
		}
	}

	status := "passed"
	if fixableCritical+fixableHigh > 0 {
		status = "failed"
	} else if len(advisory) > 0 {
		status = "passed_with_advisories"
	} else if len(ignored) > 0 {
		status = "passed_with_ignores"
	}

	return &scanResult{
		Status:          status,
		FixableCritical: fixableCritical,
		FixableHigh:     fixableHigh,
		UnfixedCount:    len(advisory),
		IgnoredCount:    len(ignored),
		Blocking:        blocking,
		Advisory:        advisory,
		Ignored:         ignored,
	}, nil
}

func printScanResult(sr *scanResult, fixOnly bool) {
	fmt.Printf("Security scan: %s\n", ui.Bold.Render(sr.Image))
	fmt.Println()

	// Blocking section
	if len(sr.Blocking) > 0 {
		fmt.Println(ui.Error.Render("BLOCKING") + " (fixable — these will fail your build):")
		printVulnsBySection(sr.Blocking, true)
		fmt.Println()
		fmt.Println("  Action: update your base image to pick up patches, then retry.")
		fmt.Println()
	}

	// Advisory section
	if !fixOnly && len(sr.Advisory) > 0 {
		fmt.Println(ui.DimStyle.Render("ADVISORY") + " (no fix available — will not block your build):")
		printVulnsBySection(sr.Advisory, false)
		fmt.Println()
	}

	// Ignored section
	if !fixOnly && len(sr.Ignored) > 0 {
		fmt.Println(ui.Warning.Render("IGNORED") + fmt.Sprintf(" (%d CVE(s) — declared in kyper.yml):", len(sr.Ignored)))
		printVulnsBySection(sr.Ignored, true)
		fmt.Println()
	}

	// Result line
	switch sr.Status {
	case "failed":
		msg := fmt.Sprintf("WOULD FAIL — %d critical, %d high fixable vulnerabilities", sr.FixableCritical, sr.FixableHigh)
		fmt.Println(ui.ErrorBanner.Render("Result: " + msg))
	case "passed_with_advisories":
		msg := fmt.Sprintf("PASS — %d unfixed CVE(s) flagged as advisory (not blocking)", sr.UnfixedCount)
		fmt.Println(ui.SuccessBanner.Render("Result: " + msg))
	case "passed_with_ignores":
		msg := fmt.Sprintf("PASS — %d CVE(s) ignored per kyper.yml", sr.IgnoredCount)
		fmt.Println(ui.SuccessBanner.Render("Result: " + msg))
	case "passed":
		fmt.Println(ui.SuccessBanner.Render("Result: PASS — no HIGH/CRITICAL vulnerabilities found"))
	}
}

func printVulnsBySection(vulns []scanVuln, showFix bool) {
	for _, sev := range []string{"CRITICAL", "HIGH"} {
		var group []scanVuln
		for _, v := range vulns {
			if v.Severity == sev {
				group = append(group, v)
			}
		}
		if len(group) == 0 {
			continue
		}
		fmt.Printf("  %s:\n", sev)
		for _, v := range group {
			fix := "no fix available"
			if showFix && v.Fixed != "" {
				fix = "fix: " + v.Fixed
			}
			fmt.Printf("    %-20s %-20s %s\n", v.ID, v.Pkg, fix)
		}
	}
}
