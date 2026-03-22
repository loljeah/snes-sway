package repeat

import (
	"strings"
	"sync"
	"time"
)

// Config for repeat behavior
type Config struct {
	InitialDelay time.Duration // Delay before repeat starts
	MinInterval  time.Duration // Fastest repeat rate
	MaxInterval  time.Duration // Slowest repeat rate (start)
	Acceleration float64       // How fast to accelerate (0.8 = 20% faster each repeat)
}

// DefaultConfig returns sensible defaults for mouse movement
func DefaultConfig() Config {
	return Config{
		InitialDelay: 150 * time.Millisecond,
		MinInterval:  16 * time.Millisecond, // ~60 FPS
		MaxInterval:  50 * time.Millisecond, // Start slower
		Acceleration: 0.85,                  // Speed up by 15% each repeat
	}
}

// Repeater handles button hold repeating with acceleration
type Repeater struct {
	mu       sync.Mutex
	config   Config
	held     map[string]*holdState
	executor func(action string) error
	isRepeat func(action string) bool
	stop     chan struct{}
	stopped  bool
}

type holdState struct {
	action    string
	startTime time.Time
	interval  time.Duration
	timer     *time.Timer
	stop      chan struct{}
}

// New creates a repeater
// executor: function to run actions
// isRepeat: function to check if action should repeat (e.g., mouse moves)
func New(cfg Config, executor func(string) error, isRepeat func(string) bool) *Repeater {
	return &Repeater{
		config:   cfg,
		held:     make(map[string]*holdState),
		executor: executor,
		isRepeat: isRepeat,
		stop:     make(chan struct{}),
	}
}

// Press handles button press - starts repeat if applicable
func (r *Repeater) Press(button, action string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.stopped {
		return
	}

	// Already held
	if _, ok := r.held[button]; ok {
		return
	}

	// Check if this action should repeat
	if !r.isRepeat(action) {
		return
	}

	// Execute immediately
	r.executor(action)

	// Start hold state
	hs := &holdState{
		action:    action,
		startTime: time.Now(),
		interval:  r.config.MaxInterval,
		stop:      make(chan struct{}),
	}
	r.held[button] = hs

	// Start repeat goroutine after initial delay
	go r.repeatLoop(button, hs)
}

// Release handles button release - stops repeat
func (r *Repeater) Release(button string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if hs, ok := r.held[button]; ok {
		close(hs.stop)
		delete(r.held, button)
	}
}

// UpdateAction updates the action for a held button (e.g., when mode changes)
func (r *Repeater) UpdateAction(button, action string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if hs, ok := r.held[button]; ok {
		hs.action = action
	}
}

// Stop stops all repeating
func (r *Repeater) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.stopped {
		return
	}
	r.stopped = true

	for button, hs := range r.held {
		close(hs.stop)
		delete(r.held, button)
	}
	close(r.stop)
}

func (r *Repeater) repeatLoop(button string, hs *holdState) {
	// Wait for initial delay
	select {
	case <-time.After(r.config.InitialDelay):
	case <-hs.stop:
		return
	case <-r.stop:
		return
	}

	for {
		select {
		case <-hs.stop:
			return
		case <-r.stop:
			return
		case <-time.After(hs.interval):
			r.mu.Lock()
			// Check if still held
			currentHs, ok := r.held[button]
			if !ok || currentHs != hs {
				r.mu.Unlock()
				return
			}
			action := hs.action
			r.mu.Unlock()

			// Execute action
			r.executor(action)

			// Accelerate
			r.mu.Lock()
			if currentHs, ok := r.held[button]; ok && currentHs == hs {
				newInterval := time.Duration(float64(hs.interval) * r.config.Acceleration)
				if newInterval < r.config.MinInterval {
					newInterval = r.config.MinInterval
				}
				hs.interval = newInterval
			}
			r.mu.Unlock()
		}
	}
}

// IsHeld checks if a button is currently held
func (r *Repeater) IsHeld(button string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.held[button]
	return ok
}

// ShouldRepeat is a helper to check if an action should repeat
// Returns true for mouse movement actions
func ShouldRepeat(action string) bool {
	if !strings.HasPrefix(action, "mouse:") {
		return false
	}
	cmd := strings.TrimPrefix(action, "mouse:")
	// Only repeat movement, not clicks
	return strings.HasPrefix(cmd, "move_")
}
