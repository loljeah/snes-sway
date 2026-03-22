package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ljsm/snes-sway/internal/config"
)

func runConfigGenerator() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("snes-sway config generator")
	fmt.Println("==========================")
	fmt.Println()

	// Device settings
	fmt.Println("Device settings:")
	vendorID := prompt(reader, "Vendor ID (hex, default 0x2341 for Arduino):", "0x2341")
	productID := prompt(reader, "Product ID (hex, default 0x8036 for Leonardo):", "0x8036")

	// Parse hex values
	vid, _ := strconv.ParseUint(strings.TrimPrefix(vendorID, "0x"), 16, 16)
	pid, _ := strconv.ParseUint(strings.TrimPrefix(productID, "0x"), 16, 16)

	// Mode timeout
	fmt.Println()
	fmt.Println("Mode timeout:")
	timeoutStr := prompt(reader, "Timeout in seconds (0 to disable, default 30):", "30")
	timeout, _ := strconv.Atoi(timeoutStr)

	// Notifications
	fmt.Println()
	notifyStr := prompt(reader, "Show notifications on mode switch? (y/n, default y):", "y")
	notify := strings.ToLower(notifyStr) == "y" || strings.ToLower(notifyStr) == "yes"

	// Modes
	fmt.Println()
	fmt.Println("Configure modes:")
	fmt.Println("Available action types:")
	fmt.Println("  sway:<cmd>   - Run swaymsg command (e.g., sway:focus left)")
	fmt.Println("  exec:<cmd>   - Run shell command (e.g., exec:firefox)")
	fmt.Println("  key:<key>    - Send keypress (e.g., key:Return)")
	fmt.Println("  mouse:<act>  - Mouse action (e.g., mouse:click_left, mouse:move_up:30)")
	fmt.Println("  mode:<name>  - Switch to mode (e.g., mode:launcher)")
	fmt.Println()

	modes := make(map[string]config.Mode)

	// Navigation mode
	fmt.Println("--- Navigation Mode (default) ---")
	modes["navigation"] = promptMode(reader, "navigation", map[string]string{
		"up":    "sway:focus up",
		"down":  "sway:focus down",
		"left":  "sway:focus left",
		"right": "sway:focus right",
		"a":     "mode:input",
		"b":     "sway:scratchpad show",
		"l":     "sway:workspace prev",
		"r":     "sway:workspace next",
	})

	// Input mode
	fmt.Println()
	fmt.Println("--- Input Mode (arrow keys to focused window) ---")
	modes["input"] = promptMode(reader, "input", map[string]string{
		"up":    "key:Up",
		"down":  "key:Down",
		"left":  "key:Left",
		"right": "key:Right",
		"a":     "key:Return",
		"b":     "mode:navigation",
	})

	// Mouse mode
	fmt.Println()
	addMouseStr := prompt(reader, "Add mouse mode? (y/n, default y):", "y")
	if strings.ToLower(addMouseStr) == "y" || strings.ToLower(addMouseStr) == "yes" {
		fmt.Println("--- Mouse Mode (cursor control) ---")
		speedStr := prompt(reader, "Mouse speed in pixels (default 20):", "20")
		speed, _ := strconv.Atoi(speedStr)
		if speed <= 0 {
			speed = 20
		}
		modes["mouse"] = promptMode(reader, "mouse", map[string]string{
			"up":    fmt.Sprintf("mouse:move_up:%d", speed),
			"down":  fmt.Sprintf("mouse:move_down:%d", speed),
			"left":  fmt.Sprintf("mouse:move_left:%d", speed),
			"right": fmt.Sprintf("mouse:move_right:%d", speed),
			"a":     "mouse:click_left",
			"b":     "mode:navigation",
			"x":     "mouse:click_right",
		})
	}

	// Additional modes
	fmt.Println()
	addMoreStr := prompt(reader, "Add launcher mode? (y/n, default n):", "n")
	if strings.ToLower(addMoreStr) == "y" || strings.ToLower(addMoreStr) == "yes" {
		fmt.Println("--- Launcher Mode (app shortcuts) ---")
		modes["launcher"] = promptMode(reader, "launcher", map[string]string{
			"a":     "exec:foot",
			"b":     "exec:firefox",
			"x":     "exec:thunar",
			"y":     "exec:code",
			"l":     "sway:workspace prev",
			"r":     "sway:workspace next",
			"up":    "sway:focus up",
			"down":  "sway:focus down",
			"left":  "sway:focus left",
			"right": "sway:focus right",
		})
	}

	// Build config
	cfg := config.Config{
		Device: config.Device{
			VendorID:  uint16(vid),
			ProductID: uint16(pid),
		},
		Indicator: config.Indicator{
			ModeFile: "~/.config/snes-sway/mode",
			Notify:   notify,
		},
		Modes:       modes,
		DefaultMode: "navigation",
		ModeTimeout: timeout,
	}

	// Validate
	if warnings := cfg.Validate(); len(warnings) > 0 {
		fmt.Println()
		fmt.Println("Validation warnings:")
		for _, w := range warnings {
			fmt.Printf("  - %s\n", w)
		}
	}

	// Output path
	fmt.Println()
	outputPath := prompt(reader, "Output path:", config.DefaultConfigPath())
	expanded, err := expandPath(outputPath)
	if err != nil {
		return fmt.Errorf("expand path: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(expanded), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Generate YAML
	content := generateYAML(cfg)

	// Write file
	if err := os.WriteFile(expanded, []byte(content), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	fmt.Printf("\nConfig written to: %s\n", expanded)
	return nil
}

func prompt(reader *bufio.Reader, question, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s] ", question, defaultVal)
	} else {
		fmt.Printf("%s ", question)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultVal
	}
	return input
}

