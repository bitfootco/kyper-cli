package cmd

import "testing"

func TestBuildKyperFileResourceTiers(t *testing.T) {
	tests := []struct {
		name      string
		tier      string
		wantMemMB int
		wantCPU   float64
	}{
		{"hobby", "hobby", 512, 0.1},
		{"basic", "starter", 512, 0.25},
		{"pro", "standard", 1024, 0.5},
		{"turbo", "pro", 2048, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kf := buildKyperFile(
				"test-app",
				"",
				"productivity",
				map[string]string{"web": "bin/start"},
				nil,
				nil,
				"",
				"/up",
				"",
				"9.99",
				tt.tier,
			)

			if kf.Resources.MinMemoryMB != tt.wantMemMB {
				t.Errorf("expected min_memory_mb %d, got %d", tt.wantMemMB, kf.Resources.MinMemoryMB)
			}
			if kf.Resources.MinCPU != tt.wantCPU {
				t.Errorf("expected min_cpu %v, got %v", tt.wantCPU, kf.Resources.MinCPU)
			}
		})
	}
}

func TestBuildKyperFileResourceTierDefaultsToHobby(t *testing.T) {
	kf := buildKyperFile(
		"test-app",
		"",
		"productivity",
		map[string]string{"web": "bin/start"},
		nil,
		nil,
		"",
		"/up",
		"",
		"9.99",
		"",
	)

	if kf.Resources.MinMemoryMB != 512 {
		t.Errorf("expected default min_memory_mb 512, got %d", kf.Resources.MinMemoryMB)
	}
	if kf.Resources.MinCPU != 0.1 {
		t.Errorf("expected default min_cpu 0.1, got %v", kf.Resources.MinCPU)
	}
}

func TestSuggestHook(t *testing.T) {
	tests := []struct {
		stacks []string
		want   string
	}{
		// suggestHook returns idempotent release-hook commands suitable for
		// cold-start (every kyper test deploy + every subscription's first
		// provision starts with an empty database).
		{[]string{"rails"}, "bundle exec rails db:prepare"},
		{[]string{"django"}, "python manage.py migrate --noinput"},
		{[]string{"prisma", "next"}, "npx prisma migrate deploy"},
		{[]string{"laravel"}, "php artisan migrate --force"},
		{[]string{"go"}, ""},
		{nil, ""},
	}

	for _, tt := range tests {
		got := suggestHook(tt.stacks)
		if got != tt.want {
			t.Errorf("suggestHook(%v) = %q, want %q", tt.stacks, got, tt.want)
		}
	}
}

func TestDefaultHealthPath(t *testing.T) {
	tests := []struct {
		stacks []string
		want   string
	}{
		{[]string{"rails"}, "/up"},
		{[]string{"django"}, "/health/"},
		{[]string{"express"}, "/health"},
		{[]string{"next"}, "/health"},
		{nil, "/up"},
	}

	for _, tt := range tests {
		got := defaultHealthPath(tt.stacks)
		if got != tt.want {
			t.Errorf("defaultHealthPath(%v) = %q, want %q", tt.stacks, got, tt.want)
		}
	}
}

func TestCategoryOptions(t *testing.T) {
	opts := categoryOptions()
	if len(opts) != 9 {
		t.Errorf("expected 9 category options, got %d", len(opts))
	}
}
