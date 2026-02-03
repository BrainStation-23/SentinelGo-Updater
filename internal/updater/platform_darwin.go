//go:build darwin
// +build darwin

package updater

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

// ensureHomeDirectory determines the home directory using multiple fallback strategies
func ensureHomeDirectory() (string, error) {
	// Strategy 1: Check $HOME environment variable
	if home := os.Getenv("HOME"); home != "" {
		LogInfo("Home directory detected from $HOME environment variable: %s", home)
		return home, nil
	}

	// Strategy 2: Use os.UserHomeDir()
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		LogInfo("Home directory detected using os.UserHomeDir(): %s", home)
		return home, nil
	}

	// Strategy 3: Use user.Current() to get home directory
	if currentUser, err := user.Current(); err == nil && currentUser.HomeDir != "" {
		LogInfo("Home directory detected using user.Current(): %s", currentUser.HomeDir)
		return currentUser.HomeDir, nil
	}

	// All strategies failed
	return "", fmt.Errorf("unable to determine home directory: all detection strategies failed")
}

// getPossibleBinaryPaths returns platform-specific possible paths for the sentinel binary
func getPossibleBinaryPaths() []string {
	var possiblePaths []string

	// Method 1: Check GOPATH environment variable
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		possiblePaths = append(possiblePaths, filepath.Join(gopath, "bin", "sentinel"))
	}

	// Method 2: Check SUDO_USER's home directory
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		userHome := filepath.Join("/Users", sudoUser)
		possiblePaths = append(possiblePaths, filepath.Join(userHome, "go", "bin", "sentinel"))
	}

	// Method 3: Check current HOME
	if home := os.Getenv("HOME"); home != "" {
		possiblePaths = append(possiblePaths, filepath.Join(home, "go", "bin", "sentinel"))
	}

	// Method 4: Try os.UserHomeDir()
	if homeDir, err := os.UserHomeDir(); err == nil {
		possiblePaths = append(possiblePaths, filepath.Join(homeDir, "go", "bin", "sentinel"))
	}

	// Method 5: Scan /Users directory (macOS-specific)
	usersDir := "/Users"
	if entries, err := os.ReadDir(usersDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && entry.Name() != "Shared" && entry.Name() != "Guest" {
				possiblePaths = append(possiblePaths, filepath.Join(usersDir, entry.Name(), "go", "bin", "sentinel"))
			}
		}
	}

	return possiblePaths
}
