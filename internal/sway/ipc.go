package sway

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Binary paths - use full paths for systemd compatibility
var (
	swaymsgPath    = findBinary("swaymsg", "/run/current-system/sw/bin/swaymsg", "/usr/bin/swaymsg", nixProfileBin("swaymsg"))
	notifySendPath = findBinary("notify-send", "/run/current-system/sw/bin/notify-send", "/usr/bin/notify-send", nixProfileBin("notify-send"))
	wtypePath      = findBinary("wtype", "/run/current-system/sw/bin/wtype", "/usr/bin/wtype", nixProfileBin("wtype"))
	wlrctlPath     = findBinary("wlrctl", "/run/current-system/sw/bin/wlrctl", "/usr/bin/wlrctl", nixProfileBin("wlrctl"))
	dotoolPath     = findBinary("dotool", "/run/current-system/sw/bin/dotool", "/usr/bin/dotool", nixProfileBin("dotool"))
)

func nixProfileBin(name string) string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".nix-profile", "bin", name)
	}
	return ""
}

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
	// Fallback to name (will fail at runtime with clear error)
	return name
}

// ValidateSetup checks if required binaries are available
func ValidateSetup() error {
	required := map[string]string{
		"swaymsg": swaymsgPath,
		"wlrctl":  wlrctlPath,
	}
	optional := map[string]string{
		"wtype":       wtypePath,
		"dotool":      dotoolPath,
		"notify-send": notifySendPath,
	}

	var missing []string
	for name, path := range required {
		if _, err := exec.LookPath(path); err != nil {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("required binaries not found: %v", missing)
	}

	// Warn about optional binaries
	for name, path := range optional {
		if _, err := exec.LookPath(path); err != nil {
			fmt.Fprintf(os.Stderr, "warning: optional binary %s not found (some features may not work)\n", name)
		}
	}

	return nil
}

type Executor struct {
	timeout time.Duration
}

func NewExecutor() *Executor {
	return &Executor{
		timeout: 5 * time.Second,
	}
}

var validActionTypes = map[string]bool{
	"sway":  true,
	"exec":  true,
	"key":   true,
	"mode":  true,
	"mouse": true,
}

func (e *Executor) Run(action string) error {
	if action == "" {
		return nil
	}

	parts := strings.SplitN(action, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid action format: %s (expected type:command)", action)
	}

	actionType := parts[0]
	command := parts[1]

	if !validActionTypes[actionType] {
		return fmt.Errorf("unknown action type: %s (valid: sway, exec, key, mode, mouse)", actionType)
	}

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
	case "mouse":
		return e.mouseAction(command)
	default:
		return fmt.Errorf("unknown action type: %s", actionType)
	}
}

func (e *Executor) swaymsg(cmd string) error {
	return e.runCommand(swaymsgPath, cmd)
}

func (e *Executor) exec(cmd string) error {
	// Use swaymsg exec to properly launch in sway context
	return e.runCommand(swaymsgPath, "exec", "--", cmd)
}

func (e *Executor) sendKey(key string) error {
	// wtype -k sends a key press+release
	return e.runCommand(wtypePath, "-k", key)
}

// mouseAction handles mouse: commands
// Supported: click_left, click_right, click_middle, move_up, move_down, move_left, move_right
//
//	hold_left, hold_right, release_left, release_right (for drag/select)
//	double_left (double click)
//
// move commands accept optional speed: move_up:50
func (e *Executor) mouseAction(cmd string) error {
	parts := strings.SplitN(cmd, ":", 2)
	action := parts[0]
	speed := 20
	if len(parts) == 2 {
		if s, err := strconv.Atoi(parts[1]); err == nil && s > 0 {
			speed = s
		}
	}

	switch action {
	case "click_left":
		return e.runCommand(wlrctlPath, "pointer", "click", "left")
	case "click_right":
		return e.runCommand(wlrctlPath, "pointer", "click", "right")
	case "click_middle":
		return e.runCommand(wlrctlPath, "pointer", "click", "middle")
	case "double_left":
		// Double click for word selection
		if err := e.runCommand(wlrctlPath, "pointer", "click", "left"); err != nil {
			return err
		}
		time.Sleep(50 * time.Millisecond)
		return e.runCommand(wlrctlPath, "pointer", "click", "left")
	case "hold_left":
		// Hold left button down (for drag/select) - uses dotool
		return e.dotool("buttondown left")
	case "hold_right":
		return e.dotool("buttondown right")
	case "release_left":
		// Release left button
		return e.dotool("buttonup left")
	case "release_right":
		return e.dotool("buttonup right")
	case "move_up":
		return e.runCommand(wlrctlPath, "pointer", "move", "0", strconv.Itoa(-speed))
	case "move_down":
		return e.runCommand(wlrctlPath, "pointer", "move", "0", strconv.Itoa(speed))
	case "move_left":
		return e.runCommand(wlrctlPath, "pointer", "move", strconv.Itoa(-speed), "0")
	case "move_right":
		return e.runCommand(wlrctlPath, "pointer", "move", strconv.Itoa(speed), "0")
	default:
		return fmt.Errorf("unknown mouse action: %s", action)
	}
}

func (e *Executor) runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run with timeout to prevent hanging
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			stderrStr := strings.TrimSpace(stderr.String())
			if stderrStr != "" {
				return fmt.Errorf("%s %v: %w (%s)", name, args, err, stderrStr)
			}
			return fmt.Errorf("%s %v: %w", name, args, err)
		}
		return nil
	case <-time.After(e.timeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait() // Reap zombie process
		}
		return fmt.Errorf("%s %v: timeout after %v", name, args, e.timeout)
	}
}

func (e *Executor) Notify(title, body string) error {
	args := []string{"-t", "1500", title}
	if body != "" {
		args = append(args, body)
	}
	return e.runCommand(notifySendPath, args...)
}

// validDotoolCommands restricts dotool input to known-safe commands
var validDotoolCommands = map[string]bool{
	"buttondown left":  true,
	"buttondown right": true,
	"buttonup left":    true,
	"buttonup right":   true,
}

// dotool sends a command to dotool via stdin.
// Only known-safe commands are allowed to prevent command injection.
func (e *Executor) dotool(command string) error {
	if !validDotoolCommands[command] {
		return fmt.Errorf("dotool: rejected unknown command %q", command)
	}

	cmd := exec.Command(dotoolPath)
	cmd.Stdin = strings.NewReader(command + "\n")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			stderrStr := strings.TrimSpace(stderr.String())
			if stderrStr != "" {
				return fmt.Errorf("dotool %s: %w (%s)", command, err, stderrStr)
			}
			return fmt.Errorf("dotool %s: %w", command, err)
		}
		return nil
	case <-time.After(e.timeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait() // Reap zombie process
		}
		return fmt.Errorf("dotool %s: timeout", command)
	}
}
