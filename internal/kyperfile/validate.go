package kyperfile

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/bitfootco/kyper-cli/internal/config"
)

var CATEGORIES = []string{
	"developer_tools",
	"productivity",
	"finance",
	"health",
	"media",
	"education",
	"business_operations",
	"data_analytics",
	"gaming",
}

var KNOWN_DEPS = []string{
	"postgres",
	"mysql",
	"redis",
	"elasticsearch",
	"opensearch",
}

var ALLOWED_DEP_VERSIONS = map[string][]string{
	"postgres":      {"14", "15", "16"},
	"mysql":         {"8"},
	"redis":         {"6", "7"},
	"elasticsearch": {"8"},
	"opensearch":    {"2"},
}

var AUTO_INJECTED_ENV = []string{
	"DATABASE_URL",
	"REDIS_URL",
	"SECRET_KEY_BASE",
	"PORT",
	"KYPER_DEPLOYMENT_ID",
	"ELASTICSEARCH_URL",
	"OPENSEARCH_URL",
}

var DB_DEPS = map[string]bool{
	"postgres": true,
	"mysql":    true,
}

var semverRegexp = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// Validate checks a KyperFile against all validation rules.
// If checkFileExists is true, it verifies the Dockerfile exists on disk.
func Validate(kf *config.KyperFile, checkFileExists bool) *ValidationResult {
	r := &ValidationResult{Valid: true}

	validateName(kf, r)
	validateVersion(kf, r)
	validateCategory(kf, r)
	validateDescription(kf, r)
	validateTagline(kf, r)
	validateDocker(kf, r, checkFileExists)
	validateProcesses(kf, r)
	validateDeps(kf, r)
	validateHooks(kf, r)
	validateHealthcheck(kf, r)
	validatePricing(kf, r)
	validateEnv(kf, r)
	checkDBWithoutHook(kf, r)

	return r
}

func addError(r *ValidationResult, msg string) {
	r.Valid = false
	r.Errors = append(r.Errors, msg)
}

func addWarning(r *ValidationResult, msg string) {
	r.Warnings = append(r.Warnings, msg)
}

func validateName(kf *config.KyperFile, r *ValidationResult) {
	if kf.Name == "" {
		addError(r, "name is required")
		return
	}
	if len(kf.Name) > 100 {
		addError(r, "name must be 100 characters or fewer")
	}
}

func validateVersion(kf *config.KyperFile, r *ValidationResult) {
	if kf.Version == "" {
		addError(r, "version is required")
		return
	}
	if !semverRegexp.MatchString(kf.Version) {
		addError(r, "version must be semver (MAJOR.MINOR.PATCH)")
	}
}

func validateCategory(kf *config.KyperFile, r *ValidationResult) {
	if kf.Category == "" {
		addError(r, "category is required")
		return
	}
	for _, c := range CATEGORIES {
		if kf.Category == c {
			return
		}
	}
	addError(r, fmt.Sprintf("category must be one of: %s", strings.Join(CATEGORIES, ", ")))
}

func validateDescription(kf *config.KyperFile, r *ValidationResult) {
	if kf.Description == "" {
		addError(r, "description is required")
		return
	}
	if len(kf.Description) > 500 {
		addError(r, "description must be 500 characters or fewer")
	}
}

func validateTagline(kf *config.KyperFile, r *ValidationResult) {
	if kf.Tagline != "" && len(kf.Tagline) > 160 {
		addError(r, "tagline must be 160 characters or fewer")
	}
}

func validateDocker(kf *config.KyperFile, r *ValidationResult, checkFileExists bool) {
	if kf.Docker.Image != "" {
		addError(r, "docker.image is not supported — Kyper builds from source using docker.dockerfile")
	}
	if kf.Docker.Dockerfile == "" {
		addError(r, "docker.dockerfile is required")
		return
	}
	if checkFileExists {
		if _, err := os.Stat(kf.Docker.Dockerfile); os.IsNotExist(err) {
			addError(r, fmt.Sprintf("docker.dockerfile %q not found", kf.Docker.Dockerfile))
		}
	}
}

