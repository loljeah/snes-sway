package tray

import (
	_ "embed"
	"sync"

	"fyne.io/systray"
)

//go:embed icons/navigation.png
var iconNavigation []byte

//go:embed icons/launcher.png
var iconLauncher []byte

//go:embed icons/disabled.png
var iconDisabled []byte

type Tray struct {
	mu          sync.RWMutex
	modeItem    *systray.MenuItem
	enableItem  *systray.MenuItem
	quitCh      chan struct{}
	onQuit      func()
	onToggle    func(enabled bool)
	currentMode string
	enabled     bool
	ready       bool
	closed      bool
}

// NewWithSystray creates a Tray and immediately sets up the systray.
// Call this from onReady callback when systray.Run is called from main.
func NewWithSystray(onQuit func(), onToggle func(enabled bool)) *Tray {
	t := &Tray{
		quitCh:   make(chan struct{}),
		onQuit:   onQuit,
		onToggle: onToggle,
		enabled:  true,
	}
	t.setup()
	return t
}

func (t *Tray) setup() {
	systray.SetIcon(iconNavigation)
	systray.SetTitle("SNES")
	systray.SetTooltip("SNES Controller - Navigation Mode")

	t.modeItem = systray.AddMenuItem("Mode: navigation", "Current controller mode")
	t.modeItem.Disable()

	systray.AddSeparator()

	t.enableItem = systray.AddMenuItem("Disable", "Disable controller input")

	mQuit := systray.AddMenuItem("Quit", "Stop snes-sway daemon")

	t.mu.Lock()
	t.ready = true
	t.mu.Unlock()

	go t.eventLoop(mQuit)
}

func (t *Tray) eventLoop(mQuit *systray.MenuItem) {
	for {
		select {
		case <-t.enableItem.ClickedCh:
			t.mu.Lock()
			t.enabled = !t.enabled
			enabled := t.enabled
			mode := t.currentMode
			onToggle := t.onToggle
			t.mu.Unlock()

			if enabled {
				t.enableItem.SetTitle("Disable")
				t.updateIconLocked(mode)
				systray.SetTooltip("SNES Controller - " + mode + " Mode")
			} else {
				t.enableItem.SetTitle("Enable")
				systray.SetIcon(iconDisabled)
				systray.SetTooltip("SNES Controller - Disabled")
			}

			if onToggle != nil {
				onToggle(enabled)
			}

		case <-mQuit.ClickedCh:
			t.mu.Lock()
			onQuit := t.onQuit
			t.mu.Unlock()

			if onQuit != nil {
				onQuit()
			}
			return

		case <-t.quitCh:
			return
		}
	}
}

func (t *Tray) updateIconLocked(mode string) {
	switch mode {
	case "navigation":
		systray.SetIcon(iconNavigation)
	case "launcher":
		systray.SetIcon(iconLauncher)
	case "input":
		systray.SetIcon(iconNavigation) // Use navigation icon for input mode
	default:
		systray.SetIcon(iconNavigation)
	}
}

func (t *Tray) SetMode(mode string) {
	t.mu.Lock()
	if !t.ready {
		t.mu.Unlock()
		return
	}

	t.currentMode = mode
	enabled := t.enabled
	t.mu.Unlock()

	if t.modeItem != nil {
		t.modeItem.SetTitle("Mode: " + mode)
	}

	if !enabled {
		return
	}

	t.updateIconLocked(mode)
	systray.SetTooltip("SNES Controller - " + mode + " Mode")
}

func (t *Tray) IsEnabled() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.enabled
}

func (t *Tray) Quit() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return
	}
	t.closed = true
	close(t.quitCh)
}
