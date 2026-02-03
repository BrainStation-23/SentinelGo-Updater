package updater

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/BrainStation-23/SentinelGo-Updater/internal/paths"
	"github.com/BrainStation-23/SentinelGo-Updater/internal/service"
)

const (
	// CheckInterval is the time between version checks
	CheckInterval = 30 * time.Second

	// MainAgentModule is the Go module path for the main agent
	MainAgentModule = "github.com/BrainStation-23/SentinelGo"

	// MainAgentServiceName is the service name for the main agent
	MainAgentServiceName = "sentinelgo"
)

var (
	serviceManager service.Manager
)

func init() {
	serviceManager = service.NewManager()
}

// Run is the main updater loop that checks for updates every CheckInterval
func Run() {
	// Initialize logging system
	if err := InitLogger(); err != nil {
		log.Fatalf("Failed to initialize logging system: %v", err)
	}
	defer CloseLogger()

	LogInfo("Updater service started")
	LogInfo("Check interval: %v", CheckInterval)
	LogInfo("Main agent module: %s", MainAgentModule)

	for {
		LogInfo("--- Starting version check ---")

		currentVersion, err := getInstalledVersion()
		if err != nil {
			LogError("Failed to get installed version: %v", err)
			LogInfo("Will retry in %v", CheckInterval)
			time.Sleep(CheckInterval)
			continue
		}

		LogInfo("Current installed version: %s", currentVersion)

		latestVersion, err := getLatestVersion()
		if err != nil {
			LogError("Failed to check latest version: %v", err)
			LogInfo("Will retry in %v", CheckInterval)
			time.Sleep(CheckInterval)
			continue
		}

		LogInfo("Latest available version: %s", latestVersion)

		if isNewerVersion(currentVersion, latestVersion) {
			LogInfo("Update available: %s -> %s", currentVersion, latestVersion)
			LogInfo("Initiating update process...")

			if err := performUpdate(latestVersion); err != nil {
				LogError("Update failed: %v", err)
				LogWarning("Main agent may need manual intervention")
			} else {
				LogInfo("Update successful: %s", latestVersion)
			}
		} else {
			LogInfo("No update needed, already running latest version")
		}

		LogInfo("Next check in %v", CheckInterval)
		time.Sleep(CheckInterval)
	}
}

// getInstalledVersion reads the current main agent version
func getInstalledVersion() (string, error) {
	binaryPath := paths.GetMainAgentBinaryPath()
	LogInfo("Checking for binary at system location: %s", binaryPath)

	// Check if binary exists at system location
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		LogInfo("Binary not found at system location, checking GOPATH...")
		// On macOS/Linux, also check user's GOPATH/bin as fallback
		if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
			gopath := os.Getenv("GOPATH")
			LogInfo("GOPATH from environment: %s", gopath)
			if gopath == "" {
				// Try multiple methods to get home directory
				homeDir := os.Getenv("HOME")
				if homeDir == "" {
					// If running as sudo, try to get the original user's home
					if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
						LogInfo("Running as sudo, SUDO_USER: %s", sudoUser)
						homeDir = filepath.Join("/Users", sudoUser)
						LogInfo("Using home directory: %s", homeDir)
					} else {
						// Fallback to os.UserHomeDir()
						var err error
						homeDir, err = os.UserHomeDir()
						if err != nil {
							LogError("Failed to get home directory: %v", err)
						} else {
							LogInfo("Got home directory from os.UserHomeDir: %s", homeDir)
						}
					}
				} else {
					LogInfo("Got home directory from HOME env: %s", homeDir)
				}

				if homeDir != "" {
					gopath = filepath.Join(homeDir, "go")
					LogInfo("GOPATH not set, using default: %s", gopath)
				}
			}

			if gopath != "" {
				binaryName := "sentinel"
				if runtime.GOOS == "windows" {
					binaryName = "sentinel.exe"
				}
				gopathBinary := filepath.Join(gopath, "bin", binaryName)
				LogInfo("Checking GOPATH binary location: %s", gopathBinary)

				if _, err := os.Stat(gopathBinary); err == nil {
					LogInfo("Found binary in GOPATH: %s", gopathBinary)
					binaryPath = gopathBinary
				} else {
					LogError("Binary not found at GOPATH location either: %v", err)
					return "", fmt.Errorf("main agent binary not found at %s or %s", binaryPath, gopathBinary)
				}
			} else {
				return "", fmt.Errorf("main agent binary not found at %s", binaryPath)
			}
		} else {
			return "", fmt.Errorf("main agent binary not found at %s", binaryPath)
		}
	} else {
		LogInfo("Found binary at system location: %s", binaryPath)
	}

	// Execute the binary with --version flag
	cmd := exec.Command(binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get version from binary: %w", err)
	}

	version := strings.TrimSpace(string(output))
	if version == "" {
		return "", fmt.Errorf("binary returned empty version")
	}

	return version, nil
}

