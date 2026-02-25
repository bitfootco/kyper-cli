package kyperfile

import (
	"strings"
	"testing"

	"github.com/bitfootco/kyper-cli/internal/config"
)

func floatPtr(f float64) *float64 { return &f }

func validKyperFile() *config.KyperFile {
	return &config.KyperFile{
		Name:        "My App",
		Version:     "1.0.0",
		Description: "A valid test app",
		Category:    "productivity",
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

func TestNameNoAlphanumeric(t *testing.T) {
	kf := validKyperFile()
	kf.Name = "!!!"
	r := Validate(kf, false)
	assertContainsError(t, r, "at least one letter or digit")
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

func TestDescriptionRequired(t *testing.T) {
	kf := validKyperFile()
	kf.Description = ""
	r := Validate(kf, false)
	assertContainsError(t, r, "description is required")
}

func TestDescriptionTooLong(t *testing.T) {
	kf := validKyperFile()
	kf.Description = strings.Repeat("a", 501)
	r := Validate(kf, false)
	assertContainsError(t, r, "500 characters")
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

func TestDBWithoutHookWarning(t *testing.T) {
	kf := validKyperFile()
	kf.Deps = []config.DepEntry{{Name: "postgres"}}
	r := Validate(kf, false)
	assertContainsWarning(t, r, "hooks.on_deploy")
}

func TestDBWithHookNoWarning(t *testing.T) {
	kf := validKyperFile()
	kf.Deps = []config.DepEntry{{Name: "postgres"}}
	kf.Hooks.OnDeploy = "bundle exec rails db:migrate"
	r := Validate(kf, false)
	for _, w := range r.Warnings {
		if strings.Contains(w, "hooks.on_deploy") {
			t.Error("should not warn about hooks.on_deploy when set")
		}
	}
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
