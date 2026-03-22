package input

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	evdev "github.com/holoplot/go-evdev"
)

// SNES button mapping from DaemonBite firmware
// Buttons are sent as BTN_TRIGGER, BTN_THUMB, etc. (codes 0x120-0x127)
// D-pad is sent as ABS_X (-1/0/+1) and ABS_Y (-1/0/+1)
const (
	// Button codes (EV_KEY) - mapped from DaemonBite HID report
	// Bit order in buttons byte: B(0), A(1), Y(2), X(3), L(4), R(5), Select(6), Start(7)
	BTN_B      = 0x120 // BTN_TRIGGER - bit 0
	BTN_A      = 0x121 // BTN_THUMB - bit 1
	BTN_Y      = 0x122 // BTN_THUMB2 - bit 2
	BTN_X      = 0x123 // BTN_TOP - bit 3
	BTN_L      = 0x124 // BTN_TOP2 - bit 4
	BTN_R      = 0x125 // BTN_PINKIE - bit 5
	BTN_SELECT = 0x126 // BTN_BASE - bit 6
	BTN_START  = 0x127 // BTN_BASE2 - bit 7
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

	// Chord buttons (Start + button)
	ButtonStartL     Button = "start+l"
	ButtonStartR     Button = "start+r"
	ButtonStartUp    Button = "start+up"
	ButtonStartDown  Button = "start+down"
	ButtonStartLeft  Button = "start+left"
	ButtonStartRight Button = "start+right"
	ButtonStartA     Button = "start+a"
	ButtonStartB     Button = "start+b"
	ButtonStartX     Button = "start+x"
	ButtonStartY     Button = "start+y"
)

type Event struct {
	Button  Button
	Pressed bool
}

type Reader struct {
	mu           sync.Mutex
	dev          *evdev.InputDevice
	events       chan Event
	stop         chan struct{}
	lastX        int32
	lastY        int32
	selectHeld   bool
	startHeld    bool
	chordUsed    bool
	closed       bool
	disconnected chan struct{}
}

func FindDevice(vendorID, productID uint16) (string, error) {
	matches, _ := filepath.Glob("/dev/input/event*")

	for _, path := range matches {
		dev, err := evdev.Open(path)
		if err != nil {
			continue
		}

		id, err := dev.InputID()
		if err != nil {
			dev.Close() // Close on error
			continue
		}

		if id.Vendor == vendorID && id.Product == productID {
			dev.Close() // Close after checking, will reopen in NewReader
			return path, nil
		}
		dev.Close() // Always close
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
		dev:          dev,
		events:       make(chan Event, 64), // Buffered to prevent blocking
		stop:         make(chan struct{}),
		disconnected: make(chan struct{}),
	}, nil
}

func (r *Reader) Events() <-chan Event {
	return r.events
}

func (r *Reader) Disconnected() <-chan struct{} {
	return r.disconnected
}

func (r *Reader) Run() {
	defer func() {
		close(r.events)
		close(r.disconnected)
	}()

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

		// Track modifier states (Select and Start)
		if btn == ButtonSelect {
			if pressed {
				r.selectHeld = true
				r.chordUsed = false
			} else {
				r.selectHeld = false
			}
			// Select never emits events - only used as modifier
			return
		}

		if btn == ButtonStart {
			if pressed {
				r.startHeld = true
				r.chordUsed = false
			} else {
				r.startHeld = false
			}
			// Start never emits events - only used as modifier
			return
		}

		// Check for chord (Select + button)
		if r.selectHeld {
			chordBtn := selectChordButton(btn)
			if chordBtn != "" {
				if pressed {
					r.chordUsed = true
					r.sendEvent(Event{Button: chordBtn, Pressed: true})
				} else if r.chordUsed {
					r.sendEvent(Event{Button: chordBtn, Pressed: false})
				}
				return
			}
		}

		// Check for chord (Start + button)
		if r.startHeld {
			chordBtn := startChordButton(btn)
			if chordBtn != "" {
				if pressed {
					r.chordUsed = true
					r.sendEvent(Event{Button: chordBtn, Pressed: true})
				} else if r.chordUsed {
					r.sendEvent(Event{Button: chordBtn, Pressed: false})
				}
				return
			}
		}

		// Normal button event (only when no modifier is held)
		r.sendEvent(Event{Button: btn, Pressed: pressed})

	case evdev.EV_ABS:
		switch ev.Code {
		case evdev.ABS_X:
			if r.selectHeld {
				r.handleChordAxis(&r.lastX, ev.Value, ButtonSelectLeft, ButtonSelectRight)
			} else if r.startHeld {
				r.handleChordAxis(&r.lastX, ev.Value, ButtonStartLeft, ButtonStartRight)
			} else {
				r.handleAxis(&r.lastX, ev.Value, ButtonLeft, ButtonRight)
			}
		case evdev.ABS_Y:
			if r.selectHeld {
				r.handleChordAxis(&r.lastY, ev.Value, ButtonSelectUp, ButtonSelectDown)
			} else if r.startHeld {
				r.handleChordAxis(&r.lastY, ev.Value, ButtonStartUp, ButtonStartDown)
			} else {
				r.handleAxis(&r.lastY, ev.Value, ButtonUp, ButtonDown)
			}
		}
	}
}

// sendEvent sends an event without blocking; drops if buffer full
func (r *Reader) sendEvent(ev Event) {
	select {
	case r.events <- ev:
	default:
		// Buffer full, drop event to prevent blocking
		fmt.Fprintf(os.Stderr, "warning: event buffer full, dropping %s\n", ev.Button)
	}
}

func selectChordButton(btn Button) Button {
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

func startChordButton(btn Button) Button {
	switch btn {
	case ButtonL:
		return ButtonStartL
	case ButtonR:
		return ButtonStartR
	case ButtonA:
		return ButtonStartA
	case ButtonB:
		return ButtonStartB
	case ButtonX:
		return ButtonStartX
	case ButtonY:
		return ButtonStartY
	case ButtonUp:
		return ButtonStartUp
	case ButtonDown:
		return ButtonStartDown
	case ButtonLeft:
		return ButtonStartLeft
	case ButtonRight:
		return ButtonStartRight
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
		r.sendEvent(Event{Button: neg, Pressed: false})
	}
	if prev == 1 && value != 1 {
		r.sendEvent(Event{Button: pos, Pressed: false})
	}

	// Pressed
	if value == -1 && prev != -1 {
		r.sendEvent(Event{Button: neg, Pressed: true})
	}
	if value == 1 && prev != 1 {
		r.sendEvent(Event{Button: pos, Pressed: true})
	}
}

func (r *Reader) handleAxis(last *int32, value int32, neg, pos Button) {
	prev := *last
	*last = value

	// Released
	if prev == -1 && value != -1 {
		r.sendEvent(Event{Button: neg, Pressed: false})
	}
	if prev == 1 && value != 1 {
		r.sendEvent(Event{Button: pos, Pressed: false})
	}

	// Pressed
	if value == -1 && prev != -1 {
		r.sendEvent(Event{Button: neg, Pressed: true})
	}
	if value == 1 && prev != 1 {
		r.sendEvent(Event{Button: pos, Pressed: true})
	}
}

func (r *Reader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}
	r.closed = true

	close(r.stop)

	// Give the reader goroutine time to exit
	time.Sleep(10 * time.Millisecond)

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
