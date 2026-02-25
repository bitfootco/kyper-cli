package cmd

import (
	"testing"

	"github.com/bitfootco/kyper-cli/internal/config"
)

func TestDerivePricingType(t *testing.T) {
	tests := []struct {
		name     string
		oneTime  *float64
		sub      *float64
		expected string
	}{
		{"both", floatPtr(29.99), floatPtr(9.99), "both"},
		{"one_time only", floatPtr(29.99), nil, "one_time"},
		{"subscription only", nil, floatPtr(9.99), "subscription"},
		{"neither", nil, nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kf := &config.KyperFile{
				Pricing: config.PricingConfig{
					OneTime:      tt.oneTime,
					Subscription: tt.sub,
				},
			}
			got := derivePricingType(kf)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestHumanizeBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{10485760, "10.0 MB"},
	}

	for _, tt := range tests {
		got := humanizeBytes(tt.input)
		if got != tt.expected {
			t.Errorf("humanizeBytes(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestBuildAppParams(t *testing.T) {
	price := 29.99
	kf := &config.KyperFile{
		Title:       "My App",
		Description: "A test app",
		Category:    "productivity",
		Tagline:     "Short pitch",
		Pricing: config.PricingConfig{
			OneTime: &price,
		},
		Processes: map[string]string{"web": "bin/start"},
	}

	params := buildAppParams(kf)
	if params["title"] != "My App" {
		t.Errorf("expected title 'My App', got %v", params["title"])
	}
	if params["pricing_type"] != "one_time" {
		t.Errorf("expected pricing_type 'one_time', got %v", params["pricing_type"])
	}
	if params["one_time_price_cents"] != 2999 {
		t.Errorf("expected 2999 cents, got %v", params["one_time_price_cents"])
	}
	if params["tagline"] != "Short pitch" {
		t.Errorf("expected tagline 'Short pitch', got %v", params["tagline"])
	}
}

func floatPtr(f float64) *float64 {
	return &f
}
