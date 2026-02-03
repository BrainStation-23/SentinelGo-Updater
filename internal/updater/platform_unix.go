//go:build linux
// +build linux

package updater

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
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

	// Strategy 4: Parse /etc/passwd for current UID (Linux fallback)
	if home, err := getHomeFromPasswd(); err == nil && home != "" {
		LogInfo("Home directory detected from /etc/passwd: %s", home)
		return home, nil
	}

	// All strategies failed
	return "", fmt.Errorf("unable to determine home directory: all detection strategies failed")
}

// getHomeFromPasswd reads /etc/passwd to find the home directory for the current UID
func getHomeFromPasswd() (string, error) {
	// Get current UID
	uid := os.Getuid()

	// Open /etc/passwd
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return "", fmt.Errorf("failed to open /etc/passwd: %w", err)
	}
	defer file.Close()

	// Parse /etc/passwd line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		// Parse passwd entry: username:password:uid:gid:gecos:home:shell
		fields := strings.Split(line, ":")
		if len(fields) < 6 {
			continue
		}

		// Check if UID matches
		entryUID, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}

		if entryUID == uid {
			homeDir := fields[5]
			if homeDir != "" {
				return homeDir, nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading /etc/passwd: %w", err)
	}

	return "", fmt.Errorf("home directory not found in /etc/passwd for UID %d", uid)
}

// getPossibleBinaryPaths returns platform-specific possible paths for the sentinel binary
func getPossibleBinaryPaths() []string {
	var possiblePaths []string

	// Method 1: Check GOPATH environment variable
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		possiblePaths = append(possiblePaths, filepath.Join(gopath, "bin", "sentinel"))
	}

	// Method 2: Check current HOME
	if home := os.Getenv("HOME"); home != "" {
		possiblePaths = append(possiblePaths, filepath.Join(home, "go", "bin", "sentinel"))
	}

	// Method 3: Try os.UserHomeDir()
	if homeDir, err := os.UserHomeDir(); err == nil {
		possiblePaths = append(possiblePaths, filepath.Join(homeDir, "go", "bin", "sentinel"))
	}

	// Method 4: Try user.Current() to get home directory
	if currentUser, err := user.Current(); err == nil && currentUser.HomeDir != "" {
		possiblePaths = append(possiblePaths, filepath.Join(currentUser.HomeDir, "go", "bin", "sentinel"))
	}

	return possiblePaths
}
