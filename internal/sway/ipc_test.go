package sway

import (
	"strings"
	"testing"
)

func TestValidActionTypes(t *testing.T) {
	expected := []string{"sway", "exec", "key", "mode", "mouse"}
	for _, actionType := range expected {
		if !validActionTypes[actionType] {
			t.Errorf("expected %q to be a valid action type", actionType)
		}
	}
}

func TestExecutor_Run_EmptyAction(t *testing.T) {
	e := NewExecutor()
	err := e.Run("")
	if err != nil {
		t.Errorf("empty action should return nil, got %v", err)
	}
}

func TestExecutor_Run_InvalidFormat(t *testing.T) {
	e := NewExecutor()
	err := e.Run("no-colon-here")
	if err == nil {
		t.Error("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "invalid action format") {
		t.Errorf("expected 'invalid action format' error, got %v", err)
	}
}

func TestExecutor_Run_UnknownType(t *testing.T) {
	e := NewExecutor()
	err := e.Run("unknown:command")
	if err == nil {
		t.Error("expected error for unknown action type")
	}
	if !strings.Contains(err.Error(), "unknown action type") {
		t.Errorf("expected 'unknown action type' error, got %v", err)
	}
}

func TestExecutor_Run_ModeAction(t *testing.T) {
	e := NewExecutor()
	// mode: actions are handled by caller, executor returns nil
	err := e.Run("mode:navigation")
	if err != nil {
		t.Errorf("mode action should return nil, got %v", err)
	}
}

func TestMouseAction_UnknownAction(t *testing.T) {
	e := NewExecutor()
	err := e.mouseAction("unknown_action")
	if err == nil {
		t.Error("expected error for unknown mouse action")
	}
	if !strings.Contains(err.Error(), "unknown mouse action") {
		t.Errorf("expected 'unknown mouse action' error, got %v", err)
	}
}

func TestFindBinary(t *testing.T) {
	// Test with a binary that should exist on any system
	path := findBinary("sh", "/bin/sh", "/usr/bin/sh")
	if path == "" || path == "sh" {
		// If not found in standard locations, it should have found via PATH
		t.Log("sh not found via candidates, checking PATH fallback")
	}

	// Test with non-existent binary
	path = findBinary("nonexistent_binary_12345")
	if path != "nonexistent_binary_12345" {
		t.Errorf("expected fallback to name, got %q", path)
	}
}

func TestNixProfileBin(t *testing.T) {
	path := nixProfileBin("test-binary")
	if path == "" {
		t.Error("expected non-empty path")
	}
	if !strings.Contains(path, ".nix-profile/bin/test-binary") {
		t.Errorf("expected path to contain .nix-profile/bin/test-binary, got %q", path)
	}
}

func TestDotool_ValidCommands(t *testing.T) {
	validCommands := []string{
		"buttondown left",
		"buttondown right",
		"buttonup left",
		"buttonup right",
	}

	for _, cmd := range validCommands {
		if !validDotoolCommands[cmd] {
			t.Errorf("expected %q to be a valid dotool command", cmd)
		}
	}
}

func TestDotool_RejectsUnknownCommands(t *testing.T) {
	e := NewExecutor()

	invalidCommands := []string{
		"key a",
		"type hello",
		"exec rm -rf /",
		"buttondown middle",
	}

	for _, cmd := range invalidCommands {
		err := e.dotool(cmd)
		if err == nil {
			t.Errorf("expected error for invalid dotool command %q", cmd)
		}
		if !strings.Contains(err.Error(), "rejected unknown command") {
			t.Errorf("expected 'rejected unknown command' error for %q, got: %v", cmd, err)
		}
	}
}
