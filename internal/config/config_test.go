package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		DefaultMode: "navigation",
		Modes: map[string]Mode{
			"navigation": {
				"up":   "sway:focus up",
				"down": "sway:focus down",
				"a":    "mode:input",
			},
			"input": {
				"up": "key:Up",
				"b":  "mode:navigation",
			},
		},
	}

	warnings := cfg.Validate()
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %d: %v", len(warnings), warnings)
	}
}

func TestValidate_UnknownButton(t *testing.T) {
	cfg := &Config{
		DefaultMode: "navigation",
		Modes: map[string]Mode{
			"navigation": {
				"invalid_button": "sway:focus up",
			},
		},
	}

	warnings := cfg.Validate()
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warnings))
	}
	if warnings[0].Button != "invalid_button" {
		t.Errorf("expected warning about invalid_button, got %s", warnings[0].Button)
	}
}

func TestValidate_UnknownActionType(t *testing.T) {
	cfg := &Config{
		DefaultMode: "navigation",
		Modes: map[string]Mode{
			"navigation": {
				"a": "invalid:command",
			},
		},
	}

	warnings := cfg.Validate()
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warnings))
	}
}

func TestValidate_InvalidActionFormat(t *testing.T) {
	cfg := &Config{
		DefaultMode: "navigation",
		Modes: map[string]Mode{
			"navigation": {
				"a": "no-colon-here",
			},
		},
	}

	warnings := cfg.Validate()
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warnings))
	}
}

func TestValidate_UndefinedModeReference(t *testing.T) {
	cfg := &Config{
		DefaultMode: "navigation",
		Modes: map[string]Mode{
			"navigation": {
				"a": "mode:nonexistent",
			},
		},
	}

	warnings := cfg.Validate()
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warnings))
	}
}

func TestValidate_UnknownMouseAction(t *testing.T) {
	cfg := &Config{
		DefaultMode: "navigation",
		Modes: map[string]Mode{
			"navigation": {
				"a": "mouse:invalid_action",
			},
		},
	}

	warnings := cfg.Validate()
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warnings))
	}
}

func TestValidate_ValidMouseActions(t *testing.T) {
	validActions := []string{
		"mouse:click_left",
		"mouse:click_right",
		"mouse:click_middle",
		"mouse:double_left",
		"mouse:hold_left",
		"mouse:release_left",
		"mouse:move_up",
		"mouse:move_up:20",
		"mouse:move_down:50",
	}

	for _, action := range validActions {
		cfg := &Config{
			DefaultMode: "navigation",
			Modes: map[string]Mode{
				"navigation": {
					"a": action,
				},
			},
		}

		warnings := cfg.Validate()
		if len(warnings) != 0 {
			t.Errorf("action %q should be valid, got warnings: %v", action, warnings)
		}
	}
}

func TestValidate_DefaultModeNotDefined(t *testing.T) {
	cfg := &Config{
		DefaultMode: "nonexistent",
		Modes: map[string]Mode{
			"navigation": {
				"a": "sway:focus up",
			},
		},
	}

	warnings := cfg.Validate()
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warnings))
	}
}

func TestValidateFileOwnership_OwnedByCurrentUser(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Write a config file owned by current user with good permissions
	if err := os.WriteFile(configFile, []byte("default_mode: navigation\n"), 0640); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := validateFileOwnership(configFile); err != nil {
		t.Errorf("expected no error for user-owned file, got: %v", err)
	}
}

func TestValidateFileOwnership_WorldWritable(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(configFile, []byte("default_mode: navigation\n"), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	// Explicitly chmod to bypass umask
	if err := os.Chmod(configFile, 0666); err != nil {
		t.Fatalf("failed to chmod: %v", err)
	}

	err := validateFileOwnership(configFile)
	if err == nil {
		t.Error("expected error for world-writable config file")
	}
}

func TestValidateFileOwnership_Symlink_NonNixStore(t *testing.T) {
	tmpDir := t.TempDir()
	realFile := filepath.Join(tmpDir, "real-config.yaml")
	symlink := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(realFile, []byte("default_mode: navigation\n"), 0640); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.Symlink(realFile, symlink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Symlinks to non-nix-store paths should be rejected
	err := validateFileOwnership(symlink)
	if err == nil {
		t.Error("expected error for symlink to non-nix-store target")
	}
}

func TestValidateFileOwnership_Symlink_NixStore(t *testing.T) {
	// On NixOS, Home Manager creates symlinks to /nix/store/. These should be allowed.
	// We can only test this if /nix/store exists (i.e., on NixOS/Nix).
	nixStore := "/nix/store"
	if _, err := os.Stat(nixStore); os.IsNotExist(err) {
		t.Skip("skipping: /nix/store not available (not a NixOS/Nix system)")
	}

	// Find any file in /nix/store to symlink to
	entries, err := os.ReadDir(nixStore)
	if err != nil || len(entries) == 0 {
		t.Skip("skipping: cannot read /nix/store")
	}

	// Use first regular file or directory entry
	var nixTarget string
	for _, e := range entries {
		candidate := filepath.Join(nixStore, e.Name())
		info, err := os.Stat(candidate)
		if err == nil && info.Mode().IsRegular() {
			nixTarget = candidate
			break
		}
	}
	if nixTarget == "" {
		// Create a symlink to a nix store directory instead
		nixTarget = filepath.Join(nixStore, entries[0].Name())
	}

	tmpDir := t.TempDir()
	symlink := filepath.Join(tmpDir, "config.yaml")
	if err := os.Symlink(nixTarget, symlink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	err = validateFileOwnership(symlink)
	if err != nil {
		t.Errorf("expected no error for symlink to /nix/store, got: %v", err)
	}
}

func TestConfigCopy(t *testing.T) {
	original := &Config{
		DefaultMode: "navigation",
		Modes: map[string]Mode{
			"navigation": {
				"a": "sway:focus up",
			},
		},
	}

	cp := original.Copy()

	// Mutate the copy
	cp.Modes["navigation"]["a"] = "MUTATED"
	cp.Modes["new_mode"] = Mode{"b": "sway:focus down"}

	// Original should be unchanged
	if original.Modes["navigation"]["a"] != "sway:focus up" {
		t.Errorf("original was mutated: a = %q", original.Modes["navigation"]["a"])
	}
	if _, ok := original.Modes["new_mode"]; ok {
		t.Error("original was mutated: new_mode exists")
	}
}

func TestValidButtons(t *testing.T) {
	buttons := []string{
		"a", "b", "x", "y", "l", "r",
		"up", "down", "left", "right",
		"select+a", "select+b", "select+x", "select+y",
		"select+l", "select+r",
		"select+up", "select+down", "select+left", "select+right",
		"start+a", "start+b", "start+x", "start+y",
		"start+l", "start+r",
		"start+up", "start+down", "start+left", "start+right",
	}

	for _, btn := range buttons {
		if !validButtons[btn] {
			t.Errorf("button %q should be valid", btn)
		}
	}
}