func validateProcesses(kf *config.KyperFile, r *ValidationResult) {
	if len(kf.Processes) == 0 {
		addError(r, "processes is required")
		return
	}
	if _, ok := kf.Processes["web"]; !ok {
		addError(r, "processes must include a 'web' entry")
	}
}

func validateDeps(kf *config.KyperFile, r *ValidationResult) {
	for _, dep := range kf.Deps {
		if dep.Name == "" {
			addError(r, "dep entry has empty name")
			continue
		}

		known := false
		for _, k := range KNOWN_DEPS {
			if dep.Name == k {
				known = true
				break
			}
		}
		if !known {
			addError(r, fmt.Sprintf("unknown dep %q — known deps: %s", dep.Name, strings.Join(KNOWN_DEPS, ", ")))
			continue
		}

		if dep.Version != "" {
			allowed := ALLOWED_DEP_VERSIONS[dep.Name]
			valid := false
			for _, v := range allowed {
				if dep.Version == v {
					valid = true
					break
				}
			}
			if !valid {
				addError(r, fmt.Sprintf("dep %q version %q is not allowed — allowed: %s", dep.Name, dep.Version, strings.Join(allowed, ", ")))
			}
		}

		if dep.StorageGB != 0 && (dep.StorageGB < 1 || dep.StorageGB > 500) {
			addError(r, fmt.Sprintf("dep %q storage_gb must be between 1 and 500", dep.Name))
		}
	}
}

func validateHooks(kf *config.KyperFile, r *ValidationResult) {
	// Hooks are strings — Go typing handles this, but check they're non-empty if present is implicit
}

func validateHealthcheck(kf *config.KyperFile, r *ValidationResult) {
	if kf.Healthcheck.Path != "" && !strings.HasPrefix(kf.Healthcheck.Path, "/") {
		addError(r, "healthcheck.path must start with /")
	}
	if kf.Healthcheck.Interval != 0 && (kf.Healthcheck.Interval < 10 || kf.Healthcheck.Interval > 300) {
		addError(r, "healthcheck.interval must be between 10 and 300")
	}
	if kf.Healthcheck.Timeout != 0 && kf.Healthcheck.Timeout < 1 {
		addError(r, "healthcheck.timeout must be a positive integer")
	}
}

func validatePricing(kf *config.KyperFile, r *ValidationResult) {
	if kf.Pricing.OneTime == nil && kf.Pricing.Subscription == nil {
		addError(r, "at least one pricing option is required (one_time or subscription)")
		return
	}
	if kf.Pricing.OneTime != nil && *kf.Pricing.OneTime < 1.0 {
		addError(r, "pricing.one_time must be at least $1.00")
	}
	if kf.Pricing.Subscription != nil && *kf.Pricing.Subscription < 1.0 {
		addError(r, "pricing.subscription must be at least $1.00")
	}
}

func validateEnv(kf *config.KyperFile, r *ValidationResult) {
	autoInjected := make(map[string]bool)
	for _, e := range AUTO_INJECTED_ENV {
		autoInjected[e] = true
	}
	for _, e := range kf.Env {
		if e == "" {
			addError(r, "env entries must be non-empty strings")
		}
		if autoInjected[e] {
			addWarning(r, fmt.Sprintf("env %q is auto-injected by Kyper and cannot be overridden", e))
		}
	}
}

func checkDBWithoutHook(kf *config.KyperFile, r *ValidationResult) {
	hasDB := false
	for _, dep := range kf.Deps {
		if DB_DEPS[dep.Name] {
			hasDB = true
			break
		}
	}
	if hasDB && kf.Hooks.OnDeploy == "" {
		addWarning(r, "database dependency present without hooks.on_deploy — consider adding a migration hook")
	}
}
