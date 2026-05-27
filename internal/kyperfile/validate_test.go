package kyperfile

import (
	"strings"
	"testing"

	"github.com/bitfootco/kyper-cli/internal/config"
	"gopkg.in/yaml.v3"
)

func floatPtr(f float64) *float64 { return &f }

func validKyperFile() *config.KyperFile {
	return &config.KyperFile{
		Name:     "my-app",
		Version:  "1.0.0",
		Category: "productivity",
		Docker: config.DockerConfig{
			Dockerfile: "./Dockerfile",
		},
		Processes: map[string]string{
			"web": "bin/start",
		},
		Pricing: config.PricingConfig{
			OneTime: floatPtr(29.99),
		},
	}
}

func TestValidFile(t *testing.T) {
	kf := validKyperFile()
	r := Validate(kf, false)
	if !r.Valid {
		t.Errorf("expected valid, got errors: %v", r.Errors)
	}
}

func TestNameRequired(t *testing.T) {
	kf := validKyperFile()
	kf.Name = ""
	r := Validate(kf, false)
	if r.Valid {
		t.Error("expected invalid")
	}
	assertContainsError(t, r, "name is required")
}

func TestNameValidFormats(t *testing.T) {
	tests := []string{
		"my-app",
		"dashi",
		"my-cool-app",
		"app2",
		"my-app-2",
		"a",
		"x1",
	}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			kf := validKyperFile()
			kf.Name = name
			r := Validate(kf, false)
			if !r.Valid {
				t.Errorf("name %q should be valid, got errors: %v", name, r.Errors)
			}
		})
	}
}

func TestNameInvalidFormats(t *testing.T) {
	tests := []string{
		"My App",        // uppercase + space
		"Dashi",         // uppercase
		"MY-APP",        // uppercase
		"-leading",      // leading hyphen
		"trailing-",     // trailing hyphen
		"App 2.0",       // uppercase + space + dot
		"café",          // non-ASCII
		"my_app",        // underscore
		"my app",        // space
		"my.app",        // dot
		"!!!",           // symbols only
		"---",           // hyphens only
		"   ",           // whitespace only
	}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			kf := validKyperFile()
			kf.Name = name
			r := Validate(kf, false)
			if r.Valid {
				t.Errorf("name %q should be invalid", name)
			}
			assertContainsError(t, r, "valid slug")
		})
	}
}

func TestNameTooLong(t *testing.T) {
	kf := validKyperFile()
	kf.Name = strings.Repeat("a", 101)
	r := Validate(kf, false)
	assertContainsError(t, r, "100 characters")
}

func TestVersionRequired(t *testing.T) {
	kf := validKyperFile()
	kf.Version = ""
	r := Validate(kf, false)
	assertContainsError(t, r, "version is required")
}

func TestVersionSemver(t *testing.T) {
	kf := validKyperFile()
	tests := []struct {
		version string
		valid   bool
	}{
		{"1.0.0", true},
		{"0.1.0", true},
		{"10.20.30", true},
		{"1.0", false},
		{"v1.0.0", false},
		{"1.0.0-beta", false},
		{"abc", false},
	}
	for _, tt := range tests {
		kf.Version = tt.version
		r := Validate(kf, false)
		if tt.valid && !r.Valid {
			t.Errorf("version %q should be valid, got errors: %v", tt.version, r.Errors)
		}
		if !tt.valid && r.Valid {
			t.Errorf("version %q should be invalid", tt.version)
		}
	}
}

func TestCategoryRequired(t *testing.T) {
	kf := validKyperFile()
	kf.Category = ""
	r := Validate(kf, false)
	assertContainsError(t, r, "category is required")
}

func TestCategoryInvalid(t *testing.T) {
	kf := validKyperFile()
	kf.Category = "not_a_category"
	r := Validate(kf, false)
	assertContainsError(t, r, "category must be one of")
}

func TestAllValidCategories(t *testing.T) {
	kf := validKyperFile()
	for _, c := range Categories {
		kf.Category = c
		r := Validate(kf, false)
		if !r.Valid {
			t.Errorf("category %q should be valid, got errors: %v", c, r.Errors)
		}
	}
}


