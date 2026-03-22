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
	Device      Device              `yaml:"device"`
	Indicator   Indicator           `yaml:"indicator"`
	Modes       map[string]Mode     `yaml:"modes"`
	DefaultMode string              `yaml:"default_mode"`
}

type Mode map[string]string // button -> action

type Manager struct {
	mu       sync.RWMutex
	config   *Config
	path     string
	onChange func(*Config)
	watcher  *fsnotify.Watcher
}

func NewManager(path string) (*Manager, error) {
	expanded := expandPath(path)

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

	cfg.Indicator.ModeFile = expandPath(cfg.Indicator.ModeFile)

	m.mu.Lock()
	m.config = &cfg
	m.mu.Unlock()

	return nil
}

func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

func (m *Manager) Watch(onChange func(*Config)) error {
	m.onChange = onChange

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	m.watcher = watcher

	go func() {
		for {
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
					if m.onChange != nil {
						m.onChange(m.Get())
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Fprintf(os.Stderr, "watcher error: %v\n", err)
			}
		}
	}()

	return watcher.Add(m.path)
}

func (m *Manager) Close() error {
	if m.watcher != nil {
		return m.watcher.Close()
	}
	return nil
}

func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}

func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "snes-sway", "config.yaml")
}

func EnsureConfigDir() error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "snes-sway")
	return os.MkdirAll(dir, 0755)
}