// getLatestVersion queries the Go module system for the latest version
func getLatestVersion() (string, error) {
	// Use 'go list -m -json' to get the latest version
	cmd := exec.Command("go", "list", "-m", "-json", fmt.Sprintf("%s@latest", MainAgentModule))

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to query latest version: %w", err)
	}

	// Parse JSON output
	var moduleInfo struct {
		Version string `json:"Version"`
	}

	if err := json.Unmarshal(output, &moduleInfo); err != nil {
		return "", fmt.Errorf("failed to parse module info: %w", err)
	}

	if moduleInfo.Version == "" {
		return "", fmt.Errorf("no version found in module info")
	}

	return moduleInfo.Version, nil
}

// isNewerVersion compares two semantic versions and returns true if latest is newer
func isNewerVersion(current, latest string) bool {
	// Remove 'v' prefix if present
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	// If versions are identical, no update needed
	if current == latest {
		return false
	}

	// Parse versions
	currentParts := parseVersion(current)
	latestParts := parseVersion(latest)

	// Compare major, minor, patch
	for i := 0; i < 3; i++ {
		if latestParts[i] > currentParts[i] {
			return true
		}
		if latestParts[i] < currentParts[i] {
			return false
		}
	}

	return false
}

// parseVersion parses a semantic version string into [major, minor, patch]
func parseVersion(version string) [3]int {
	var parts [3]int

	// Split by '.' and parse each part
	segments := strings.Split(version, ".")
	for i := 0; i < len(segments) && i < 3; i++ {
		// Parse the number, ignoring any non-numeric suffixes
		var num int
		fmt.Sscanf(segments[i], "%d", &num)
		parts[i] = num
	}

	return parts
}

