package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/huh"
	"github.com/bitfootco/kyper-cli/internal/config"
	"github.com/bitfootco/kyper-cli/internal/detect"
	"github.com/bitfootco/kyper-cli/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive project setup wizard",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if jsonOutput {
			return fmt.Errorf("init command requires interactive mode (remove --json flag)")
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		defaultTitle := toHumanTitle(filepath.Base(cwd))

		// Step 2: Auto-detect (before step 1 form so detection runs first)
		stacks := detect.DetectStack(cwd)
		detectedProcesses := detect.DetectProcesses(cwd)
		detectedDeps := detect.DetectDeps(cwd)

		if len(stacks) > 0 || len(detectedProcesses) > 0 || len(detectedDeps) > 0 {
			fmt.Println()
			fmt.Println(ui.Bold.Render("Auto-detected:"))
			for _, s := range stacks {
				fmt.Printf("  Stack: %s (%s)\n", ui.InfoStyle.Render(s.Name), ui.DimStyle.Render(s.Source))
			}
			for _, p := range detectedProcesses {
				fmt.Printf("  Process: %s → %s (%s)\n", ui.InfoStyle.Render(p.Name), p.Command, ui.DimStyle.Render(p.Source))
			}
			for _, d := range detectedDeps {
				fmt.Printf("  Dep: %s (%s)\n", ui.InfoStyle.Render(d.Name), ui.DimStyle.Render(d.Source))
			}
			fmt.Println()
		}

		// Step 1: App basics — split into two groups so the category list doesn't crowd other fields
		var title, category, tagline, description string

		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("App title").
					Description("Human-readable display name (e.g. \"My App\"). Slug is auto-derived.").
					Value(&title).
					Placeholder(defaultTitle),
				huh.NewSelect[string]().
					Title("Category").
					Options(categoryOptions()...).
					Value(&category),
			),
			huh.NewGroup(
				huh.NewInput().
					Title("Tagline").
					Description("Short pitch (max 160 chars, optional)").
					Value(&tagline).
					CharLimit(160),
				huh.NewText().
					Title("Description").
					Description("What does your app do? (max 500 chars)").
					Value(&description).
					CharLimit(500),
			),
		).Run()
		if err != nil {
			return err
		}

		if title == "" {
			title = defaultTitle
		}

		// Step 3: Processes
		processes := make(map[string]string)
		if len(detectedProcesses) > 0 {
			useDetected := true // default Yes
			if err := huh.NewConfirm().
				Title("Use detected processes?").
				Value(&useDetected).
				Run(); err != nil {
				return err
			}
			if useDetected {
				for _, p := range detectedProcesses {
					processes[p.Name] = p.Command
				}
			}
		}

		if _, ok := processes["web"]; !ok {
			var webCmd string
			if err := huh.NewInput().
				Title("Web process command").
				Description("Required. The command to start your web server.").
				Value(&webCmd).
				Run(); err != nil {
				return err
			}
			processes["web"] = webCmd
		}

		// Step 4: Dependencies
		var selectedDeps []config.DepEntry
		if len(detectedDeps) > 0 {
			depOptions := make([]huh.Option[string], len(detectedDeps))
			for i, d := range detectedDeps {
				depOptions[i] = huh.NewOption(fmt.Sprintf("%s (from %s)", d.Name, d.Source), d.Name)
			}
			var chosen []string
			if err := huh.NewMultiSelect[string]().
				Title("Select dependencies").
				Options(depOptions...).
				Value(&chosen).
				Run(); err != nil {
				return err
			}

			// Suggest versions from lockfiles
			versionSuggestions := detect.SuggestDepVersions(cwd, detectedDeps)
			versionMap := make(map[string]string)
			for _, vs := range versionSuggestions {
				versionMap[vs.Dep] = vs.Version
			}

			for _, depName := range chosen {
				dep := config.DepEntry{Name: depName}
				if v, ok := versionMap[depName]; ok {
					dep.Version = v
				}
				selectedDeps = append(selectedDeps, dep)
			}
		}

		// Step 5: Hooks
		stackNames := detect.StackNames(stacks)
		var onDeploy string
		hasDB := false
		for _, d := range selectedDeps {
			if d.Name == "postgres" || d.Name == "mysql" {
				hasDB = true
				break
			}
		}
		if hasDB {
			suggestion := suggestHook(stackNames)
			if err := huh.NewInput().
				Title("Deploy hook").
				Description("Run after first deployment (e.g., database migration)").
				Value(&onDeploy).
				Placeholder(suggestion).
				Run(); err != nil {
				return err
			}
			if onDeploy == "" {
				onDeploy = suggestion
			}
		}

		// Steps 6–8: Health check, pricing, resources — one form to avoid terminal artifacts
		defaultPath := defaultHealthPath(stackNames)
		var healthPath, oneTimeStr, subStr, memoryTier string
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Health check path").
					Description("Path to check if your app is running").
					Value(&healthPath).
					Placeholder(defaultPath),
			),
			huh.NewGroup(
				huh.NewInput().
					Title("One-time price (USD)").
					Description("Leave blank if not applicable").
					Value(&oneTimeStr),
				huh.NewInput().
					Title("Monthly subscription (USD)").
					Description("Leave blank if not applicable").
					Value(&subStr),
			),
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Resource tier").
					Description("Minimum resources allocated to your app").
					Options(
						huh.NewOption("Hobby — 512 MB RAM, 1 vCPU ($6/mo)", "512"),
						huh.NewOption("Starter — 1 GB RAM, 1 vCPU ($12/mo)", "1024"),
						huh.NewOption("Pro — 2 GB RAM, 2 vCPU ($18/mo)", "2048"),
						huh.NewOption("Business — 4 GB RAM, 4 vCPU ($24/mo)", "4096"),
					).
					Value(&memoryTier),
			),
		).Run()
		if err != nil {
			return err
		}
		if healthPath == "" {
			healthPath = defaultPath
		}

		// Build KyperFile struct
		kf := buildKyperFile(title, description, tagline, category, processes,
			selectedDeps, onDeploy, healthPath, oneTimeStr, subStr, memoryTier)

		// Step 9: Preview
		yamlBytes, err := yaml.Marshal(kf)
		if err != nil {
			return fmt.Errorf("generating YAML: %w", err)
		}

		rendered, err := glamour.Render("```yaml\n"+string(yamlBytes)+"\n```", "dark")
		if err != nil {
			// Fallback to plain output
			fmt.Println(string(yamlBytes))
		} else {
			fmt.Println(rendered)
		}

		var confirm bool
		if err := huh.NewConfirm().
			Title("Write kyper.yml?").
			Value(&confirm).
			Run(); err != nil {
			return err
		}

		if !confirm {
			fmt.Println("Cancelled.")
			return nil
		}

		if err := os.WriteFile("kyper.yml", yamlBytes, 0644); err != nil {
			return fmt.Errorf("writing kyper.yml: %w", err)
		}

		ui.PrintSuccess("Created kyper.yml")
		return nil
	},
}

