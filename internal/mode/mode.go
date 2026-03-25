package mode

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unicode"
)

type Manager struct {
	mu           sync.RWMutex
	current      string
	defaultMode  string
	modeFile     string
	notify       bool
	notifier     func(title, body string) error
	onModeChange func(mode string)
	timeout      time.Duration
	timer        *time.Timer
	stopTimer    chan struct{}
}

func NewManager(defaultMode, modeFile string, notify bool, notifier func(string, string) error) *Manager {
	m := &Manager{
		current:     defaultMode,
		defaultMode: defaultMode,
		modeFile:    modeFile,
		notify:      notify,
		notifier:    notifier,
		stopTimer:   make(chan struct{}),
	}
	m.writeModeFile()
	return m
}

// SetTimeout sets the mode timeout duration. 0 or negative disables timeout.
func (m *Manager) SetTimeout(seconds int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if seconds <= 0 {
		m.timeout = 0
		if m.timer != nil {
			m.timer.Stop()
			m.timer = nil
		}
	} else {
		m.timeout = time.Duration(seconds) * time.Second
	}
}

// ResetTimer resets the mode timeout timer. Call on any button press.
func (m *Manager) ResetTimer() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.timeout <= 0 {
		return
	}

	// Already on default mode, no need for timer
	if m.current == m.defaultMode {
		if m.timer != nil {
			m.timer.Stop()
			m.timer = nil
		}
		return
	}

	// Reset or create timer
	if m.timer != nil {
		m.timer.Stop()
	}

	m.timer = time.AfterFunc(m.timeout, m.timerFired)
}

// timerFired is called when the mode timeout expires.
// It runs in a separate goroutine (from time.AfterFunc), so it's safe to acquire the lock.
func (m *Manager) timerFired() {
	m.mu.RLock()
	current := m.current
	defaultMode := m.defaultMode
	m.mu.RUnlock()

	if current != defaultMode {
		m.Switch(defaultMode)
		fmt.Fprintf(os.Stderr, "mode timeout: switched to %s\n", defaultMode)
	}
}

func (m *Manager) Current() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

func (m *Manager) OnModeChange(fn func(mode string)) {
	m.mu.Lock()
	m.onModeChange = fn
	current := m.current
	m.mu.Unlock()

	// Notify immediately with current mode
	if fn != nil {
		fn(current)
	}
}

func (m *Manager) Switch(name string) {
	// Validate mode name: only allow printable ASCII, no control chars
	for _, r := range name {
		if !unicode.IsPrint(r) || r > 127 {
			fmt.Fprintf(os.Stderr, "invalid mode name rejected\n")
			return
		}
	}
	if len(name) > 64 {
		fmt.Fprintf(os.Stderr, "mode name too long, rejected\n")
		return
	}

	m.mu.Lock()
	if m.current == name {
		m.mu.Unlock()
		return
	}
	m.current = name
	notify := m.notify
	notifier := m.notifier
	onChange := m.onModeChange
	modeFile := m.modeFile
	timeout := m.timeout
	defaultMode := m.defaultMode

	// Stop existing timer
	if m.timer != nil {
		m.timer.Stop()
		m.timer = nil
	}

	// Start new timer if not switching to default mode
	if timeout > 0 && name != defaultMode {
		fmt.Fprintf(os.Stderr, "timer started: %v until auto-switch to %s\n", timeout, defaultMode)
		m.timer = time.AfterFunc(timeout, m.timerFired)
	}
	m.mu.Unlock()

	// Write mode file (outside lock)
	if modeFile != "" {
		if err := writeModeFileAtomic(modeFile, name); err != nil {
			fmt.Fprintf(os.Stderr, "write mode file: %v\n", err)
		}
	}

	// Send notification
	if notify && notifier != nil {
		icon := modeIcon(name)
		if err := notifier(fmt.Sprintf("%s %s", icon, name), ""); err != nil {
			fmt.Fprintf(os.Stderr, "notification error: %v\n", err)
		}
	}

	// Callback
	if onChange != nil {
		onChange(name)
	}
}

func (m *Manager) writeModeFile() {
	m.mu.RLock()
	modeFile := m.modeFile
	current := m.current
	m.mu.RUnlock()

	if modeFile == "" {
		return
	}

	if err := writeModeFileAtomic(modeFile, current); err != nil {
		fmt.Fprintf(os.Stderr, "write mode file: %v\n", err)
	}
}

// writeModeFileAtomic writes the mode file atomically via temp file + rename.
// This prevents readers (e.g. waybar) from seeing partial writes.
func writeModeFileAtomic(path, mode string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".mode-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.WriteString(mode); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Chmod(0640); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

func modeIcon(name string) string {
	switch name {
	case "navigation":
		return "🎮"
	case "launcher":
		return "🚀"
	case "input":
		return "⌨️"
	default:
		return "⚡"
	}
}