func TestTaglineTooLong(t *testing.T) {
	kf := validKyperFile()
	kf.Tagline = strings.Repeat("a", 161)
	r := Validate(kf, false)
	assertContainsError(t, r, "160 characters")
}

func TestDockerDockerfileRequired(t *testing.T) {
	kf := validKyperFile()
	kf.Docker.Dockerfile = ""
	r := Validate(kf, false)
	assertContainsError(t, r, "docker.dockerfile is required")
}

func TestDockerImageRejected(t *testing.T) {
	kf := validKyperFile()
	kf.Docker.Image = "myimage:latest"
	r := Validate(kf, false)
	assertContainsError(t, r, "docker.image is not supported")
}

func TestProcessesRequired(t *testing.T) {
	kf := validKyperFile()
	kf.Processes = nil
	r := Validate(kf, false)
	assertContainsError(t, r, "processes is required")
}

func TestProcessesWebRequired(t *testing.T) {
	kf := validKyperFile()
	kf.Processes = map[string]string{"worker": "bundle exec sidekiq"}
	r := Validate(kf, false)
	assertContainsError(t, r, "processes must include a 'web' entry")
}

func TestDepsKnown(t *testing.T) {
	kf := validKyperFile()
	kf.Deps = []config.DepEntry{{Name: "unknown_dep"}}
	r := Validate(kf, false)
	assertContainsError(t, r, "unknown dep")
}

func TestDepsValidVersion(t *testing.T) {
	kf := validKyperFile()
	kf.Deps = []config.DepEntry{{Name: "postgres", Version: "16"}}
	r := Validate(kf, false)
	if !r.Valid {
		t.Errorf("expected valid, got errors: %v", r.Errors)
	}
}

func TestDepsInvalidVersion(t *testing.T) {
	kf := validKyperFile()
	kf.Deps = []config.DepEntry{{Name: "postgres", Version: "99"}}
	r := Validate(kf, false)
	assertContainsError(t, r, "not allowed")
}

func TestDepsStorageGBValid(t *testing.T) {
	kf := validKyperFile()
	kf.Deps = []config.DepEntry{{Name: "postgres", StorageGB: 50}}
	r := Validate(kf, false)
	if !r.Valid {
		t.Errorf("expected valid, got errors: %v", r.Errors)
	}
}

func TestDepsStorageGBInvalid(t *testing.T) {
	kf := validKyperFile()
	kf.Deps = []config.DepEntry{{Name: "postgres", StorageGB: 501}}
	r := Validate(kf, false)
	assertContainsError(t, r, "storage_gb must be between 1 and 500")
}

func TestDepsStorageGBZero(t *testing.T) {
	kf := validKyperFile()
	kf.Deps = []config.DepEntry{{Name: "postgres", StorageGB: -1}}
	r := Validate(kf, false)
	assertContainsError(t, r, "storage_gb must be between 1 and 500")
}

func TestHealthcheckPathMustStartWithSlash(t *testing.T) {
	kf := validKyperFile()
	kf.Healthcheck.Path = "health"
	r := Validate(kf, false)
	assertContainsError(t, r, "healthcheck.path must start with /")
}

func TestHealthcheckPathValid(t *testing.T) {
	kf := validKyperFile()
	kf.Healthcheck.Path = "/up"
	r := Validate(kf, false)
	if !r.Valid {
		t.Errorf("expected valid, got errors: %v", r.Errors)
	}
}

func TestHealthcheckIntervalRange(t *testing.T) {
	kf := validKyperFile()
	kf.Healthcheck.Interval = 5
	r := Validate(kf, false)
	assertContainsError(t, r, "healthcheck.interval must be between 10 and 300")

	kf.Healthcheck.Interval = 301
	r = Validate(kf, false)
	assertContainsError(t, r, "healthcheck.interval must be between 10 and 300")
}

func TestHealthcheckTimeoutPositive(t *testing.T) {
	kf := validKyperFile()
	kf.Healthcheck.Timeout = 0
	r := Validate(kf, false)
	if !r.Valid {
		t.Errorf("expected valid with timeout 0 (unset)")
	}

	kf.Healthcheck.Timeout = -1
	r = Validate(kf, false)
	assertContainsError(t, r, "healthcheck.timeout must be a positive integer")
}

