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
			continue
		}
		if result == "" {
			t.Errorf("ValidatePathUnderHome(%q) returned empty string", path)
			continue
		}
		expanded, _ := ExpandPath(result)
		if !strings.HasPrefix(expanded, home) {
			t.Errorf("ValidatePathUnderHome(%q) returned %q which is not under home", path, result)
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

func TestValidatePathUnderHome_TraversalAttacks(t *testing.T) {
	// Attempts to escape home via ../
	traversalPaths := []string{
		"~/../../etc/passwd",
		"~/../../../tmp/evil",
		"~/safe/../../../../../../etc/shadow",
	}

	for _, path := range traversalPaths {
		_, err := ValidatePathUnderHome(path)
		if err == nil {
			t.Errorf("ValidatePathUnderHome(%q) should reject traversal attack", path)
		}
	}
}

func TestValidatePathUnderHome_PrefixBypass(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("could not get home dir: %v", err)
	}

	// Construct a path that has home as a prefix but is actually a different directory
	// e.g., if home is /home/ljsm, test /home/ljsm_evil/...
	fakeHome := home + "_evil/config"

	_, err = ValidatePathUnderHome(fakeHome)
	if err == nil {
		t.Errorf("ValidatePathUnderHome(%q) should reject path with home as prefix but different directory", fakeHome)
	}
}

func TestValidatePathUnderHome_NixOSSymlinks(t *testing.T) {
	// On NixOS, Home Manager creates symlinks from ~/.config/ to /nix/store/.
	// ValidatePathUnderHome checks the logical path (filepath.Clean), not the
	// symlink target. Paths under home should be accepted regardless of symlinks.
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("could not get home dir: %v", err)
	}

	configPath := filepath.Join(home, ".config", "snes-sway", "config.yaml")
	result, err := ValidatePathUnderHome(configPath)
	if err != nil {
		t.Errorf("ValidatePathUnderHome(%q) should accept path under home: %v", configPath, err)
	}
	if result == "" {
		t.Errorf("ValidatePathUnderHome(%q) returned empty string", configPath)
	}
}
