package mode

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
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

	m.timer = time.AfterFunc(m.timeout, func() {
		m.mu.Lock()
		if m.current != m.defaultMode {
			m.mu.Unlock()
			m.Switch(m.defaultMode)
			fmt.Fprintf(os.Stderr, "mode timeout: switched to %s\n", m.defaultMode)
		} else {
			m.mu.Unlock()
		}
	})
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
		m.timer = time.AfterFunc(timeout, func() {
			m.mu.Lock()
			if m.current != m.defaultMode {
				m.mu.Unlock()
				m.Switch(m.defaultMode)
				fmt.Fprintf(os.Stderr, "mode timeout: switched to %s\n", m.defaultMode)
			} else {
				m.mu.Unlock()
			}
		})
	}
	m.mu.Unlock()

	// Write mode file (outside lock)
	if modeFile != "" {
		if err := m.writeModeFileSync(modeFile, name); err != nil {
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

	if err := m.writeModeFileSync(modeFile, current); err != nil {
		fmt.Fprintf(os.Stderr, "write mode file: %v\n", err)
	}
}

func (m *Manager) writeModeFileSync(path, mode string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	if err := os.WriteFile(path, []byte(mode), 0640); err != nil {
		return fmt.Errorf("write file: %w", err)
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
