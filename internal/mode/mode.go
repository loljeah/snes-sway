package mode

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Manager struct {
	mu           sync.RWMutex
	current      string
	modeFile     string
	notify       bool
	notifier     func(title, body string) error
	onModeChange func(mode string)
}

func NewManager(defaultMode, modeFile string, notify bool, notifier func(string, string) error) *Manager {
	m := &Manager{
		current:  defaultMode,
		modeFile: modeFile,
		notify:   notify,
		notifier: notifier,
	}
	m.writeModeFile()
	return m
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
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	if err := os.WriteFile(path, []byte(mode), 0644); err != nil {
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
