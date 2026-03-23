package repeat

import (
	"testing"
)

func TestShouldRepeat_MouseMove(t *testing.T) {
	tests := []struct {
		action   string
		expected bool
	}{
		{"mouse:move_up", true},
		{"mouse:move_down", true},
		{"mouse:move_left", true},
		{"mouse:move_right", true},
		{"mouse:move_up:20", true},
		{"mouse:move_down:50", true},
		{"mouse:click_left", false},
		{"mouse:click_right", false},
		{"mouse:hold_left", false},
		{"mouse:release_left", false},
		{"sway:focus up", false},
		{"key:Up", false},
		{"mode:navigation", false},
		{"exec:firefox", false},
		{"", false},
	}

	for _, tt := range tests {
		result := ShouldRepeat(tt.action)
		if result != tt.expected {
			t.Errorf("ShouldRepeat(%q) = %v, want %v", tt.action, result, tt.expected)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.InitialDelay <= 0 {
		t.Error("InitialDelay should be positive")
	}
	if cfg.MinInterval <= 0 {
		t.Error("MinInterval should be positive")
	}
	if cfg.MaxInterval <= 0 {
		t.Error("MaxInterval should be positive")
	}
	if cfg.MinInterval >= cfg.MaxInterval {
		t.Error("MinInterval should be less than MaxInterval")
	}
	if cfg.Acceleration <= 0 || cfg.Acceleration >= 1 {
		t.Error("Acceleration should be between 0 and 1")
	}
}

func TestRepeater_PressRelease(t *testing.T) {
	executed := 0
	r := New(
		DefaultConfig(),
		func(action string) error {
			executed++
			return nil
		},
		ShouldRepeat,
	)
	defer r.Stop()

	// Press non-repeating action - should not track
	r.Press("a", "sway:focus up")
	if r.IsHeld("a") {
		t.Error("non-repeating action should not be held")
	}

	// Press repeating action
	r.Press("up", "mouse:move_up")
	if !r.IsHeld("up") {
		t.Error("repeating action should be held")
	}
	if executed != 1 {
		t.Errorf("expected 1 immediate execution, got %d", executed)
	}

	// Release
	r.Release("up")
	if r.IsHeld("up") {
		t.Error("button should not be held after release")
	}
}

func TestRepeater_Stop(t *testing.T) {
	r := New(
		DefaultConfig(),
		func(action string) error { return nil },
		ShouldRepeat,
	)

	r.Press("up", "mouse:move_up")
	r.Stop()

	if r.IsHeld("up") {
		t.Error("button should not be held after stop")
	}

	// Should not panic on second stop
	r.Stop()
}

func TestRepeater_DoublePress(t *testing.T) {
	executed := 0
	r := New(
		DefaultConfig(),
		func(action string) error {
			executed++
			return nil
		},
		ShouldRepeat,
	)
	defer r.Stop()

	r.Press("up", "mouse:move_up")
	r.Press("up", "mouse:move_up") // Second press should be ignored

	if executed != 1 {
		t.Errorf("double press should only execute once, got %d", executed)
	}
}
