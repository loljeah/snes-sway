package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ljsm/snes-sway/internal/config"
	"github.com/ljsm/snes-sway/internal/input"
	"github.com/ljsm/snes-sway/internal/mode"
	"github.com/ljsm/snes-sway/internal/sway"
	"github.com/ljsm/snes-sway/internal/tray"
)

var (
	debug   bool
	noTray  bool
)

func main() {
	configPath := flag.String("config", config.DefaultConfigPath(), "config file path")
	flag.BoolVar(&debug, "debug", false, "print button events")
	flag.BoolVar(&noTray, "no-tray", false, "disable system tray icon")
	flag.Parse()

	if err := run(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
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
	quitChan := make(chan struct{})

	modeMgr := mode.NewManager(
		cfg.DefaultMode,
		cfg.Indicator.ModeFile,
		cfg.Indicator.Notify,
		executor.Notify,
	)

	// System tray
	var systray *tray.Tray
	if !noTray {
		systray = tray.New(func() {
			close(quitChan)
		})
		modeMgr.OnModeChange(func(mode string) {
			systray.SetMode(mode)
		})
		go systray.Run()
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
			if systray != nil {
				systray.Quit()
			}
			return nil

		case <-quitChan:
			fmt.Fprintln(os.Stderr, "quit from tray")
			return nil

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
