package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

type Device struct {
	Path      string `yaml:"path"`
	VendorID  uint16 `yaml:"vendor_id"`
	ProductID uint16 `yaml:"product_id"`
}

type Indicator struct {
	ModeFile string `yaml:"mode_file"`
	Notify   bool   `yaml:"notify"`
}

type Config struct {
	Device      Device          `yaml:"device"`
	Indicator   Indicator       `yaml:"indicator"`
	Modes       map[string]Mode `yaml:"modes"`
	DefaultMode string          `yaml:"default_mode"`
}

type Mode map[string]string // button -> action

type Manager struct {
	mu       sync.RWMutex
	config   *Config
	path     string
	onChange func(*Config)
	watcher  *fsnotify.Watcher
	closed   bool
}

func NewManager(path string) (*Manager, error) {
	expanded, err := expandPath(path)
	if err != nil {
		return nil, fmt.Errorf("expand path: %w", err)
	}

	m := &Manager{path: expanded}
	if err := m.load(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	// Expand mode file path
	if cfg.Indicator.ModeFile != "" {
		expanded, err := expandPath(cfg.Indicator.ModeFile)
		if err != nil {
			return fmt.Errorf("expand mode file path: %w", err)
		}
		cfg.Indicator.ModeFile = expanded
	}

	// Validate config
	if cfg.DefaultMode == "" {
		cfg.DefaultMode = "navigation"
	}
	if cfg.Modes == nil {
		cfg.Modes = make(map[string]Mode)
	}

	m.mu.Lock()
	m.config = &cfg
	m.mu.Unlock()

	return nil
}

func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		// Return empty config to prevent nil dereference
		return &Config{
			DefaultMode: "navigation",
			Modes:       make(map[string]Mode),
		}
	}
	return m.config
}

func (m *Manager) Watch(onChange func(*Config)) error {
	m.mu.Lock()
	m.onChange = onChange
	m.mu.Unlock()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}

	m.mu.Lock()
	m.watcher = watcher
	m.mu.Unlock()

	go m.watchLoop()

	if err := watcher.Add(m.path); err != nil {
		watcher.Close()
		return fmt.Errorf("watch file: %w", err)
	}

	return nil
}

func (m *Manager) watchLoop() {
	for {
		m.mu.RLock()
		watcher := m.watcher
		closed := m.closed
		m.mu.RUnlock()

		if closed || watcher == nil {
			return
		}

		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				if err := m.load(); err != nil {
					fmt.Fprintf(os.Stderr, "reload config: %v\n", err)
					continue
				}
				m.mu.RLock()
				onChange := m.onChange
				cfg := m.config
				m.mu.RUnlock()

				if onChange != nil && cfg != nil {
					onChange(cfg)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "watcher error: %v\n", err)
		}
	}
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	if m.watcher != nil {
		return m.watcher.Close()
	}
	return nil
}

func expandPath(path string) (string, error) {
	if len(path) == 0 {
		return path, nil
	}
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home dir: %w", err)
		}
		return filepath.Join(home, path[1:]), nil
	}
	return path, nil
}

func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to relative path
		return ".config/snes-sway/config.yaml"
	}
	return filepath.Join(home, ".config", "snes-sway", "config.yaml")
}

func EnsureConfigDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	dir := filepath.Join(home, ".config", "snes-sway")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	return nil
}
