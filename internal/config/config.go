package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/ljsm/snes-sway/internal/util"
	"gopkg.in/yaml.v3"
)

// Valid action types
var validActionTypes = map[string]bool{
	"sway":  true,
	"exec":  true,
	"key":   true,
	"mode":  true,
	"mouse": true,
}

// Valid button names
var validButtons = map[string]bool{
	"a": true, "b": true, "x": true, "y": true,
	"l": true, "r": true,
	"up": true, "down": true, "left": true, "right": true,
	"select+a": true, "select+b": true, "select+x": true, "select+y": true,
	"select+l": true, "select+r": true,
	"select+up": true, "select+down": true, "select+left": true, "select+right": true,
	"start+a": true, "start+b": true, "start+x": true, "start+y": true,
	"start+l": true, "start+r": true,
	"start+up": true, "start+down": true, "start+left": true, "start+right": true,
}

// Valid mouse actions
var validMouseActions = map[string]bool{
	"click_left": true, "click_right": true, "click_middle": true,
	"double_left": true,
	"hold_left": true, "hold_right": true,
	"release_left": true, "release_right": true,
	"move_up": true, "move_down": true, "move_left": true, "move_right": true,
}

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
	ModeTimeout int             `yaml:"mode_timeout"` // seconds, 0 = disabled
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
	expanded, err := util.ExpandPath(path)
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

	// Expand and validate mode file path (must be under home)
	if cfg.Indicator.ModeFile != "" {
		expanded, err := util.ValidatePathUnderHome(cfg.Indicator.ModeFile)
		if err != nil {
			return fmt.Errorf("mode file path: %w", err)
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
	// Default mode timeout: 30 seconds (use -1 to disable)
	if cfg.ModeTimeout == 0 {
		cfg.ModeTimeout = 30
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

// ValidationWarning represents a non-fatal config issue
type ValidationWarning struct {
	Mode    string
	Button  string
	Action  string
	Message string
}

func (w ValidationWarning) String() string {
	if w.Mode != "" && w.Button != "" {
		return fmt.Sprintf("[%s.%s] %s", w.Mode, w.Button, w.Message)
	}
	return w.Message
}

// Validate checks the config for issues and returns warnings
func (c *Config) Validate() []ValidationWarning {
	var warnings []ValidationWarning

	// Check default mode exists
	if _, ok := c.Modes[c.DefaultMode]; !ok && len(c.Modes) > 0 {
		warnings = append(warnings, ValidationWarning{
			Message: fmt.Sprintf("default_mode '%s' not defined in modes", c.DefaultMode),
		})
	}

	// Check each mode
	for modeName, mode := range c.Modes {
		for button, action := range mode {
			// Validate button name
			if !validButtons[button] {
				warnings = append(warnings, ValidationWarning{
					Mode:    modeName,
					Button:  button,
					Message: fmt.Sprintf("unknown button '%s'", button),
				})
			}

			// Validate action format
			parts := strings.SplitN(action, ":", 2)
			if len(parts) != 2 {
				warnings = append(warnings, ValidationWarning{
					Mode:   modeName,
					Button: button,
					Action: action,
					Message: "invalid action format (expected type:command)",
				})
				continue
			}

			actionType := parts[0]
			actionCmd := parts[1]

			if !validActionTypes[actionType] {
				warnings = append(warnings, ValidationWarning{
					Mode:    modeName,
					Button:  button,
					Action:  action,
					Message: fmt.Sprintf("unknown action type '%s'", actionType),
				})
				continue
			}

			// Validate mode references
			if actionType == "mode" {
				if _, ok := c.Modes[actionCmd]; !ok {
					warnings = append(warnings, ValidationWarning{
						Mode:    modeName,
						Button:  button,
						Action:  action,
						Message: fmt.Sprintf("mode '%s' not defined", actionCmd),
					})
				}
			}

			// Validate mouse actions
			if actionType == "mouse" {
				// Extract action name (before optional :speed)
				mouseParts := strings.SplitN(actionCmd, ":", 2)
				mouseAction := mouseParts[0]
				if !validMouseActions[mouseAction] {
					warnings = append(warnings, ValidationWarning{
						Mode:    modeName,
						Button:  button,
						Action:  action,
						Message: fmt.Sprintf("unknown mouse action '%s'", mouseAction),
					})
				}
			}
		}
	}

	return warnings
}

// PrintValidationWarnings prints warnings to stderr
func PrintValidationWarnings(warnings []ValidationWarning) {
	if len(warnings) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, "config validation warnings:")
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "  - %s\n", w)
	}
}
