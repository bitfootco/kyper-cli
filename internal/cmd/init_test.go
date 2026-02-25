package cmd

import "testing"

func TestSuggestHook(t *testing.T) {
	tests := []struct {
		stacks []string
		want   string
	}{
		{[]string{"rails"}, "bundle exec rails db:migrate"},
		{[]string{"django"}, "python manage.py migrate"},
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
