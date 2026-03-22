package input

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	evdev "github.com/holoplot/go-evdev"
)

// SNES button mapping from DaemonBite firmware
// Buttons are sent as BTN_TRIGGER, BTN_THUMB, etc. (codes 0x120-0x127)
// D-pad is sent as ABS_X (-1/0/+1) and ABS_Y (-1/0/+1)
const (
	// Button codes (EV_KEY)
	BTN_B      = 0x120 // BTN_TRIGGER
	BTN_Y      = 0x121 // BTN_THUMB
	BTN_SELECT = 0x122 // BTN_THUMB2
	BTN_START  = 0x123 // BTN_TOP
	BTN_A      = 0x124 // BTN_TOP2
	BTN_X      = 0x125 // BTN_PINKIE
	BTN_L      = 0x126 // BTN_BASE
	BTN_R      = 0x127 // BTN_BASE2
)

type Button string

const (
	ButtonA      Button = "a"
	ButtonB      Button = "b"
	ButtonX      Button = "x"
	ButtonY      Button = "y"
	ButtonL      Button = "l"
	ButtonR      Button = "r"
	ButtonStart  Button = "start"
	ButtonSelect Button = "select"
	ButtonUp     Button = "up"
	ButtonDown   Button = "down"
	ButtonLeft   Button = "left"
	ButtonRight  Button = "right"

	// Chord buttons (Select + button)
	ButtonSelectL     Button = "select+l"
	ButtonSelectR     Button = "select+r"
	ButtonSelectUp    Button = "select+up"
	ButtonSelectDown  Button = "select+down"
	ButtonSelectLeft  Button = "select+left"
	ButtonSelectRight Button = "select+right"
	ButtonSelectA     Button = "select+a"
	ButtonSelectB     Button = "select+b"
	ButtonSelectX     Button = "select+x"
	ButtonSelectY     Button = "select+y"
)

type Event struct {
	Button  Button
	Pressed bool
}

type Reader struct {
	dev          *evdev.InputDevice
	events       chan Event
	stop         chan struct{}
	lastX        int32
	lastY        int32
	selectHeld   bool
	chordUsed    bool // Track if Select was used as modifier
}

func FindDevice(vendorID, productID uint16) (string, error) {
	matches, _ := filepath.Glob("/dev/input/event*")

	for _, path := range matches {
		dev, err := evdev.Open(path)
		if err != nil {
			continue
		}

		id, err := dev.InputID()
		dev.Close()
		if err != nil {
			continue
		}

		if id.Vendor == vendorID && id.Product == productID {
			return path, nil
		}
	}

	return "", fmt.Errorf("device %04x:%04x not found", vendorID, productID)
}

func NewReader(path string) (*Reader, error) {
	dev, err := evdev.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open device: %w", err)
	}

	name, _ := dev.Name()
	fmt.Fprintf(os.Stderr, "opened device: %s (%s)\n", path, strings.TrimSpace(name))

	return &Reader{
		dev:    dev,
		events: make(chan Event, 32),
		stop:   make(chan struct{}),
	}, nil
}

func (r *Reader) Events() <-chan Event {
	return r.events
}

func (r *Reader) Run() {
	defer close(r.events)

	for {
		select {
		case <-r.stop:
			return
		default:
		}

		ev, err := r.dev.ReadOne()
		if err != nil {
			fmt.Fprintf(os.Stderr, "read error: %v\n", err)
			return
		}

		r.handleEvent(*ev)
	}
}

