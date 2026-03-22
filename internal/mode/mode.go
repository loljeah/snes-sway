package mode

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Manager struct {
	mu         sync.RWMutex
	current    string
	modeFile   string
	notify     bool
	notifier   func(title, body string) error
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

func (m *Manager) OnModeChange(fn func(mode string)) {
	m.mu.Lock()
	m.onModeChange = fn
	m.mu.Unlock()
	// Notify immediately with current mode
	if fn != nil {
		fn(m.current)
	}
}

func (m *Manager) Current() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

func (m *Manager) Switch(name string) {
	m.mu.Lock()
	if m.current == name {
		m.mu.Unlock()
		return
	}
	m.current = name
	onChange := m.onModeChange
	m.mu.Unlock()

	m.writeModeFile()

	if m.notify && m.notifier != nil {
		icon := modeIcon(name)
		m.notifier(fmt.Sprintf("%s %s", icon, name), "")
	}

	if onChange != nil {
		onChange(name)
	}
}

func (m *Manager) writeModeFile() {
	if m.modeFile == "" {
		return
	}

	dir := filepath.Dir(m.modeFile)
	os.MkdirAll(dir, 0755)

	m.mu.RLock()
	data := []byte(m.current)
	m.mu.RUnlock()

	os.WriteFile(m.modeFile, data, 0644)
}

func modeIcon(name string) string {
	switch name {
	case "navigation":
		return "🎮"
	case "launcher":
		return "🚀"
	default:
		return "⚡"
	}
}
