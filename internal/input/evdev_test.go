package input

import (
	"testing"

	evdev "github.com/holoplot/go-evdev"
)

func TestCodeToButton(t *testing.T) {
	tests := []struct {
		code     evdev.EvCode
		expected Button
	}{
		{BTN_A, ButtonA},
		{BTN_B, ButtonB},
		{BTN_X, ButtonX},
		{BTN_Y, ButtonY},
		{BTN_L, ButtonL},
		{BTN_R, ButtonR},
		{BTN_START, ButtonStart},
		{BTN_SELECT, ButtonSelect},
		{0x999, ""}, // Unknown code
	}

	for _, tt := range tests {
		result := codeToButton(tt.code)
		if result != tt.expected {
			t.Errorf("codeToButton(0x%x) = %q, want %q", tt.code, result, tt.expected)
		}
	}
}

func TestSelectChordButton(t *testing.T) {
	tests := []struct {
		btn      Button
		expected Button
	}{
		{ButtonA, ButtonSelectA},
		{ButtonB, ButtonSelectB},
		{ButtonX, ButtonSelectX},
		{ButtonY, ButtonSelectY},
		{ButtonL, ButtonSelectL},
		{ButtonR, ButtonSelectR},
		{ButtonUp, ButtonSelectUp},
		{ButtonDown, ButtonSelectDown},
		{ButtonLeft, ButtonSelectLeft},
		{ButtonRight, ButtonSelectRight},
		{ButtonStart, ""}, // No chord for start with select
		{ButtonSelect, ""}, // No chord for select with select
	}

	for _, tt := range tests {
		result := selectChordButton(tt.btn)
		if result != tt.expected {
			t.Errorf("selectChordButton(%q) = %q, want %q", tt.btn, result, tt.expected)
		}
	}
}

func TestStartChordButton(t *testing.T) {
	tests := []struct {
		btn      Button
		expected Button
	}{
		{ButtonA, ButtonStartA},
		{ButtonB, ButtonStartB},
		{ButtonX, ButtonStartX},
		{ButtonY, ButtonStartY},
		{ButtonL, ButtonStartL},
		{ButtonR, ButtonStartR},
		{ButtonUp, ButtonStartUp},
		{ButtonDown, ButtonStartDown},
		{ButtonLeft, ButtonStartLeft},
		{ButtonRight, ButtonStartRight},
		{ButtonStart, ""}, // No chord for start with start
		{ButtonSelect, ""}, // No chord for select with start
	}

	for _, tt := range tests {
		result := startChordButton(tt.btn)
		if result != tt.expected {
			t.Errorf("startChordButton(%q) = %q, want %q", tt.btn, result, tt.expected)
		}
	}
}

func TestButtonConstants(t *testing.T) {
	// Verify button codes match DaemonBite HID report
	if BTN_B != 0x120 {
		t.Errorf("BTN_B should be 0x120, got 0x%x", BTN_B)
	}
	if BTN_A != 0x121 {
		t.Errorf("BTN_A should be 0x121, got 0x%x", BTN_A)
	}
	if BTN_Y != 0x122 {
		t.Errorf("BTN_Y should be 0x122, got 0x%x", BTN_Y)
	}
	if BTN_X != 0x123 {
		t.Errorf("BTN_X should be 0x123, got 0x%x", BTN_X)
	}
	if BTN_L != 0x124 {
		t.Errorf("BTN_L should be 0x124, got 0x%x", BTN_L)
	}
	if BTN_R != 0x125 {
		t.Errorf("BTN_R should be 0x125, got 0x%x", BTN_R)
	}
	if BTN_SELECT != 0x126 {
		t.Errorf("BTN_SELECT should be 0x126, got 0x%x", BTN_SELECT)
	}
	if BTN_START != 0x127 {
		t.Errorf("BTN_START should be 0x127, got 0x%x", BTN_START)
	}
}
