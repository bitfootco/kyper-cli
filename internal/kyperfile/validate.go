package kyperfile

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bitfootco/kyper-cli/internal/config"
)

var Categories = []string{
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

var KnownDeps = []string{
	"postgres",
	"mysql",
	"redis",
	"elasticsearch",
	"opensearch",
	"s3",
}

var KnownIntegrations = []string{
	"stripe",
	"twilio",
	"sendgrid",
	"google_maps",
	"resend",
	"postmark",
}

var KnownCapabilities = []string{
	"email",
	"maps",
	"payments",
	"sms",
}

var AllowedDepVersions = map[string][]string{
	"postgres":      {"14", "15", "16"},
	"mysql":         {"8"},
	"redis":         {"6", "7"},
	"elasticsearch": {"8"},
	"opensearch":    {"2"},
	"s3":            {}, // no version pinning — Kyper manages the SeaweedFS image
}

var AutoInjectedEnv = []string{
	"DATABASE_URL",
	"REDIS_URL",
	"SECRET_KEY_BASE",
	"PORT",
	"KYPER_DEPLOYMENT_ID",
	"ELASTICSEARCH_URL",
	"OPENSEARCH_URL",
	"AWS_ACCESS_KEY_ID",
	"AWS_SECRET_ACCESS_KEY",
	"AWS_BUCKET",
	"AWS_ENDPOINT_URL",
}

var DBDeps = map[string]bool{
	"postgres": true,
	"mysql":    true,
}

var semverRegexp = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
var nameSlugRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)
var cveIDRegexp = regexp.MustCompile(`^CVE-\d{4}-\d{4,}$`)
var storageMountPathRegexp = regexp.MustCompile(`^/?[A-Za-z0-9._/-]+$`)

const maxIgnoredCVEs = 25
const minReasonLength = 10
const maxStorageMounts = 5

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
	validateTagline(kf, r)
	validateDocker(kf, r, checkFileExists)
	validateProcesses(kf, r)
	validateDeps(kf, r)
	validateHealthcheck(kf, r)
	validatePricing(kf, r)
	validateResources(kf, r)
	validateEnv(kf, r)
	validateCapabilities(kf, r)
	validateIntegrations(kf, r)
	validateStorage(kf, r)
	validateHooks(kf, r)
	checkDBWithoutHook(kf, r)
	validateSecurity(kf, r)

	return r
}

func validateResources(kf *config.KyperFile, r *ValidationResult) {
	if kf.Resources.HasMinMemoryMB() && kf.Resources.MinMemoryMB <= 0 {
		addError(r, "resources.min_memory_mb must be a positive integer")
	}
	if kf.Resources.HasMinCPU() && kf.Resources.MinCPU <= 0 {
		addError(r, "resources.min_cpu must be a positive number")
	}
}

func validateStorage(kf *config.KyperFile, r *ValidationResult) {
	if len(kf.Storage.Mounts) == 0 {
		return
	}
	if len(kf.Storage.Mounts) > maxStorageMounts {
		addError(r, fmt.Sprintf("storage.mounts is limited to %d entries", maxStorageMounts))
	}

	seen := make(map[string]bool)
	for i, mount := range kf.Storage.Mounts {
		normalized, ok := normalizeStorageMountPath(mount.Path)
		if !ok {
			addError(r, fmt.Sprintf("storage.mounts[%d].path must be a safe absolute or relative path", i))
		} else if seen[normalized] {
			addError(r, fmt.Sprintf("storage.mounts contains duplicate path %q", normalized))
		} else {
			seen[normalized] = true
		}

		if mount.StorageGB != nil && (*mount.StorageGB < 1 || *mount.StorageGB > 10) {
			addError(r, fmt.Sprintf("storage.mounts[%d].storage_gb must be between 1 and 10", i))
		}
	}
}

