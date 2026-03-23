package config

import (
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