func promptMode(reader *bufio.Reader, modeName string, defaults map[string]string) config.Mode {
	mode := make(config.Mode)

	buttons := []string{"up", "down", "left", "right", "a", "b", "x", "y", "l", "r"}

	fmt.Printf("Configure buttons for %s mode (press Enter for default, '-' to skip):\n", modeName)
	for _, btn := range buttons {
		defaultVal := defaults[btn]
		action := prompt(reader, fmt.Sprintf("  %s:", btn), defaultVal)
		if action != "" && action != "-" {
			mode[btn] = action
		}
	}

	// Select chords
	fmt.Println("Select chords (optional):")
	selectChords := []string{"select+a", "select+b", "select+l", "select+r"}
	for _, btn := range selectChords {
		defaultVal := defaults[btn]
		action := prompt(reader, fmt.Sprintf("  %s:", btn), defaultVal)
		if action != "" && action != "-" {
			mode[btn] = action
		}
	}

	return mode
}

func generateYAML(cfg config.Config) string {
	var sb strings.Builder

	sb.WriteString("# snes-sway configuration\n")
	sb.WriteString("# Generated by snes-sway --generate-config\n\n")

	sb.WriteString("device:\n")
	sb.WriteString(fmt.Sprintf("  vendor_id: 0x%04x\n", cfg.Device.VendorID))
	sb.WriteString(fmt.Sprintf("  product_id: 0x%04x\n", cfg.Device.ProductID))
	sb.WriteString("\n")

	sb.WriteString("indicator:\n")
	sb.WriteString(fmt.Sprintf("  mode_file: %s\n", cfg.Indicator.ModeFile))
	sb.WriteString(fmt.Sprintf("  notify: %t\n", cfg.Indicator.Notify))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("default_mode: %s\n", cfg.DefaultMode))
	sb.WriteString(fmt.Sprintf("mode_timeout: %d\n", cfg.ModeTimeout))
	sb.WriteString("\n")

	sb.WriteString("modes:\n")
	for modeName, mode := range cfg.Modes {
		sb.WriteString(fmt.Sprintf("  %s:\n", modeName))
		for btn, action := range mode {
			sb.WriteString(fmt.Sprintf("    %s: \"%s\"\n", btn, action))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func expandPath(path string) (string, error) {
	if len(path) == 0 {
		return path, nil
	}
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home dir: %w", err)
		}
		return filepath.Join(home, path[1:]), nil
	}
	return path, nil
}