func TestHealthcheckInitialDelay(t *testing.T) {
	kf := validKyperFile()
	kf.Healthcheck.InitialDelay = 30
	r := Validate(kf, false)
	if !r.Valid {
		t.Errorf("expected valid with initial_delay 30")
	}

	kf.Healthcheck.InitialDelay = 0
	r = Validate(kf, false)
	if !r.Valid {
		t.Errorf("expected valid with initial_delay 0 (unset)")
	}

	kf.Healthcheck.InitialDelay = 301
	r = Validate(kf, false)
	assertContainsError(t, r, "healthcheck.initial_delay must be between 0 and 300")

	kf.Healthcheck.InitialDelay = -1
	r = Validate(kf, false)
	assertContainsError(t, r, "healthcheck.initial_delay must be between 0 and 300")
}

func TestPricingMinimum(t *testing.T) {
	kf := validKyperFile()
	low := 0.50
	kf.Pricing.OneTime = &low
	r := Validate(kf, false)
	assertContainsError(t, r, "pricing.one_time must be at least $1.00")

	kf.Pricing.OneTime = nil
	kf.Pricing.Subscription = &low
	r = Validate(kf, false)
	assertContainsError(t, r, "pricing.subscription must be at least $1.00")
}

func TestPricingValid(t *testing.T) {
	kf := validKyperFile()
	price := 29.99
	kf.Pricing.OneTime = &price
	r := Validate(kf, false)
	if !r.Valid {
		t.Errorf("expected valid, got errors: %v", r.Errors)
	}
}

func TestEnvNonEmpty(t *testing.T) {
	kf := validKyperFile()
	kf.Env = []string{"API_KEY", ""}
	r := Validate(kf, false)
	assertContainsError(t, r, "env entries must be non-empty strings")
}

func TestEnvAutoInjectedWarning(t *testing.T) {
	kf := validKyperFile()
	kf.Env = []string{"DATABASE_URL"}
	r := Validate(kf, false)
	if !r.Valid {
		t.Errorf("auto-injected env should be a warning, not error")
	}
	assertContainsWarning(t, r, "auto-injected")
}

func TestDepsS3Valid(t *testing.T) {
	kf := validKyperFile()
	kf.Deps = []config.DepEntry{{Name: "s3"}}
	r := Validate(kf, false)
	if !r.Valid {
		t.Errorf("expected valid, got errors: %v", r.Errors)
	}
}

func TestDepsS3ValidWithStorageGB(t *testing.T) {
	kf := validKyperFile()
	kf.Deps = []config.DepEntry{{Name: "s3", StorageGB: 100}}
	r := Validate(kf, false)
	if !r.Valid {
		t.Errorf("expected valid, got errors: %v", r.Errors)
	}
}

func TestDepsS3VersionPinningRejected(t *testing.T) {
	kf := validKyperFile()
	kf.Deps = []config.DepEntry{{Name: "s3", Version: "2"}}
	r := Validate(kf, false)
	assertContainsError(t, r, "does not support version pinning")
}

func TestDepsS3StorageGBBelowTenWarns(t *testing.T) {
	kf := validKyperFile()
	kf.Deps = []config.DepEntry{{Name: "s3", StorageGB: 5}}
	r := Validate(kf, false)
	assertContainsWarning(t, r, "s3 storage_gb is below 10 GB")
}

func TestDepsS3StorageGBTenNoWarn(t *testing.T) {
	kf := validKyperFile()
	kf.Deps = []config.DepEntry{{Name: "s3", StorageGB: 10}}
	r := Validate(kf, false)
	for _, w := range r.Warnings {
		if strings.Contains(w, "s3 storage_gb") {
			t.Error("should not warn about s3 storage_gb when >= 10")
		}
	}
}

func TestEnvAWSVarsAutoInjectedWarning(t *testing.T) {
	kf := validKyperFile()
	kf.Env = []string{"AWS_ACCESS_KEY_ID"}
	r := Validate(kf, false)
	if !r.Valid {
		t.Errorf("auto-injected AWS env should be a warning, not error")
	}
	assertContainsWarning(t, r, "auto-injected")
}

func TestDBWithoutHookWarning(t *testing.T) {
	kf := validKyperFile()
	kf.Deps = []config.DepEntry{{Name: "postgres"}}
	r := Validate(kf, false)
	assertContainsWarning(t, r, "hooks.release")
}

