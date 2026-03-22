package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"fyne.io/systray"
	"github.com/ljsm/snes-sway/internal/config"
	"github.com/ljsm/snes-sway/internal/input"
	"github.com/ljsm/snes-sway/internal/mode"
	"github.com/ljsm/snes-sway/internal/repeat"
	"github.com/ljsm/snes-sway/internal/sway"
	"github.com/ljsm/snes-sway/internal/tray"
)

var (
	debug        bool
	noTray       bool
	configPath   string
	generateConf bool
	validateOnly bool
)

func main() {
	flag.StringVar(&configPath, "config", config.DefaultConfigPath(), "config file path")
	flag.BoolVar(&debug, "debug", false, "print button events")
	flag.BoolVar(&noTray, "no-tray", false, "disable system tray icon")
	flag.BoolVar(&generateConf, "generate-config", false, "interactively generate config file")
	flag.BoolVar(&validateOnly, "validate", false, "validate config and exit")
	flag.Parse()

	if generateConf {
		if err := runConfigGenerator(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if validateOnly {
		if err := runValidation(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if noTray {
		// Run without systray
		if err := runDaemon(nil); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// systray.Run must be called from main thread
		systray.Run(onReady, onExit)
	}
}

func runValidation() error {
	cfgMgr, err := config.NewManager(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	defer cfgMgr.Close()

	cfg := cfgMgr.Get()
	warnings := cfg.Validate()

	if len(warnings) == 0 {
		fmt.Println("config is valid")
		return nil
	}

	config.PrintValidationWarnings(warnings)
	return fmt.Errorf("%d validation warnings", len(warnings))
}

var (
	mu           sync.RWMutex
	trayInstance *tray.Tray
	quitChan     = make(chan struct{})
	quitOnce     sync.Once
	enabled      = true
)

func onReady() {
	trayInstance = tray.NewWithSystray(
		func() {
			quitOnce.Do(func() {
				close(quitChan)
			})
			systray.Quit()
		},
		func(e bool) {
			mu.Lock()
			enabled = e
			mu.Unlock()
			if e {
				fmt.Fprintln(os.Stderr, "controller enabled")
			} else {
				fmt.Fprintln(os.Stderr, "controller disabled")
			}
		},
	)

	go func() {
		if err := runDaemon(trayInstance); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			systray.Quit()
		}
	}()
}

func onExit() {
	// Cleanup - ensure quitChan is closed
	quitOnce.Do(func() {
		close(quitChan)
	})
}

func isEnabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return enabled
}

func runDaemon(t *tray.Tray) error {
	if err := config.EnsureConfigDir(); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	cfgMgr, err := config.NewManager(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	defer cfgMgr.Close()

	cfg := cfgMgr.Get()

	// Validate config on startup
	if warnings := cfg.Validate(); len(warnings) > 0 {
		config.PrintValidationWarnings(warnings)
	}

	// Validate sway setup
	if err := sway.ValidateSetup(); err != nil {
		return fmt.Errorf("validate setup: %w", err)
	}

	executor := sway.NewExecutor()

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	modeMgr := mode.NewManager(
		cfg.DefaultMode,
		cfg.Indicator.ModeFile,
		cfg.Indicator.Notify,
		executor.Notify,
	)

	// Set mode timeout
	modeMgr.SetTimeout(cfg.ModeTimeout)

	if t != nil {
		modeMgr.OnModeChange(func(m string) {
			t.SetMode(m)
		})
		// Set initial mode
		t.SetMode(cfg.DefaultMode)
	}

	// Watch config for hot reload
	if err := cfgMgr.Watch(func(newCfg *config.Config) {
		fmt.Fprintln(os.Stderr, "config reloaded")
		modeMgr.SetTimeout(newCfg.ModeTimeout)
		if warnings := newCfg.Validate(); len(warnings) > 0 {
			config.PrintValidationWarnings(warnings)
		}
	}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: config watch failed: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "snes-sway running (mode: %s, timeout: %ds)\n", cfg.DefaultMode, cfg.ModeTimeout)

	// Auto-reconnect loop
	for {
		err := runInputLoop(t, cfgMgr, modeMgr, executor, cfg, sigChan)
		if err == nil {
			// Clean shutdown
			return nil
		}

		// Check if it's a device disconnect
		if strings.Contains(err.Error(), "disconnected") || strings.Contains(err.Error(), "find device") {
			fmt.Fprintf(os.Stderr, "device disconnected, waiting for reconnect...\n")
			if t != nil {
				t.SetMode("disconnected")
			}

			// Wait for device to reappear
			for {
				select {
				case <-sigChan:
					fmt.Fprintln(os.Stderr, "shutting down")
					return nil
				case <-quitChan:
					fmt.Fprintln(os.Stderr, "quit from tray")
					return nil
				case <-time.After(2 * time.Second):
					// Try to find device again
					cfg = cfgMgr.Get()
					devicePath := cfg.Device.Path
					if devicePath == "" {
						path, findErr := input.FindDevice(cfg.Device.VendorID, cfg.Device.ProductID)
						if findErr != nil {
							continue // Keep waiting
						}
						devicePath = path
					}

					// Device found, break out of reconnect loop
					fmt.Fprintf(os.Stderr, "device reconnected: %s\n", devicePath)
					if t != nil {
						t.SetMode(modeMgr.Current())
					}
					goto reconnected
				}
			}
		reconnected:
			continue
		}

		// Other error, return it
		return err
	}
}

func runInputLoop(t *tray.Tray, cfgMgr *config.Manager, modeMgr *mode.Manager, executor *sway.Executor, cfg *config.Config, sigChan chan os.Signal) error {
	// Find or use configured device
	devicePath := cfg.Device.Path
	if devicePath == "" {
		path, err := input.FindDevice(cfg.Device.VendorID, cfg.Device.ProductID)
		if err != nil {
			return fmt.Errorf("find device: %w", err)
		}
		devicePath = path
	}

	reader, err := input.NewReader(devicePath)
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer reader.Close()

	// Create repeater for held button actions (mouse movement)
	repeater := repeat.New(
		repeat.DefaultConfig(),
		func(action string) error {
			return executor.Run(action)
		},
		repeat.ShouldRepeat,
	)
	defer repeater.Stop()

	go reader.Run()

	for {
		select {
		case <-sigChan:
			fmt.Fprintln(os.Stderr, "shutting down")
			if t != nil {
				systray.Quit()
			}
			return nil

		case <-quitChan:
			fmt.Fprintln(os.Stderr, "quit from tray")
			return nil

		case <-reader.Disconnected():
			return fmt.Errorf("input device disconnected")

		case ev, ok := <-reader.Events():
			if !ok {
				return fmt.Errorf("input device disconnected")
			}

			if debug {
				state := "released"
				if ev.Pressed {
					state = "pressed"
				}
				fmt.Fprintf(os.Stderr, "[debug] %s %s\n", ev.Button, state)
			}

			// Skip if disabled
			if !isEnabled() {
				if !ev.Pressed {
					repeater.Release(string(ev.Button))
				}
				continue
			}

			// Reset mode timeout on any button press
			if ev.Pressed {
				modeMgr.ResetTimer()
			}

			currentMode := modeMgr.Current()
			currentCfg := cfgMgr.Get()

			modeConfig, ok := currentCfg.Modes[currentMode]
			if !ok {
				fmt.Fprintf(os.Stderr, "unknown mode: %s\n", currentMode)
				continue
			}

			action, ok := modeConfig[string(ev.Button)]

			// Handle button release - stop repeating
			if !ev.Pressed {
				repeater.Release(string(ev.Button))
				continue
			}

			if !ok {
				continue
			}

			// Handle mode switch
			if strings.HasPrefix(action, "mode:") {
				newMode := strings.TrimPrefix(action, "mode:")
				modeMgr.Switch(newMode)
				continue
			}

			// Check if this is a repeatable action (mouse movement)
			if repeat.ShouldRepeat(action) {
				repeater.Press(string(ev.Button), action)
				continue
			}

			// Normal one-shot action
			if err := executor.Run(action); err != nil {
				fmt.Fprintf(os.Stderr, "action error: %v\n", err)
			}
		}
	}
}