// performUpdate executes the complete update cycle with rollback support
func performUpdate(targetVersion string) error {
	LogInfo("=== Starting update to %s ===", targetVersion)

	// Get current version before any changes
	currentVersion, err := getInstalledVersion()
	if err != nil {
		LogWarning("Could not get current version: %v", err)
		currentVersion = "unknown"
	}

	// Create backup before any changes
	LogInfo("Creating backup before update...")
	backup, err := createBackup(currentVersion)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	// Ensure backup is cleaned up on success
	defer func() {
		if backup != nil && backup.BackupPath != "" {
			// Only remove backup if update was successful (no panic/error)
			if r := recover(); r == nil {
				LogInfo("Cleaning up backup file: %s", backup.BackupPath)
				if err := os.Remove(backup.BackupPath); err != nil && !os.IsNotExist(err) {
					LogWarning("Failed to remove backup file: %v", err)
				}
			}
		}
	}()

	// Wrap update steps in error handling with rollback
	updateErr := func() error {
		// Step 1: Stop the main agent service
		LogInfo("Step 1: Stopping main agent service...")
		if err := serviceManager.Stop(MainAgentServiceName); err != nil {
			return fmt.Errorf("failed to stop main agent: %w", err)
		}
		LogInfo("Main agent service stopped successfully")

		// Step 2: Uninstall the main agent service
		LogInfo("Step 2: Uninstalling main agent service...")
		if err := serviceManager.Uninstall(MainAgentServiceName); err != nil {
			return fmt.Errorf("failed to uninstall main agent: %w", err)
		}
		LogInfo("Main agent service uninstalled successfully")

		// Step 3: Clean up old files (except database)
		LogInfo("Step 3: Cleaning up old files...")
		if err := cleanupOldFiles(); err != nil {
			LogWarning("Cleanup failed: %v", err)
			// Continue anyway, this is not critical
		}
		LogInfo("Cleanup completed")

		// Step 4: Download and compile new version
		LogInfo("Step 4: Downloading and compiling version %s...", targetVersion)
		newBinaryPath, err := downloadAndCompile(targetVersion)
		if err != nil {
			return fmt.Errorf("failed to compile: %w", err)
		}
		LogInfo("Compilation successful, binary at: %s", newBinaryPath)

		// Step 5: Install new binary
		LogInfo("Step 5: Installing new binary...")
		if err := installBinary(newBinaryPath); err != nil {
			return fmt.Errorf("failed to install binary: %w", err)
		}
		LogInfo("Binary installed successfully")

		// Step 6: Reinstall service
		LogInfo("Step 6: Reinstalling main agent service...")
		installedBinaryPath := paths.GetMainAgentBinaryPath()
		if err := serviceManager.Install(MainAgentServiceName, installedBinaryPath); err != nil {
			return fmt.Errorf("failed to install service: %w", err)
		}
		LogInfo("Service reinstalled successfully")

		// Step 7: Start service
		LogInfo("Step 7: Starting main agent service...")
		if err := serviceManager.Start(MainAgentServiceName); err != nil {
			return fmt.Errorf("failed to start service: %w", err)
		}
		LogInfo("Service started successfully")

		// Step 8: Verify service is running
		LogInfo("Step 8: Verifying main agent is running...")
		if err := verifyMainAgentRunning(); err != nil {
			LogError("Service verification failed: %v", err)
			return fmt.Errorf("service not running after update: %w", err)
		}
		LogInfo("Main agent verified running")

		return nil
	}()

	// If update failed, trigger rollback
	if updateErr != nil {
		LogError("Update failed: %v", updateErr)
		LogInfo("Triggering rollback to previous version...")

		if rollbackErr := rollback(backup); rollbackErr != nil {
			LogCritical("Rollback failed: %v", rollbackErr)
			return fmt.Errorf("update failed and rollback failed: update error: %w, rollback error: %v", updateErr, rollbackErr)
		}

		LogInfo("Rollback successful, restored version %s", backup.Version)
		return fmt.Errorf("update failed, rolled back to version %s: %w", backup.Version, updateErr)
	}

	LogInfo("=== Update completed successfully ===")
	return nil
}

// cleanupOldFiles removes old binary and backup files while preserving database and logs
func cleanupOldFiles() error {
	var errors []string

	// Delete main agent binary
	binaryPath := paths.GetMainAgentBinaryPath()
	LogInfo("Deleting main agent binary: %s", binaryPath)
	if err := os.Remove(binaryPath); err != nil && !os.IsNotExist(err) {
		errors = append(errors, fmt.Sprintf("failed to delete binary %s: %v", binaryPath, err))
	} else if err == nil {
		LogInfo("Deleted: %s", binaryPath)
	}

	// Delete backup binary (.old)
	backupOldPath := binaryPath + ".old"
	LogInfo("Checking for backup file: %s", backupOldPath)
	if err := os.Remove(backupOldPath); err != nil && !os.IsNotExist(err) {
		errors = append(errors, fmt.Sprintf("failed to delete backup %s: %v", backupOldPath, err))
	} else if err == nil {
		LogInfo("Deleted: %s", backupOldPath)
	}

	// Delete backup binary (.backup)
	backupPath := binaryPath + ".backup"
	LogInfo("Checking for backup file: %s", backupPath)
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		errors = append(errors, fmt.Sprintf("failed to delete backup %s: %v", backupPath, err))
	} else if err == nil {
		LogInfo("Deleted: %s", backupPath)
	}

	// Verify database is preserved
	dbPath := paths.GetDatabasePath()
	if _, err := os.Stat(dbPath); err == nil {
		LogInfo("Database preserved at: %s", dbPath)
	} else if os.IsNotExist(err) {
		LogInfo("Database does not exist yet at: %s", dbPath)
	}

	// Verify log files are preserved
	updaterLogPath := paths.GetUpdaterLogPath()
	if _, err := os.Stat(updaterLogPath); err == nil {
		LogInfo("Updater log preserved at: %s", updaterLogPath)
	}

	agentLogPath := paths.GetAgentLogPath()
	if _, err := os.Stat(agentLogPath); err == nil {
		LogInfo("Agent log preserved at: %s", agentLogPath)
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup encountered errors: %s", strings.Join(errors, "; "))
	}

	LogInfo("Cleanup completed successfully")
	return nil
}

