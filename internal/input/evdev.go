package input

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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
	wg           sync.WaitGroup
}

func FindDevice(vendorID, productID uint16) (string, error) {
	matches, err := filepath.Glob("/dev/input/event*")
	if err != nil {
		return "", fmt.Errorf("glob input devices: %w", err)
	}

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
	r.wg.Add(1)
	defer func() {
		r.wg.Done()
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
	r.mu.Lock()
	defer r.mu.Unlock()

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
					r.sendEventLocked(Event{Button: chordBtn, Pressed: true})
				} else if r.chordUsed {
					r.sendEventLocked(Event{Button: chordBtn, Pressed: false})
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
					r.sendEventLocked(Event{Button: chordBtn, Pressed: true})
				} else if r.chordUsed {
					r.sendEventLocked(Event{Button: chordBtn, Pressed: false})
				}
				return
			}
		}

		// Normal button event (only when no modifier is held)
		r.sendEventLocked(Event{Button: btn, Pressed: pressed})

	case evdev.EV_ABS:
		switch ev.Code {
		case evdev.ABS_X:
			if r.selectHeld {
				r.handleChordAxisLocked(&r.lastX, ev.Value, ButtonSelectLeft, ButtonSelectRight)
			} else if r.startHeld {
				r.handleChordAxisLocked(&r.lastX, ev.Value, ButtonStartLeft, ButtonStartRight)
			} else {
				r.handleAxisLocked(&r.lastX, ev.Value, ButtonLeft, ButtonRight)
			}
		case evdev.ABS_Y:
			if r.selectHeld {
				r.handleChordAxisLocked(&r.lastY, ev.Value, ButtonSelectUp, ButtonSelectDown)
			} else if r.startHeld {
				r.handleChordAxisLocked(&r.lastY, ev.Value, ButtonStartUp, ButtonStartDown)
			} else {
				r.handleAxisLocked(&r.lastY, ev.Value, ButtonUp, ButtonDown)
			}
		}
	}
}

// eventDrops tracks dropped events for monitoring (accessed atomically)
var eventDrops uint64

// sendEventLocked sends an event without blocking; drops if buffer full
// Must be called with r.mu held
func (r *Reader) sendEventLocked(ev Event) {
	select {
	case r.events <- ev:
	default:
		// Buffer full, drop event to prevent blocking
		drops := atomic.AddUint64(&eventDrops, 1)
		fmt.Fprintf(os.Stderr, "warning: event buffer full, dropping %s (total drops: %d)\n", ev.Button, drops)
	}
}

// EventDrops returns the total number of dropped events
func EventDrops() uint64 {
	return atomic.LoadUint64(&eventDrops)
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

// handleChordAxisLocked handles axis events with chord modifier
// Must be called with r.mu held
func (r *Reader) handleChordAxisLocked(last *int32, value int32, neg, pos Button) {
	prev := *last
	*last = value

	if value != prev {
		r.chordUsed = true
	}

	// Released
	if prev == -1 && value != -1 {
		r.sendEventLocked(Event{Button: neg, Pressed: false})
	}
	if prev == 1 && value != 1 {
		r.sendEventLocked(Event{Button: pos, Pressed: false})
	}

	// Pressed
	if value == -1 && prev != -1 {
		r.sendEventLocked(Event{Button: neg, Pressed: true})
	}
	if value == 1 && prev != 1 {
		r.sendEventLocked(Event{Button: pos, Pressed: true})
	}
}

// handleAxisLocked handles axis events without modifier
// Must be called with r.mu held
func (r *Reader) handleAxisLocked(last *int32, value int32, neg, pos Button) {
	prev := *last
	*last = value

	// Released
	if prev == -1 && value != -1 {
		r.sendEventLocked(Event{Button: neg, Pressed: false})
	}
	if prev == 1 && value != 1 {
		r.sendEventLocked(Event{Button: pos, Pressed: false})
	}

	// Pressed
	if value == -1 && prev != -1 {
		r.sendEventLocked(Event{Button: neg, Pressed: true})
	}
	if value == 1 && prev != 1 {
		r.sendEventLocked(Event{Button: pos, Pressed: true})
	}
}

func (r *Reader) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	close(r.stop)
	r.mu.Unlock()

	// Wait for reader goroutine to exit (with timeout)
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		fmt.Fprintf(os.Stderr, "warning: reader goroutine did not exit in time\n")
	}

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