func buildKyperFile(title, description, tagline, category string,
	processes map[string]string, deps []config.DepEntry,
	onDeploy, healthPath, oneTimeStr, subStr, memoryTier string) *config.KyperFile {

	kf := &config.KyperFile{
		Title:       title,
		Version:     "0.1.0",
		Description: description,
		Category:    category,
		Docker: config.DockerConfig{
			Dockerfile: "./Dockerfile",
		},
		Processes: processes,
		Deps:      deps,
		Healthcheck: config.HealthcheckConfig{
			Path:     healthPath,
			Interval: 30,
			Timeout:  10,
		},
	}

	if tagline != "" {
		kf.Tagline = tagline
	}

	if onDeploy != "" {
		kf.Hooks.OnDeploy = onDeploy
	}

	if p := parsePrice(oneTimeStr); p != nil {
		kf.Pricing.OneTime = p
	}
	if p := parsePrice(subStr); p != nil {
		kf.Pricing.Subscription = p
	}

	if mem := parseInt(memoryTier); mem > 0 {
		kf.Resources.MinMemoryMB = mem
		switch mem {
		case 2048:
			kf.Resources.MinCPU = 2
		case 4096:
			kf.Resources.MinCPU = 4
		default:
			kf.Resources.MinCPU = 1
		}
	}

	return kf
}

// toHumanTitle converts a filesystem name like "my_app" or "my-project" to "My App" / "My Project".
func toHumanTitle(s string) string {
	s = strings.NewReplacer("_", " ", "-", " ").Replace(s)
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

func parsePrice(s string) *float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "$")
	if s == "" {
		return nil
	}
	var f float64
	if _, err := fmt.Sscanf(s, "%f", &f); err == nil && f > 0 {
		return &f
	}
	return nil
}

func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func suggestHook(stacks []string) string {
	for _, s := range stacks {
		switch s {
		case "rails":
			return "bundle exec rails db:migrate"
		case "django":
			return "python manage.py migrate"
		case "prisma":
			return "npx prisma migrate deploy"
		case "laravel":
			return "php artisan migrate --force"
		}
	}
	return ""
}

func defaultHealthPath(stacks []string) string {
	for _, s := range stacks {
		switch s {
		case "rails":
			return "/up"
		case "django":
			return "/health/"
		case "express", "next", "nest", "koa":
			return "/health"
		}
	}
	return "/up"
}

func categoryOptions() []huh.Option[string] {
	categories := []struct {
		label string
		value string
	}{
		{"Developer Tools", "developer_tools"},
		{"Productivity", "productivity"},
		{"Finance", "finance"},
		{"Health", "health"},
		{"Media", "media"},
		{"Education", "education"},
		{"Business Operations", "business_operations"},
		{"Data & Analytics", "data_analytics"},
		{"Gaming", "gaming"},
	}
	opts := make([]huh.Option[string], len(categories))
	for i, c := range categories {
		opts[i] = huh.NewOption(c.label, c.value)
	}
	return opts
}
