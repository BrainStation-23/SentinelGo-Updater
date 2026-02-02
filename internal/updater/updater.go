package updater

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
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

	// Strategy 4: Parse /etc/passwd for current UID (Linux/Unix fallback)
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		if home, err := getHomeFromPasswd(); err == nil && home != "" {
			LogInfo("Home directory detected from /etc/passwd: %s", home)
			return home, nil
		}
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

// setEnvironmentVariables ensures required environment variables are set for child processes
func setEnvironmentVariables() error {
	LogInfo("Setting up environment variables for update process...")

	// Ensure $HOME is set
	homeDir, err := ensureHomeDirectory()
	if err != nil {
		LogError("Failed to determine home directory: %v", err)
		return fmt.Errorf("failed to determine home directory: %w", err)
	}

	// Set $HOME if not already set
	if os.Getenv("HOME") == "" {
		if err := os.Setenv("HOME", homeDir); err != nil {
			LogError("Failed to set $HOME environment variable: %v", err)
			return fmt.Errorf("failed to set $HOME: %w", err)
		}
		LogInfo("Set $HOME environment variable to: %s", homeDir)
	} else {
		LogInfo("$HOME environment variable already set to: %s", os.Getenv("HOME"))
	}

	// Set $GOPATH if not already set (default to $HOME/go)
	if os.Getenv("GOPATH") == "" {
		gopath := filepath.Join(homeDir, "go")
		if err := os.Setenv("GOPATH", gopath); err != nil {
			LogError("Failed to set $GOPATH environment variable: %v", err)
			return fmt.Errorf("failed to set $GOPATH: %w", err)
		}
		LogInfo("Set $GOPATH environment variable to: %s", gopath)
	} else {
		LogInfo("$GOPATH environment variable already set to: %s", os.Getenv("GOPATH"))
	}

	LogInfo("Environment variables configured successfully")
	return nil
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
			LogInfo("This is a transient error - detection will be retried automatically")
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
	// Use retry-enabled detection with detailed logging
	binaryPath, detectionMethod, err := getMainAgentBinaryPathWithDetails()
	if err != nil {
		// Log detailed error but allow retry on next check
		LogError("Failed to detect binary path: %v", err)
		LogWarning("Will retry detection on next update check")
		LogInfo("Detection will be retried in %v", CheckInterval)
		return "", fmt.Errorf("binary path detection failed: %w", err)
	}

	// Log successful detection with method used
	LogInfo("Binary path successfully detected using method: %s", detectionMethod)
	LogInfo("Using binary at: %s", binaryPath)

	// Check if binary exists (additional validation)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		LogError("Binary not found at detected path: %s", binaryPath)
		LogWarning("Invalidating cache and will retry on next check")
		paths.InvalidateBinaryPathCache()
		return "", fmt.Errorf("main agent binary not found at %s", binaryPath)
	}

	// Execute the binary with --version flag
	cmd := exec.Command(binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		// If version check fails, it might be a corrupted binary
		LogError("Failed to get version from binary at %s: %v", binaryPath, err)
		LogWarning("Binary may be corrupted or incompatible")
		LogWarning("Invalidating cache to force re-detection on next check")
		paths.InvalidateBinaryPathCache()
		return "", fmt.Errorf("failed to get version from binary: %w", err)
	}

	version := strings.TrimSpace(string(output))
	if version == "" {
		LogError("Binary at %s returned empty version", binaryPath)
		LogWarning("This may indicate an incompatible or corrupted binary")
		return "", fmt.Errorf("binary returned empty version")
	}

	// Extract just the version number from output like "SentinelGo v1.6.116"
	// The binary may return various formats, so we need to extract the version
	versionParts := strings.Fields(version)
	for _, part := range versionParts {
		// Look for a part that starts with 'v' followed by a digit
		if len(part) > 1 && part[0] == 'v' && part[1] >= '0' && part[1] <= '9' {
			return part, nil
		}
	}

	// If no version pattern found, return the full output
	LogWarning("Could not extract version number from output: %s", version)
	return version, nil
}

