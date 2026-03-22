package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"fyne.io/systray"
	"github.com/ljsm/snes-sway/internal/config"
	"github.com/ljsm/snes-sway/internal/input"
	"github.com/ljsm/snes-sway/internal/mode"
	"github.com/ljsm/snes-sway/internal/sway"
	"github.com/ljsm/snes-sway/internal/tray"
)

var (
	debug      bool
	noTray     bool
	configPath string
)

func main() {
	flag.StringVar(&configPath, "config", config.DefaultConfigPath(), "config file path")
	flag.BoolVar(&debug, "debug", false, "print button events")
	flag.BoolVar(&noTray, "no-tray", false, "disable system tray icon")
	flag.Parse()

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

var (
	trayInstance *tray.Tray
	quitChan     = make(chan struct{})
	enabled      = true
	enabledChan  = make(chan bool, 1)
)

func onReady() {
	trayInstance = tray.NewWithSystray(
		func() {
			close(quitChan)
			systray.Quit()
		},
		func(e bool) {
			enabled = e
			select {
			case enabledChan <- e:
			default:
			}
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
	// Cleanup
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

	if t != nil {
		modeMgr.OnModeChange(func(mode string) {
			t.SetMode(mode)
		})
		// Set initial mode
		t.SetMode(cfg.DefaultMode)
	}

	// Watch config for hot reload
	cfgMgr.Watch(func(newCfg *config.Config) {
		fmt.Fprintln(os.Stderr, "config reloaded")
	})

	go reader.Run()

	fmt.Fprintf(os.Stderr, "snes-sway running (mode: %s)\n", cfg.DefaultMode)

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

		case <-enabledChan:
			// Enabled state changed, just continue loop
			continue

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
			if !enabled {
				continue
			}

			if !ev.Pressed {
				continue
			}

			currentMode := modeMgr.Current()
			cfg := cfgMgr.Get()

			modeConfig, ok := cfg.Modes[currentMode]
			if !ok {
				fmt.Fprintf(os.Stderr, "unknown mode: %s\n", currentMode)
				continue
			}

			action, ok := modeConfig[string(ev.Button)]
			if !ok {
				continue
			}

			// Handle mode switch
			if strings.HasPrefix(action, "mode:") {
				newMode := strings.TrimPrefix(action, "mode:")
				modeMgr.Switch(newMode)
				continue
			}

			if err := executor.Run(action); err != nil {
				fmt.Fprintf(os.Stderr, "action error: %v\n", err)
			}
		}
	}
}