func normalizeStorageMountPath(path string) (string, bool) {
	raw := strings.TrimSpace(path)
	if raw == "" || raw == "/" || raw == "." {
		return "", false
	}
	if !storageMountPathRegexp.MatchString(raw) {
		return "", false
	}

	absolute := strings.HasPrefix(raw, "/")
	clean := filepath.Clean(raw)
	if clean == "." || clean == "/" {
		return "", false
	}
	if clean != strings.TrimSuffix(raw, "/") {
		return "", false
	}
	for _, segment := range strings.Split(clean, "/") {
		if segment == ".." {
			return "", false
		}
	}
	if absolute && !strings.HasPrefix(clean, "/") {
		clean = "/" + clean
	}
	return clean, true
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
	if !nameSlugRegexp.MatchString(kf.Name) {
		addError(r, "name must be a valid slug: lowercase letters, digits, and hyphens only, no leading or trailing hyphens (e.g. \"my-app\" or \"dashi\")")
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
	for _, c := range Categories {
		if kf.Category == c {
			return
		}
	}
	addError(r, fmt.Sprintf("category must be one of: %s", strings.Join(Categories, ", ")))
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
		for _, k := range KnownDeps {
			if dep.Name == k {
				known = true
				break
			}
		}
		if !known {
			addError(r, fmt.Sprintf("unknown dep %q — known deps: %s", dep.Name, strings.Join(KnownDeps, ", ")))
			continue
		}

		if dep.Version != "" {
			allowed := AllowedDepVersions[dep.Name]
			if len(allowed) == 0 {
				addError(r, fmt.Sprintf("dep %q does not support version pinning", dep.Name))
			} else {
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
		}

		if dep.StorageGB != 0 && (dep.StorageGB < 1 || dep.StorageGB > 500) {
			addError(r, fmt.Sprintf("dep %q storage_gb must be between 1 and 500", dep.Name))
		}

		if dep.Name == "s3" && dep.StorageGB > 0 && dep.StorageGB < 10 {
			addWarning(r, "s3 storage_gb is below 10 GB — files can be large; consider at least 10 GB")
		}
	}
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
	if kf.Healthcheck.InitialDelay < 0 || kf.Healthcheck.InitialDelay > 300 {
		addError(r, "healthcheck.initial_delay must be between 0 and 300")
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
	for _, e := range AutoInjectedEnv {
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

func validateIntegrations(kf *config.KyperFile, r *ValidationResult) {
	if len(kf.Integrations) == 0 {
		return
	}
	for _, name := range kf.Integrations {
		if name == "" {
			addError(r, "integration entries must be non-empty strings")
			continue
		}
		known := false
		for _, k := range KnownIntegrations {
			if name == k {
				known = true
				break
			}
		}
		if !known {
			addError(r, fmt.Sprintf("unknown integration %q — known integrations: %s", name, strings.Join(KnownIntegrations, ", ")))
		}
	}
	addWarning(r, "integrations is deprecated; use capabilities instead (email, maps, payments, sms)")
}

func validateCapabilities(kf *config.KyperFile, r *ValidationResult) {
	if len(kf.Capabilities) == 0 {
		return
	}
	for _, name := range kf.Capabilities {
		if strings.TrimSpace(name) == "" {
			addError(r, "capability entries must be non-empty strings")
			continue
		}
		known := false
		for _, k := range KnownCapabilities {
			if name == k {
				known = true
				break
			}
		}
		if !known {
			addError(r, fmt.Sprintf("unknown capability %q — known capabilities: %s", name, strings.Join(KnownCapabilities, ", ")))
		}
	}
}

// validateHooks rejects unknown keys under `hooks:`. Without this check, a
// typo like `hooks.releaes` would silently no-op the hook, and the failure
// mode is the exact one hooks.release exists to prevent (an app crashing on
// first boot waiting for a schema that no migration ever ran). The known-key
// list mirrors the Rails validator at app/services/kyper_file/validator.rb.
func validateHooks(kf *config.KyperFile, r *ValidationResult) {
	if len(kf.Hooks.Unknown) == 0 {
		return
	}

	keys := make([]string, 0, len(kf.Hooks.Unknown))
	for k := range kf.Hooks.Unknown {
		keys = append(keys, fmt.Sprintf("%q", k))
	}
	addError(r, fmt.Sprintf(
		"hooks: unknown key(s) %s — allowed keys: release, on_deploy, on_update",
		strings.Join(keys, ", "),
	))
}

func checkDBWithoutHook(kf *config.KyperFile, r *ValidationResult) {
	hasDB := false
	for _, dep := range kf.Deps {
		if DBDeps[dep.Name] {
			hasDB = true
			break
		}
	}
	if hasDB && kf.Hooks.Release == "" {
		addWarning(r, "database dependency present without hooks.release — apps with a database usually need a migration command (e.g. 'bundle exec rails db:prepare' or 'prisma migrate deploy'). hooks.release runs after the DB is ready and before your app starts, on every cold deploy. Do NOT use hooks.on_deploy for migrations — it runs after pods are healthy, so it can't help an app that crashes on first boot.")
	}
}

func validateSecurity(kf *config.KyperFile, r *ValidationResult) {
	if len(kf.Security.IgnoreCVEs) == 0 {
		return
	}

	if len(kf.Security.IgnoreCVEs) > maxIgnoredCVEs {
		addError(r, fmt.Sprintf("security.ignore_cves is limited to %d entries (got %d)", maxIgnoredCVEs, len(kf.Security.IgnoreCVEs)))
	}

	for i, entry := range kf.Security.IgnoreCVEs {
		if entry.ID == "" {
			addError(r, fmt.Sprintf("security.ignore_cves[%d].id is required", i))
		} else if !cveIDRegexp.MatchString(entry.ID) {
			addError(r, fmt.Sprintf("security.ignore_cves[%d].id must match CVE-YYYY-NNNNN format (got %q)", i, entry.ID))
		}

		if entry.Reason == "" {
			addError(r, fmt.Sprintf("security.ignore_cves[%d].reason is required", i))
		} else if len(entry.Reason) < minReasonLength {
			addError(r, fmt.Sprintf("security.ignore_cves[%d].reason must be at least %d characters", i, minReasonLength))
		}
	}

	addWarning(r, fmt.Sprintf("security.ignore_cves is set — %d CVE(s) will bypass the security gate", len(kf.Security.IgnoreCVEs)))
}