func TestDBWithReleaseHookNoWarning(t *testing.T) {
	kf := validKyperFile()
	kf.Deps = []config.DepEntry{{Name: "postgres"}}
	kf.Hooks.Release = "bundle exec rails db:prepare"
	r := Validate(kf, false)
	for _, w := range r.Warnings {
		if strings.Contains(w, "hooks.release") {
			t.Error("should not warn about hooks.release when it is set")
		}
	}
}

// hooks.on_deploy is not a substitute for hooks.release for migrations on a
// fresh DB — the warning should still fire if only on_deploy is set.
func TestDBWithOnlyOnDeployStillWarns(t *testing.T) {
	kf := validKyperFile()
	kf.Deps = []config.DepEntry{{Name: "postgres"}}
	kf.Hooks.OnDeploy = "bundle exec rails db:migrate"
	r := Validate(kf, false)
	assertContainsWarning(t, r, "hooks.release")
}

func TestUnknownHookKeyRejected(t *testing.T) {
	kf := validKyperFile()
	kf.Hooks.Unknown = map[string]interface{}{"releaes": "prisma migrate deploy"}
	r := Validate(kf, false)
	if r.Valid {
		t.Error("expected validation to fail for unknown hook key")
	}
	found := false
	for _, e := range r.Errors {
		if strings.Contains(e, "releaes") && strings.Contains(e, "unknown key") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an unknown-key error mentioning %q, got errors: %v", "releaes", r.Errors)
	}
}

func TestKnownHooksDoNotPopulateUnknown(t *testing.T) {
	// Round-trip a YAML with all three known hooks set, confirming the
	// Unknown map stays empty (and the validator doesn't trip).
	yamlStr := `
name: test
version: 1.0.0
category: productivity
docker:
  dockerfile: ./Dockerfile
processes:
  web: bin/server
pricing:
  one_time: 5.0
hooks:
  release: rails db:prepare
  on_deploy: rails cache:warmup
  on_update: rails cache:warmup
`
	var kf config.KyperFile
	if err := yaml.Unmarshal([]byte(yamlStr), &kf); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(kf.Hooks.Unknown) != 0 {
		t.Errorf("expected Unknown map to be empty, got %v", kf.Hooks.Unknown)
	}
	r := Validate(&kf, false)
	for _, e := range r.Errors {
		if strings.Contains(e, "unknown key") {
			t.Errorf("did not expect an unknown-key error, got: %s", e)
		}
	}
}


func TestIntegrationsValid(t *testing.T) {
	kf := validKyperFile()
	kf.Integrations = []string{"stripe", "twilio"}
	r := Validate(kf, false)
	if !r.Valid {
		t.Errorf("expected valid, got errors: %v", r.Errors)
	}
}

func TestIntegrationsUnknown(t *testing.T) {
	kf := validKyperFile()
	kf.Integrations = []string{"stripe", "unknown_service"}
	r := Validate(kf, false)
	assertContainsError(t, r, "unknown integration \"unknown_service\"")
}

func TestIntegrationsEmptyString(t *testing.T) {
	kf := validKyperFile()
	kf.Integrations = []string{"stripe", ""}
	r := Validate(kf, false)
	assertContainsError(t, r, "integration entries must be non-empty strings")
}

func TestPricingRequired(t *testing.T) {
	kf := validKyperFile()
	kf.Pricing.OneTime = nil
	kf.Pricing.Subscription = nil
	r := Validate(kf, false)
	assertContainsError(t, r, "at least one pricing option")
}

func assertContainsError(t *testing.T, r *ValidationResult, substr string) {
	t.Helper()
	if r.Valid {
		t.Errorf("expected invalid result for %q", substr)
		return
	}
	for _, e := range r.Errors {
		if strings.Contains(e, substr) {
			return
		}
	}
	t.Errorf("expected error containing %q, got: %v", substr, r.Errors)
}

func assertContainsWarning(t *testing.T, r *ValidationResult, substr string) {
	t.Helper()
	for _, w := range r.Warnings {
		if strings.Contains(w, substr) {
			return
		}
	}
	t.Errorf("expected warning containing %q, got: %v", substr, r.Warnings)
}
