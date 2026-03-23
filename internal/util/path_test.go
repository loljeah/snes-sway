package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("could not get home dir: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~/test", filepath.Join(home, "test")},
		{"~/.config/snes-sway", filepath.Join(home, ".config/snes-sway")},
	}

	for _, tt := range tests {
		result, err := ExpandPath(tt.input)
		if err != nil {
			t.Errorf("ExpandPath(%q) returned error: %v", tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestValidatePathUnderHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("could not get home dir: %v", err)
	}

	// Valid paths under home
	validPaths := []string{
		"~/.config/snes-sway/config.yaml",
		"~/.config/snes-sway/mode",
		filepath.Join(home, ".config", "test"),
	}

	for _, path := range validPaths {
		result, err := ValidatePathUnderHome(path)
		if err != nil {
			t.Errorf("ValidatePathUnderHome(%q) should be valid, got error: %v", path, err)
		}
		if !strings.HasPrefix(result, home) && result[0] != '~' {
			// Result should start with home or be tilde-expanded
			expanded, _ := ExpandPath(result)
			if !strings.HasPrefix(expanded, home) {
				t.Errorf("ValidatePathUnderHome(%q) returned %q which is not under home", path, result)
			}
		}
	}

	// Invalid paths outside home
	invalidPaths := []string{
		"/etc/passwd",
		"/tmp/test",
		"/var/log/test",
	}

	for _, path := range invalidPaths {
		_, err := ValidatePathUnderHome(path)
		if err == nil {
			t.Errorf("ValidatePathUnderHome(%q) should fail for path outside home", path)
		}
	}
}
