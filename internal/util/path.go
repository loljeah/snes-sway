package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath expands ~ to user's home directory
func ExpandPath(path string) (string, error) {
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

// ValidatePathUnderHome checks that the logical path is under user's home directory.
// Uses filepath.Clean (not EvalSymlinks) to resolve traversal like ../ without
// following symlinks. This is intentional: on NixOS, Home Manager creates symlinks
// from ~/.config/ to /nix/store/, which is expected and safe.
func ValidatePathUnderHome(path string) (string, error) {
	expanded, err := ExpandPath(path)
	if err != nil {
		return "", err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}

	// Resolve to absolute path and clean traversal sequences
	absPath, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	cleanPath := filepath.Clean(absPath)

	// Check if cleaned path is under home (append separator to prevent
	// /home/lj matching /home/ljsm)
	homePrefix := home + string(filepath.Separator)
	if cleanPath != home && !strings.HasPrefix(cleanPath, homePrefix) {
		return "", fmt.Errorf("path %s is outside home directory", path)
	}

	return expanded, nil
}
