package cmd

import "testing"

func TestSlugFromTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"My App", "my-app"},
		{"my-app", "my-app"},
		{"My Cool App!!!", "my-cool-app"},
		{"  hello  world  ", "hello-world"},
		{"UPPERCASE", "uppercase"},
		{"with_underscores", "with-underscores"},
		{"simple", "simple"},
		{"123-numbers", "123-numbers"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := slugFromTitle(tt.input)
			if got != tt.want {
				t.Errorf("slugFromTitle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
