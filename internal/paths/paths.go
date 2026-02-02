package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	detectorInstance     *BinaryDetector
	detectorInstanceOnce sync.Once
)

// GetDataDirectory returns the platform-specific data directory
// Linux/macOS: /var/lib/sentinelgo
// Windows: %ProgramData%\SentinelGo
func GetDataDirectory() string {
	switch runtime.GOOS {
	case "windows":
		programData := os.Getenv("ProgramData")
		if programData == "" {
			programData = "C:\\ProgramData"
		}
		return filepath.Join(programData, "SentinelGo")
	case "darwin", "linux":
		return "/var/lib/sentinelgo"
	default:
		return "/var/lib/sentinelgo"
	}
}

// GetDatabasePath returns the full path to the database file
func GetDatabasePath() string {
	return filepath.Join(GetDataDirectory(), "sentinel.db")
}

// GetUpdaterLogPath returns the full path to the updater log file
func GetUpdaterLogPath() string {
	return filepath.Join(GetDataDirectory(), "updater.log")
}

// GetAgentLogPath returns the full path to the agent log file
func GetAgentLogPath() string {
	return filepath.Join(GetDataDirectory(), "agent.log")
}

// GetBinaryDirectory returns the platform-specific binary installation directory
// Linux/macOS: /usr/local/bin
// Windows: %ProgramFiles%\SentinelGo
func GetBinaryDirectory() string {
	switch runtime.GOOS {
	case "windows":
		programFiles := os.Getenv("ProgramFiles")
		if programFiles == "" {
			programFiles = "C:\\Program Files"
		}
		return filepath.Join(programFiles, "SentinelGo")
	case "darwin", "linux":
		return "/usr/local/bin"
	default:
		return "/usr/local/bin"
	}
}

// GetMainAgentBinaryPath returns the full path to the main agent binary
// using dynamic detection with fallback to hardcoded paths
func GetMainAgentBinaryPath() string {
	// Initialize detector singleton
	detectorInstanceOnce.Do(func() {
		detectorInstance = GetDetector()
	})

	// Try dynamic detection
	path, err := detectorInstance.DetectBinaryPath()
	if err != nil {
		fmt.Printf("[WARN] Dynamic binary detection failed: %v\n", err)
		fmt.Println("[WARN] Falling back to hardcoded path")
		return getFallbackBinaryPath()
	}

	fmt.Printf("[INFO] Using dynamically detected binary path: %s\n", path)
	return path
}

// GetMainAgentBinaryPathWithRetry attempts to detect the binary path with retry logic
// This should be used by the updater to ensure transient failures are handled
func GetMainAgentBinaryPathWithRetry() (string, error) {
	// Initialize detector singleton
	detectorInstanceOnce.Do(func() {
		detectorInstance = GetDetector()
	})

	// Try dynamic detection
	path, err := detectorInstance.DetectBinaryPath()
	if err != nil {
		// Return error to caller so they can decide whether to retry
		return "", fmt.Errorf("binary path detection failed: %w", err)
	}

	return path, nil
}

// InvalidateBinaryPathCache forces re-detection of the binary path on next call
// This should be called when an update operation fails due to invalid path
func InvalidateBinaryPathCache() {
	if detectorInstance != nil {
		fmt.Println("[INFO] Invalidating binary path cache")
		detectorInstance.invalidateCache()
	}
}

// getFallbackBinaryPath returns the hardcoded fallback path for backward compatibility
func getFallbackBinaryPath() string {
	binaryName := "sentinel"
	if runtime.GOOS == "windows" {
		binaryName = "sentinel.exe"
	}
	return filepath.Join(GetBinaryDirectory(), binaryName)
}

// EnsureDataDirectory creates the data directory if it doesn't exist
// with 0755 permissions (rwxr-xr-x)
func EnsureDataDirectory() error {
	dataDir := GetDataDirectory()
	return os.MkdirAll(dataDir, 0755)
}
