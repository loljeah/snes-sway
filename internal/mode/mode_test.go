package mode

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	m := NewManager("navigation", "", false, nil)
	if m.Current() != "navigation" {
		t.Errorf("expected current mode 'navigation', got %q", m.Current())
	}
}

func TestSwitch(t *testing.T) {
	switched := ""
	m := NewManager("navigation", "", false, nil)
	m.OnModeChange(func(mode string) {
		switched = mode
	})

	m.Switch("launcher")
	if m.Current() != "launcher" {
		t.Errorf("expected current mode 'launcher', got %q", m.Current())
	}
	if switched != "launcher" {
		t.Errorf("expected callback with 'launcher', got %q", switched)
	}

	// Switch to same mode should not trigger callback
	switched = ""
	m.Switch("launcher")
	if switched != "" {
		t.Errorf("expected no callback on same mode switch, got %q", switched)
	}
}

func TestModeFile(t *testing.T) {
	tmpDir := t.TempDir()
	modeFile := filepath.Join(tmpDir, "mode")

	m := NewManager("navigation", modeFile, false, nil)

	// Check file was created
	content, err := os.ReadFile(modeFile)
	if err != nil {
		t.Fatalf("failed to read mode file: %v", err)
	}
	if string(content) != "navigation" {
		t.Errorf("expected mode file content 'navigation', got %q", string(content))
	}

	// Check file permissions
	info, err := os.Stat(modeFile)
	if err != nil {
		t.Fatalf("failed to stat mode file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0640 {
		t.Errorf("expected mode file permissions 0640, got %04o", perm)
	}

	// Switch and check file updated
	m.Switch("launcher")
	content, err = os.ReadFile(modeFile)
	if err != nil {
		t.Fatalf("failed to read mode file: %v", err)
	}
	if string(content) != "launcher" {
		t.Errorf("expected mode file content 'launcher', got %q", string(content))
	}
}

func TestSetTimeout(t *testing.T) {
	m := NewManager("navigation", "", false, nil)

	// Initially no timeout
	m.SetTimeout(0)

	// Set timeout
	m.SetTimeout(1)

	// Switch to non-default mode
	m.Switch("launcher")
	if m.Current() != "launcher" {
		t.Errorf("expected 'launcher', got %q", m.Current())
	}

	// Wait for timeout
	time.Sleep(1500 * time.Millisecond)

	if m.Current() != "navigation" {
		t.Errorf("expected timeout switch to 'navigation', got %q", m.Current())
	}
}

func TestResetTimer(t *testing.T) {
	m := NewManager("navigation", "", false, nil)
	m.SetTimeout(1)

	m.Switch("launcher")

	// Reset timer before timeout
	time.Sleep(500 * time.Millisecond)
	m.ResetTimer()

	// Wait less than full timeout
	time.Sleep(700 * time.Millisecond)

	// Should still be in launcher (timer was reset)
	if m.Current() != "launcher" {
		t.Errorf("expected 'launcher' after timer reset, got %q", m.Current())
	}

	// Wait for remaining timeout
	time.Sleep(500 * time.Millisecond)

	if m.Current() != "navigation" {
		t.Errorf("expected 'navigation' after full timeout, got %q", m.Current())
	}
}

func TestModeIcon(t *testing.T) {
	tests := []struct {
		mode     string
		expected string
	}{
		{"navigation", "🎮"},
		{"launcher", "🚀"},
		{"input", "⌨️"},
		{"unknown", "⚡"},
	}

	for _, tt := range tests {
		result := modeIcon(tt.mode)
		if result != tt.expected {
			t.Errorf("modeIcon(%q) = %q, want %q", tt.mode, result, tt.expected)
		}
	}
}