// downloadAndCompile downloads and compiles the specified version of the main agent
func downloadAndCompile(version string) (string, error) {
	LogInfo("Setting up Go environment for compilation...")

	// Setup Go environment variables
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		gopath = filepath.Join(homeDir, "go")
		LogInfo("GOPATH not set, using default: %s", gopath)
	}

	goroot := os.Getenv("GOROOT")
	if goroot == "" {
		// Try to detect GOROOT
		cmd := exec.Command("go", "env", "GOROOT")
		output, err := cmd.Output()
		if err == nil {
			goroot = strings.TrimSpace(string(output))
			LogInfo("Detected GOROOT: %s", goroot)
		}
	}

	gocache := os.Getenv("GOCACHE")
	if gocache == "" {
		gocache = filepath.Join(gopath, "cache")
		LogInfo("GOCACHE not set, using: %s", gocache)
	}

	gomodcache := os.Getenv("GOMODCACHE")
	if gomodcache == "" {
		gomodcache = filepath.Join(gopath, "pkg", "mod")
		LogInfo("GOMODCACHE not set, using: %s", gomodcache)
	}

	// Prepare environment variables
	env := os.Environ()
	env = append(env, "CGO_ENABLED=1") // Enable CGO for SQLite support
	env = append(env, fmt.Sprintf("GOPATH=%s", gopath))
	if goroot != "" {
		env = append(env, fmt.Sprintf("GOROOT=%s", goroot))
	}
	env = append(env, fmt.Sprintf("GOCACHE=%s", gocache))
	env = append(env, fmt.Sprintf("GOMODCACHE=%s", gomodcache))

	LogInfo("Environment variables configured:")
	LogInfo("  CGO_ENABLED=1")
	LogInfo("  GOPATH=%s", gopath)
	if goroot != "" {
		LogInfo("  GOROOT=%s", goroot)
	}
	LogInfo("  GOCACHE=%s", gocache)
	LogInfo("  GOMODCACHE=%s", gomodcache)

	// On Windows, locate and add GCC to PATH
	if runtime.GOOS == "windows" {
		LogInfo("Windows detected, checking for GCC...")
		gccPath, err := findGCCOnWindows()
		if err != nil {
			LogWarning("GCC not found: %v", err)
			LogWarning("CGO compilation may fail without GCC")
		} else {
			LogInfo("Found GCC at: %s", gccPath)
			// Add GCC directory to PATH
			pathEnv := os.Getenv("PATH")
			newPath := fmt.Sprintf("%s%c%s", gccPath, os.PathListSeparator, pathEnv)
			env = setEnvVar(env, "PATH", newPath)
			LogInfo("Added GCC to PATH")
		}
	}

	// Execute go install command
	moduleWithVersion := fmt.Sprintf("%s/cmd/sentinel@%s", MainAgentModule, version)
	LogInfo("Executing: go install %s", moduleWithVersion)

	cmd := exec.Command("go", "install", moduleWithVersion)
	cmd.Env = env

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()

	// Log compilation output
	if len(output) > 0 {
		LogInfo("Compilation output:\n%s", string(output))
	}

	if err != nil {
		LogError("Compilation failed: %v", err)
		LogError("Output: %s", string(output))
		return "", fmt.Errorf("compilation failed: %w\nOutput: %s", err, string(output))
	}

	// Determine the path to the compiled binary
	binaryName := "sentinel"
	if runtime.GOOS == "windows" {
		binaryName = "sentinel.exe"
	}
	compiledBinaryPath := filepath.Join(gopath, "bin", binaryName)

	// Verify the binary exists
	if _, err := os.Stat(compiledBinaryPath); os.IsNotExist(err) {
		LogError("Compiled binary not found at expected location: %s", compiledBinaryPath)
		return "", fmt.Errorf("compiled binary not found at expected location: %s", compiledBinaryPath)
	}

	LogInfo("Compilation successful, binary located at: %s", compiledBinaryPath)
	return compiledBinaryPath, nil
}

