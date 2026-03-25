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

func TestSwitch_RejectsInvalidName(t *testing.T) {
	m := NewManager("navigation", "", false, nil)

	// Control character should be rejected
	m.Switch("nav\x00igation")
	if m.Current() != "navigation" {
		t.Errorf("expected mode unchanged after invalid name, got %q", m.Current())
	}

	// Too long name should be rejected
	longName := string(make([]byte, 65))
	for i := range longName {
		longName = longName[:i] + "a" + longName[i+1:]
	}
	m.Switch(longName)
	if m.Current() != "navigation" {
		t.Errorf("expected mode unchanged after too-long name, got %q", m.Current())
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

// waitForMode polls until the manager reaches the expected mode or times out.
// Using polling instead of fixed sleep makes tests reliable under CI load.
func waitForMode(m *Manager, expected string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if m.Current() == expected {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func TestSetTimeout(t *testing.T) {
	m := NewManager("navigation", "", false, nil)

	// Initially no timeout
	m.SetTimeout(0)

	// Set timeout (use 1 second)
	m.SetTimeout(1)

	// Switch to non-default mode
	m.Switch("launcher")
	if m.Current() != "launcher" {
		t.Errorf("expected 'launcher', got %q", m.Current())
	}

	// Wait for timeout with generous margin
	if !waitForMode(m, "navigation", 3*time.Second) {
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

	// Shortly after reset, should still be in launcher
	time.Sleep(500 * time.Millisecond)
	if m.Current() != "launcher" {
		t.Errorf("expected 'launcher' after timer reset, got %q", m.Current())
	}

	// Wait for the full reset timeout to fire
	if !waitForMode(m, "navigation", 3*time.Second) {
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
