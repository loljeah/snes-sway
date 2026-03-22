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
	ready       chan struct{}
}

func New(onQuit func()) *Tray {
	return &Tray{
		quitCh: make(chan struct{}),
		onQuit: onQuit,
		ready:  make(chan struct{}),
	}
}

func (t *Tray) Run() {
	systray.Run(t.onReady, t.onExit)
}

func (t *Tray) onReady() {
	systray.SetIcon(iconNavigation)
	systray.SetTitle("SNES-Sway")
	systray.SetTooltip("SNES Controller - Navigation Mode")

	t.modeItem = systray.AddMenuItem("Mode: navigation", "Current controller mode")
	t.modeItem.Disable()

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "Stop snes-sway daemon")

	close(t.ready)

	go func() {
		for {
			select {
			case <-mQuit.ClickedCh:
				if t.onQuit != nil {
					t.onQuit()
				}
				systray.Quit()
				return
			case <-t.quitCh:
				systray.Quit()
				return
			}
		}
	}()
}

func (t *Tray) onExit() {
	// Cleanup if needed
}

func (t *Tray) SetMode(mode string) {
	// Wait for tray to be ready
	<-t.ready

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