// findGCCOnWindows attempts to locate GCC on Windows systems
func findGCCOnWindows() (string, error) {
	// Common GCC installation paths on Windows
	commonPaths := []string{
		"C:\\MinGW\\bin",
		"C:\\MinGW64\\bin",
		"C:\\TDM-GCC-64\\bin",
		"C:\\msys64\\mingw64\\bin",
		"C:\\msys64\\ucrt64\\bin",
		"C:\\Program Files\\mingw-w64\\bin",
		"C:\\Program Files (x86)\\mingw-w64\\bin",
	}

	// Check if gcc is already in PATH
	if _, err := exec.LookPath("gcc"); err == nil {
		// GCC found in PATH, get its directory
		cmd := exec.Command("where", "gcc")
		output, err := cmd.Output()
		if err == nil {
			gccPath := strings.TrimSpace(strings.Split(string(output), "\n")[0])
			return filepath.Dir(gccPath), nil
		}
	}

	// Check common installation paths
	for _, path := range commonPaths {
		gccExe := filepath.Join(path, "gcc.exe")
		if _, err := os.Stat(gccExe); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("GCC not found in common locations or PATH")
}

// setEnvVar sets or updates an environment variable in the env slice
func setEnvVar(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

// installBinary copies the compiled binary to the installation directory
func installBinary(sourcePath string) error {
	targetPath := paths.GetMainAgentBinaryPath()

	LogInfo("Installing binary from %s to %s", sourcePath, targetPath)

	// Ensure the target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Read source file
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source binary: %w", err)
	}

	// Write to target location
	if err := os.WriteFile(targetPath, sourceData, 0755); err != nil {
		return fmt.Errorf("failed to write target binary: %w", err)
	}

	LogInfo("Binary written to: %s", targetPath)

	// On Unix systems, set executable permissions and ownership
	if runtime.GOOS != "windows" {
		// Set executable permissions (0755)
		if err := os.Chmod(targetPath, 0755); err != nil {
			return fmt.Errorf("failed to set executable permissions: %w", err)
		}
		LogInfo("Set executable permissions (0755) on: %s", targetPath)

		// Set ownership to root if running as root
		if os.Geteuid() == 0 {
			if err := os.Chown(targetPath, 0, 0); err != nil {
				LogWarning("Failed to set ownership to root: %v", err)
				// Don't fail the installation for this
			} else {
				LogInfo("Set ownership to root:root on: %s", targetPath)
			}
		}
	}

	// Verify binary exists and is executable
	fileInfo, err := os.Stat(targetPath)
	if err != nil {
		return fmt.Errorf("failed to verify installed binary: %w", err)
	}

	if runtime.GOOS != "windows" {
		// Check if file has executable bit set
		if fileInfo.Mode()&0111 == 0 {
			return fmt.Errorf("binary is not executable")
		}
	}

	LogInfo("Binary installation verified successfully")
	return nil
}

// verifyMainAgentRunning checks if the main agent service is running
func verifyMainAgentRunning() error {
	const maxRetries = 3
	const retryDelay = 2 * time.Second

	LogInfo("Verifying service is running (max %d retries, %v delay)...", maxRetries, retryDelay)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		LogInfo("Verification attempt %d/%d", attempt, maxRetries)

		isRunning, err := serviceManager.IsRunning(MainAgentServiceName)
		if err != nil {
			LogError("Error checking service status: %v", err)
			if attempt < maxRetries {
				LogInfo("Retrying in %v...", retryDelay)
				time.Sleep(retryDelay)
				continue
			}
			return fmt.Errorf("failed to check service status after %d attempts: %w", maxRetries, err)
		}

		if isRunning {
			LogInfo("Service is running (verified on attempt %d)", attempt)
			return nil
		}

		LogWarning("Service is not running yet")
		if attempt < maxRetries {
			LogInfo("Retrying in %v...", retryDelay)
			time.Sleep(retryDelay)
		}
	}

	return fmt.Errorf("service not running after %d verification attempts", maxRetries)
}

// BackupInfo stores information about a backup
type BackupInfo struct {
	Version    string
	BackupPath string
	Timestamp  time.Time
}

