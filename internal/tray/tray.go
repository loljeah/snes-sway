package tray

import (
	_ "embed"

	"fyne.io/systray"
)

//go:embed icons/navigation.png
var iconNavigation []byte

//go:embed icons/launcher.png
var iconLauncher []byte

type Tray struct {
	modeItem    *systray.MenuItem
	quitCh      chan struct{}
	onQuit      func()
	currentMode string
	ready       bool
}

// NewWithSystray creates a Tray and immediately sets up the systray.
// Call this from onReady callback when systray.Run is called from main.
func NewWithSystray(onQuit func()) *Tray {
	t := &Tray{
		quitCh: make(chan struct{}),
		onQuit: onQuit,
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

	mQuit := systray.AddMenuItem("Quit", "Stop snes-sway daemon")

	t.ready = true

	go func() {
		for {
			select {
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

func (t *Tray) SetMode(mode string) {
	if !t.ready {
		return
	}

	t.currentMode = mode

	if t.modeItem != nil {
		t.modeItem.SetTitle("Mode: " + mode)
	}

	switch mode {
	case "navigation":
		systray.SetIcon(iconNavigation)
		systray.SetTooltip("SNES Controller - Navigation Mode")
	case "launcher":
		systray.SetIcon(iconLauncher)
		systray.SetTooltip("SNES Controller - Launcher Mode")
	default:
		systray.SetTooltip("SNES Controller - " + mode)
	}
}

func (t *Tray) Quit() {
	select {
	case <-t.quitCh:
		// Already closed
	default:
		close(t.quitCh)
	}
}
