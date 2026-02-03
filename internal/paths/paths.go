package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetDataDirectory returns the platform-specific data directory
// macOS: /Library/Application Support/SentinelGo
// Linux: /var/lib/sentinelgo
// Windows: %ProgramData%\SentinelGo
func GetDataDirectory() string {
	switch runtime.GOOS {
	case "windows":
		programData := os.Getenv("ProgramData")
		if programData == "" {
			programData = "C:\\ProgramData"
		}
		return filepath.Join(programData, "SentinelGo")
	case "darwin":
		return "/Library/Application Support/SentinelGo"
	case "linux":
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
// with platform-specific binary names (sentinel on Unix, sentinel.exe on Windows)
func GetMainAgentBinaryPath() string {
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