func (r *Reader) handleEvent(ev evdev.InputEvent) {
	switch ev.Type {
	case evdev.EV_KEY:
		btn := codeToButton(ev.Code)
		if btn == "" {
			return
		}

		pressed := ev.Value == 1

		// Track Select state for chords
		if btn == ButtonSelect {
			if pressed {
				r.selectHeld = true
				r.chordUsed = false
			} else {
				r.selectHeld = false
				// Only emit Select release if it wasn't used as a chord modifier
				if !r.chordUsed {
					r.events <- Event{Button: btn, Pressed: false}
				}
				return
			}
			// Don't emit Select press immediately - wait to see if it's a chord
			return
		}

		// Check for chord (Select + button)
		if r.selectHeld {
			chordBtn := chordButton(btn)
			if chordBtn != "" {
				if pressed {
					r.chordUsed = true
					r.events <- Event{Button: chordBtn, Pressed: true}
				} else if r.chordUsed {
					// Only emit chord release if we emitted a chord press
					r.events <- Event{Button: chordBtn, Pressed: false}
				}
				// Don't emit normal button when Select is held
				return
			}
		}

		// Normal button event (only when Select is NOT held)
		r.events <- Event{Button: btn, Pressed: pressed}

	case evdev.EV_ABS:
		switch ev.Code {
		case evdev.ABS_X:
			if r.selectHeld {
				r.handleChordAxis(&r.lastX, ev.Value, ButtonSelectLeft, ButtonSelectRight)
			} else {
				r.handleAxis(&r.lastX, ev.Value, ButtonLeft, ButtonRight)
			}
		case evdev.ABS_Y:
			if r.selectHeld {
				r.handleChordAxis(&r.lastY, ev.Value, ButtonSelectUp, ButtonSelectDown)
			} else {
				r.handleAxis(&r.lastY, ev.Value, ButtonUp, ButtonDown)
			}
		}
	}
}

func chordButton(btn Button) Button {
	switch btn {
	case ButtonL:
		return ButtonSelectL
	case ButtonR:
		return ButtonSelectR
	case ButtonA:
		return ButtonSelectA
	case ButtonB:
		return ButtonSelectB
	case ButtonX:
		return ButtonSelectX
	case ButtonY:
		return ButtonSelectY
	case ButtonUp:
		return ButtonSelectUp
	case ButtonDown:
		return ButtonSelectDown
	case ButtonLeft:
		return ButtonSelectLeft
	case ButtonRight:
		return ButtonSelectRight
	default:
		return ""
	}
}

func (r *Reader) handleChordAxis(last *int32, value int32, neg, pos Button) {
	prev := *last
	*last = value

	if value != prev {
		r.chordUsed = true
	}

	// Released
	if prev == -1 && value != -1 {
		r.events <- Event{Button: neg, Pressed: false}
	}
	if prev == 1 && value != 1 {
		r.events <- Event{Button: pos, Pressed: false}
	}

	// Pressed
	if value == -1 && prev != -1 {
		r.events <- Event{Button: neg, Pressed: true}
	}
	if value == 1 && prev != 1 {
		r.events <- Event{Button: pos, Pressed: true}
	}
}

func (r *Reader) handleAxis(last *int32, value int32, neg, pos Button) {
	prev := *last
	*last = value

	// Released
	if prev == -1 && value != -1 {
		r.events <- Event{Button: neg, Pressed: false}
	}
	if prev == 1 && value != 1 {
		r.events <- Event{Button: pos, Pressed: false}
	}

	// Pressed
	if value == -1 && prev != -1 {
		r.events <- Event{Button: neg, Pressed: true}
	}
	if value == 1 && prev != 1 {
		r.events <- Event{Button: pos, Pressed: true}
	}
}

func (r *Reader) Close() error {
	close(r.stop)
	return r.dev.Close()
}

func codeToButton(code evdev.EvCode) Button {
	switch code {
	case BTN_A:
		return ButtonA
	case BTN_B:
		return ButtonB
	case BTN_X:
		return ButtonX
	case BTN_Y:
		return ButtonY
	case BTN_L:
		return ButtonL
	case BTN_R:
		return ButtonR
	case BTN_START:
		return ButtonStart
	case BTN_SELECT:
		return ButtonSelect
	default:
		return ""
	}
}
