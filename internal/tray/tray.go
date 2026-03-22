package tray

import (
	_ "embed"

	"fyne.io/systray"
)

//go:embed icons/navigation.png
var iconNavigation []byte

//go:embed icons/launcher.png
var iconLauncher []byte

//go:embed icons/disabled.png
var iconDisabled []byte

type Tray struct {
	modeItem    *systray.MenuItem
	enableItem  *systray.MenuItem
	quitCh      chan struct{}
	onQuit      func()
	onToggle    func(enabled bool)
	currentMode string
	enabled     bool
	ready       bool
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

	t.ready = true

	go func() {
		for {
			select {
			case <-t.enableItem.ClickedCh:
				t.enabled = !t.enabled
				if t.enabled {
					t.enableItem.SetTitle("Disable")
					t.updateIcon()
					systray.SetTooltip("SNES Controller - " + t.currentMode + " Mode")
				} else {
					t.enableItem.SetTitle("Enable")
					systray.SetIcon(iconDisabled)
					systray.SetTooltip("SNES Controller - Disabled")
				}
				if t.onToggle != nil {
					t.onToggle(t.enabled)
				}
			case <-mQuit.ClickedCh:
				if t.onQuit != nil {
					t.onQuit()
				}
				return
			case <-t.quitCh:
				return
			}
		}
	}()
}

func (t *Tray) updateIcon() {
	switch t.currentMode {
	case "navigation":
		systray.SetIcon(iconNavigation)
	case "launcher":
		systray.SetIcon(iconLauncher)
	default:
		systray.SetIcon(iconNavigation)
	}
}

func (t *Tray) SetMode(mode string) {
	if !t.ready {
		return
	}

	t.currentMode = mode

	if t.modeItem != nil {
		t.modeItem.SetTitle("Mode: " + mode)
	}

	if !t.enabled {
		return
	}

	t.updateIcon()
	systray.SetTooltip("SNES Controller - " + mode + " Mode")
}

func (t *Tray) IsEnabled() bool {
	return t.enabled
}

func (t *Tray) Quit() {
	select {
	case <-t.quitCh:
		// Already closed
	default:
		close(t.quitCh)
	}
}
