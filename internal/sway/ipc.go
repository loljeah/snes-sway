package sway

import (
	"fmt"
	"os/exec"
	"strings"
)

// Binary paths - use full paths for systemd compatibility
var (
	swaymsgPath    = findBinary("swaymsg", "/run/current-system/sw/bin/swaymsg", "/usr/bin/swaymsg")
	notifySendPath = findBinary("notify-send", "/run/current-system/sw/bin/notify-send", "/usr/bin/notify-send")
	wtypePath      = findBinary("wtype", "/run/current-system/sw/bin/wtype", "/usr/bin/wtype", "/home/ljsm/.nix-profile/bin/wtype")
)

func findBinary(name string, candidates ...string) string {
	// Try PATH first
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	// Try candidates
	for _, path := range candidates {
		if _, err := exec.LookPath(path); err == nil {
			return path
		}
	}
	// Fallback to name (will fail at runtime)
	return name
}

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) Run(action string) error {
	if action == "" {
		return nil
	}

	parts := strings.SplitN(action, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid action format: %s", action)
	}

	actionType := parts[0]
	command := parts[1]

	switch actionType {
	case "sway":
		return e.swaymsg(command)
	case "exec":
		return e.exec(command)
	case "key":
		return e.sendKey(command)
	case "mode":
		// Handled by caller
		return nil
	default:
		return fmt.Errorf("unknown action type: %s", actionType)
	}
}

func (e *Executor) swaymsg(cmd string) error {
	return exec.Command(swaymsgPath, cmd).Run()
}

func (e *Executor) exec(cmd string) error {
	// Use swaymsg exec to properly launch in sway context
	return exec.Command(swaymsgPath, "exec", "--", cmd).Run()
}

func (e *Executor) sendKey(key string) error {
	// wtype -k sends a key press+release
	return exec.Command(wtypePath, "-k", key).Run()
}

func (e *Executor) Notify(title, body string) error {
	args := []string{"-t", "1500", title}
	if body != "" {
		args = append(args, body)
	}
	return exec.Command(notifySendPath, args...).Run()
}