// createBackup creates a backup of the current binary before update
func createBackup(currentVersion string) (*BackupInfo, error) {
	LogInfo("Creating backup of current binary...")

	binaryPath := paths.GetMainAgentBinaryPath()

	// Check if current binary exists at system location
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// On macOS/Linux, also check user's GOPATH/bin as fallback
		if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
			gopath := os.Getenv("GOPATH")
			if gopath == "" {
				homeDir, err := os.UserHomeDir()
				if err == nil {
					gopath = filepath.Join(homeDir, "go")
				}
			}

			if gopath != "" {
				binaryName := "sentinel"
				gopathBinary := filepath.Join(gopath, "bin", binaryName)

				if _, err := os.Stat(gopathBinary); err == nil {
					LogInfo("Found binary in GOPATH for backup: %s", gopathBinary)
					binaryPath = gopathBinary
				} else {
					return nil, fmt.Errorf("current binary not found at %s or %s", binaryPath, gopathBinary)
				}
			} else {
				return nil, fmt.Errorf("current binary not found at %s", binaryPath)
			}
		} else {
			return nil, fmt.Errorf("current binary not found at %s", binaryPath)
		}
	}

	backupPath := binaryPath + ".backup"

	// Read current binary
	LogInfo("Reading current binary from: %s", binaryPath)
	binaryData, err := os.ReadFile(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read current binary: %w", err)
	}

	// Write backup file
	LogInfo("Writing backup to: %s", backupPath)
	if err := os.WriteFile(backupPath, binaryData, 0755); err != nil {
		return nil, fmt.Errorf("failed to write backup file: %w", err)
	}

	// Verify backup was created
	backupInfo, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to verify backup file: %w", err)
	}

	backup := &BackupInfo{
		Version:    currentVersion,
		BackupPath: backupPath,
		Timestamp:  time.Now(),
	}

	LogInfo("Backup created successfully:")
	LogInfo("  Version: %s", backup.Version)
	LogInfo("  Path: %s", backup.BackupPath)
	LogInfo("  Size: %d bytes", backupInfo.Size())
	LogInfo("  Timestamp: %s", backup.Timestamp.Format(time.RFC3339))

	return backup, nil
}

// rollback restores the previous version from backup
func rollback(backup *BackupInfo) error {
	LogInfo("=== Starting rollback process ===")
	LogInfo("Rolling back to version: %s", backup.Version)
	LogInfo("Backup path: %s", backup.BackupPath)

	// Step 1: Verify backup file exists
	LogInfo("Step 1: Verifying backup file exists...")
	if _, err := os.Stat(backup.BackupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found at %s", backup.BackupPath)
	}
	LogInfo("Backup file verified")

	// Step 2: Restore binary from backup
	LogInfo("Step 2: Restoring binary from backup...")
	binaryPath := paths.GetMainAgentBinaryPath()

	// Read backup file
	backupData, err := os.ReadFile(backup.BackupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Write to binary location
	if err := os.WriteFile(binaryPath, backupData, 0755); err != nil {
		return fmt.Errorf("failed to restore binary: %w", err)
	}
	LogInfo("Binary restored to: %s", binaryPath)

	// On Unix systems, set proper permissions
	if runtime.GOOS != "windows" {
		if err := os.Chmod(binaryPath, 0755); err != nil {
			LogWarning("Failed to set executable permissions: %v", err)
		}
		if os.Geteuid() == 0 {
			if err := os.Chown(binaryPath, 0, 0); err != nil {
				LogWarning("Failed to set ownership to root: %v", err)
			}
		}
	}

	// Step 3: Reinstall service using service manager
	LogInfo("Step 3: Reinstalling service...")
	if err := serviceManager.Install(MainAgentServiceName, binaryPath); err != nil {
		return fmt.Errorf("failed to reinstall service: %w", err)
	}
	LogInfo("Service reinstalled successfully")

	// Step 4: Start service using service manager
	LogInfo("Step 4: Starting service...")
	if err := serviceManager.Start(MainAgentServiceName); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	LogInfo("Service started successfully")

	// Step 5: Verify service is running
	LogInfo("Step 5: Verifying service is running...")
	if err := verifyMainAgentRunning(); err != nil {
		return fmt.Errorf("service not running after rollback: %w", err)
	}
	LogInfo("Service verified running")

	LogInfo("=== Rollback completed successfully to version %s ===", backup.Version)
	return nil
}