// getMainAgentBinaryPathWithDetails wraps the path detection and returns the detection method used
func getMainAgentBinaryPathWithDetails() (path string, method string, err error) {
	// Get the detector instance to access detection details
	detector := paths.GetDetector()

	// Attempt detection
	detectedPath, detectionErr := detector.DetectBinaryPath()
	if detectionErr != nil {
		return "", "", detectionErr
	}

	// Determine which method was used by checking the detector's last successful method
	// Since we don't have direct access to the method, we'll infer it from the detection process
	method = inferDetectionMethod(detectedPath)

	return detectedPath, method, nil
}

// inferDetectionMethod attempts to determine which detection method was used
func inferDetectionMethod(detectedPath string) string {
	// Check if path matches common patterns to infer the detection method

	// Check if it's from a manual config
	configPath := filepath.Join(paths.GetDataDirectory(), "updater-config.json")
	if _, err := os.Stat(configPath); err == nil {
		return "manual_configuration"
	}

	// Check if it's in PATH
	pathEnv := os.Getenv("PATH")
	if pathEnv != "" {
		separator := ":"
		if runtime.GOOS == "windows" {
			separator = ";"
		}
		pathDirs := strings.Split(pathEnv, separator)
		detectedDir := filepath.Dir(detectedPath)
		for _, dir := range pathDirs {
			if dir == detectedDir {
				return "path_environment_variable"
			}
		}
	}

	// Check if it's in common paths
	commonPaths := getCommonInstallationPaths()
	for _, commonPath := range commonPaths {
		if detectedPath == commonPath {
			return "common_installation_directory"
		}
	}

	// Check if it's likely from service config (platform-specific paths)
	switch runtime.GOOS {
	case "linux":
		if strings.Contains(detectedPath, "/systemd/") || strings.Contains(detectedPath, "/lib/") {
			return "systemd_service_configuration"
		}
	case "darwin":
		if strings.Contains(detectedPath, "/Library/") || strings.Contains(detectedPath, "/LaunchDaemons/") {
			return "launchd_service_configuration"
		}
	case "windows":
		if strings.Contains(detectedPath, "Program Files") || strings.Contains(detectedPath, "ProgramData") {
			return "windows_service_configuration"
		}
	}

	// Default to "auto_detection" if we can't determine the specific method
	return "auto_detection"
}

