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

// ValidatePathUnderHome checks that the resolved path is under user's home directory
// Returns the expanded path if valid, error if path escapes home
func ValidatePathUnderHome(path string) (string, error) {
	expanded, err := ExpandPath(path)
	if err != nil {
		return "", err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	// Resolve any symlinks
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil && !os.IsNotExist(err) {
		// If file doesn't exist yet, check parent directory
		parentPath := filepath.Dir(absPath)
		realParent, parentErr := filepath.EvalSymlinks(parentPath)
		if parentErr != nil {
			return "", fmt.Errorf("resolve parent path: %w", parentErr)
		}
		realPath = filepath.Join(realParent, filepath.Base(absPath))
	} else if err != nil {
		realPath = absPath
	}

	// Check if path is under home
	if !strings.HasPrefix(realPath, home) {
		return "", fmt.Errorf("path %s is outside home directory", path)
	}

	return expanded, nil
}