// getCommonInstallationPaths returns platform-specific common installation paths
func getCommonInstallationPaths() []string {
	binaryName := "sentinel"
	if runtime.GOOS == "windows" {
		binaryName = "sentinel.exe"
	}

	switch runtime.GOOS {
	case "linux":
		return []string{
			"/usr/local/bin/" + binaryName,
			"/usr/bin/" + binaryName,
			"/opt/sentinelgo/" + binaryName,
			filepath.Join(os.Getenv("HOME"), "go/bin", binaryName),
			filepath.Join(os.Getenv("HOME"), ".local/bin", binaryName),
		}
	case "darwin":
		return []string{
			"/usr/local/bin/" + binaryName,
			"/usr/bin/" + binaryName,
			"/opt/sentinelgo/" + binaryName,
			filepath.Join(os.Getenv("HOME"), "go/bin", binaryName),
			"/Applications/SentinelGo/" + binaryName,
		}
	case "windows":
		return []string{
			filepath.Join(os.Getenv("ProgramFiles"), "SentinelGo", binaryName),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "SentinelGo", binaryName),
			filepath.Join(os.Getenv("USERPROFILE"), "go", "bin", binaryName),
			"C:\\SentinelGo\\" + binaryName,
		}
	default:
		return []string{
			"/usr/local/bin/" + binaryName,
			"/usr/bin/" + binaryName,
		}
	}
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

	// Set up environment variables before any operations
	LogInfo("Setting up environment for update...")
	if err := setEnvironmentVariables(); err != nil {
		LogError("Environment setup failed: %v", err)
		return fmt.Errorf("failed to set up environment: %w", err)
	}
	LogInfo("Environment setup completed successfully")

	// Get current version before any changes
	currentVersion, err := getInstalledVersion()
	if err != nil {
		LogWarning("Could not get current version: %v", err)
		LogWarning("This may indicate the binary is not properly installed")
		currentVersion = "unknown"

		// If we can't even detect the current binary, we should not proceed with update
		if currentVersion == "unknown" {
			LogError("Cannot proceed with update - current binary not detected")
			LogError("Please ensure sentinel is properly installed before updating")
			return fmt.Errorf("cannot update: current binary not detected: %w", err)
		}
	}

	// Create backup before any changes
	LogInfo("Creating backup before update...")
	backup, err := createBackup(currentVersion)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

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

		// Invalidate binary path cache since we just installed a new binary
		LogInfo("Invalidating binary path cache after installation")
		paths.InvalidateBinaryPathCache()

		// Step 6: Reinstall service
		LogInfo("Step 6: Reinstalling main agent service...")

		// Re-detect the binary path after installation
		installedBinaryPath, detectionMethod, detectErr := getMainAgentBinaryPathWithDetails()
		if detectErr != nil {
			LogError("Failed to detect newly installed binary: %v", detectErr)
			// Fall back to non-retry method as a last resort
			installedBinaryPath = paths.GetMainAgentBinaryPath()
			LogWarning("Using fallback path detection: %s", installedBinaryPath)
		} else {
			LogInfo("Newly installed binary detected using method: %s", detectionMethod)
			LogInfo("Binary path: %s", installedBinaryPath)
		}

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

	// Update successful - clean up backup file
	LogInfo("Update completed successfully, cleaning up backup file...")
	if err := cleanupBackupFile(backup.BackupPath); err != nil {
		LogWarning("Failed to clean up backup file: %v", err)
		LogWarning("Backup file may need to be manually deleted: %s", backup.BackupPath)
		// Don't fail the update for cleanup errors
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

	// Delete legacy backup binary (.old)
	backupOldPath := binaryPath + ".old"
	LogInfo("Checking for legacy backup file: %s", backupOldPath)
	if err := os.Remove(backupOldPath); err != nil && !os.IsNotExist(err) {
		errors = append(errors, fmt.Sprintf("failed to delete legacy backup %s: %v", backupOldPath, err))
	} else if err == nil {
		LogInfo("Deleted legacy backup: %s", backupOldPath)
	} else if os.IsNotExist(err) {
		LogInfo("No legacy backup file found (this is normal)")
	}

	// Preserve current backup binary (.backup) for potential rollback
	backupPath := binaryPath + ".backup"
	LogInfo("Checking for current backup file: %s", backupPath)
	if _, err := os.Stat(backupPath); err == nil {
		LogInfo("Preserving backup file for potential rollback: %s", backupPath)
	} else if os.IsNotExist(err) {
		LogWarning("Backup file not found at: %s", backupPath)
		LogWarning("Rollback will not be possible if update fails")
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
	backupPath := binaryPath + ".backup"

	// Check if current binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("current binary not found at %s", binaryPath)
	}

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
		LogCritical("Backup file not found at %s", backup.BackupPath)
		LogCritical("RECOVERY INSTRUCTIONS:")
		LogCritical("  1. The system is in an unrecoverable state without the backup file")
		LogCritical("  2. You may need to manually reinstall the sentinel binary")
		LogCritical("  3. Check if a backup exists at an alternate location")
		LogCritical("  4. Contact support if you need assistance with manual recovery")
		return fmt.Errorf("backup file not found at %s - manual recovery required", backup.BackupPath)
	}
	LogInfo("Backup file verified")

	// Step 2: Restore binary from backup
	LogInfo("Step 2: Restoring binary from backup...")
	binaryPath := paths.GetMainAgentBinaryPath()

	// Read backup file
	backupData, err := os.ReadFile(backup.BackupPath)
	if err != nil {
		LogCritical("Failed to read backup file: %v", err)
		LogCritical("RECOVERY INSTRUCTIONS:")
		LogCritical("  1. Verify the backup file has correct permissions: %s", backup.BackupPath)
		LogCritical("  2. Check disk space and file system integrity")
		LogCritical("  3. Attempt manual restoration of the backup file")
		return fmt.Errorf("failed to read backup file: %w - manual recovery may be required", err)
	}

	// Write to binary location
	if err := os.WriteFile(binaryPath, backupData, 0755); err != nil {
		LogCritical("Failed to restore binary: %v", err)
		LogCritical("RECOVERY INSTRUCTIONS:")
		LogCritical("  1. Verify write permissions to: %s", binaryPath)
		LogCritical("  2. Check available disk space")
		LogCritical("  3. Manually copy backup file from %s to %s", backup.BackupPath, binaryPath)
		return fmt.Errorf("failed to restore binary: %w - manual recovery required", err)
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
		LogError("Failed to reinstall service: %v", err)
		LogError("RECOVERY INSTRUCTIONS:")
		LogError("  1. The binary has been restored to: %s", binaryPath)
		LogError("  2. Manually reinstall the service using your system's service manager")
		LogError("  3. Backup file preserved at: %s", backup.BackupPath)
		return fmt.Errorf("failed to reinstall service: %w - manual service installation required", err)
	}
	LogInfo("Service reinstalled successfully")

	// Step 4: Start service using service manager
	LogInfo("Step 4: Starting service...")
	if err := serviceManager.Start(MainAgentServiceName); err != nil {
		LogError("Failed to start service: %v", err)
		LogError("RECOVERY INSTRUCTIONS:")
		LogError("  1. The service has been reinstalled but failed to start")
		LogError("  2. Check service logs for startup errors")
		LogError("  3. Manually start the service: systemctl start %s (Linux) or equivalent", MainAgentServiceName)
		LogError("  4. Backup file preserved at: %s", backup.BackupPath)
		return fmt.Errorf("failed to start service: %w - manual service start required", err)
	}
	LogInfo("Service started successfully")

	// Step 5: Verify service is running
	LogInfo("Step 5: Verifying service is running...")
	if err := verifyMainAgentRunning(); err != nil {
		LogError("Service not running after rollback: %v", err)
		LogError("RECOVERY INSTRUCTIONS:")
		LogError("  1. The service was started but verification failed")
		LogError("  2. Check service status manually: systemctl status %s (Linux) or equivalent", MainAgentServiceName)
		LogError("  3. Review service logs for errors")
		LogError("  4. Backup file preserved at: %s", backup.BackupPath)
		return fmt.Errorf("service not running after rollback: %w - manual verification required", err)
	}
	LogInfo("Service verified running")

	// Preserve backup file for manual inspection after rollback
	LogInfo("=== Rollback completed successfully to version %s ===", backup.Version)
	LogInfo("Backup file preserved for manual inspection at: %s", backup.BackupPath)
	LogInfo("You may manually delete the backup file after verifying system health:")
	LogInfo("  rm %s", backup.BackupPath)

	return nil
}

// cleanupBackupFile removes the backup file after a successful update
func cleanupBackupFile(backupPath string) error {
	LogInfo("Cleaning up backup file after successful update...")
	LogInfo("Backup file path: %s", backupPath)

	// Check if backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		LogWarning("Backup file not found at: %s (may have been already deleted)", backupPath)
		return nil
	}

	// Delete the backup file
	if err := os.Remove(backupPath); err != nil {
		LogError("Failed to delete backup file: %v", err)
		return fmt.Errorf("failed to delete backup file: %w", err)
	}

	LogInfo("Backup file deleted successfully: %s", backupPath)
	return nil
}
