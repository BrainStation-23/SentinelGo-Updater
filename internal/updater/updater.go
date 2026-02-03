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
	CheckInterval        = 30 * time.Second
	MainAgentModule      = "github.com/BrainStation-23/SentinelGo"
	MainAgentServiceName = "sentinelgo"
)

var (
	serviceManager service.Manager
)

func init() {
	serviceManager = service.NewManager()
}

// setEnvironmentVariables ensures required environment variables are set for child processes
func setEnvironmentVariables() error {
	LogInfo("Setting up environment variables for update process...")

	// Ensure $HOME is set using platform-specific function
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

func Run() {
	if err := InitLogger(); err != nil {
		log.Fatalf("Failed to initialize logging system: %v", err)
	}
	defer CloseLogger()

	LogInfo("Updater service started")
	LogInfo("Check interval: %v", CheckInterval)
	LogInfo("Main agent module: %s", MainAgentModule)

	// Set up environment variables at startup
	LogInfo("Setting up environment variables...")
	if err := setEnvironmentVariables(); err != nil {
		LogError("Failed to set up environment variables: %v", err)
		LogWarning("Continuing anyway, but some operations may fail")
	} else {
		LogInfo("Environment variables configured successfully")
	}

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

func getInstalledVersion() (string, error) {
	binaryPath, detectionMethod, err := getMainAgentBinaryPathWithDetails()
	if err != nil {
		LogError("Failed to detect binary path: %v", err)
		LogWarning("Will retry detection on next update check")
		LogInfo("Detection will be retried in %v", CheckInterval)
		return "", fmt.Errorf("binary path detection failed: %w", err)
	}

	LogInfo("Binary path successfully detected using method: %s", detectionMethod)
	LogInfo("Using binary at: %s", binaryPath)

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		LogError("Binary not found at detected path: %s", binaryPath)
		LogWarning("Will retry on next check")
		return "", fmt.Errorf("main agent binary not found at %s", binaryPath)
	}

	cmd := exec.Command(binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		LogError("Failed to get version from binary at %s: %v", binaryPath, err)
		LogWarning("Binary may be corrupted or incompatible")
		LogWarning("Will retry on next check")
		return "", fmt.Errorf("failed to get version from binary: %w", err)
	}

	version := strings.TrimSpace(string(output))
	if version == "" {
		LogError("Binary at %s returned empty version", binaryPath)
		LogWarning("This may indicate an incompatible or corrupted binary")
		return "", fmt.Errorf("binary returned empty version")
	}

	versionParts := strings.Fields(version)
	for _, part := range versionParts {
		if len(part) > 1 && part[0] == 'v' && part[1] >= '0' && part[1] <= '9' {
			return part, nil
		}
	}

	LogWarning("Could not extract version number from output: %s", version)
	return version, nil
}

func getMainAgentBinaryPathWithDetails() (path string, method string, err error) {
	// Try to get binary path from paths package
	detectedPath := paths.GetMainAgentBinaryPath()

	// Check if binary exists at system location
	if _, err := os.Stat(detectedPath); err == nil {
		method = inferDetectionMethod(detectedPath)
		return detectedPath, method, nil
	}

	// If not found at system location, try platform-specific paths
	possiblePaths := getPossibleBinaryPaths()
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			method = "user_gopath_location"
			return path, method, nil
		}
	}

	return "", "", fmt.Errorf("binary not found at %s or any fallback location", detectedPath)
}

func inferDetectionMethod(detectedPath string) string {
	configPath := filepath.Join(paths.GetDataDirectory(), "updater-config.json")
	if _, err := os.Stat(configPath); err == nil {
		return "manual_configuration"
	}

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

	commonPaths := getCommonInstallationPaths()
	for _, commonPath := range commonPaths {
		if detectedPath == commonPath {
			return "common_installation_directory"
		}
	}

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

	return "auto_detection"
}

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

func getLatestVersion() (string, error) {
	goBinary, err := findGoBinary()
	if err != nil {
		return "", fmt.Errorf("go command not found: %w", err)
	}
	LogInfo("Using go binary: %s", goBinary)

	cmd := exec.Command(goBinary, "list", "-m", "-json", fmt.Sprintf("%s@latest", MainAgentModule))
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to query latest version: %w", err)
	}

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

func findGoBinary() (string, error) {
	if path, err := exec.LookPath("go"); err == nil {
		return path, nil
	}

	commonPaths := []string{
		"/usr/local/go/bin/go",
		"/opt/homebrew/bin/go",
		"/usr/local/bin/go",
		"/opt/local/bin/go",
	}

	if home := os.Getenv("HOME"); home != "" {
		commonPaths = append(commonPaths, filepath.Join(home, "go", "bin", "go"))
		commonPaths = append(commonPaths, filepath.Join(home, ".go", "bin", "go"))
	}

	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		if runtime.GOOS == "darwin" {
			userHome := filepath.Join("/Users", sudoUser)
			commonPaths = append(commonPaths, filepath.Join(userHome, "go", "bin", "go"))
			commonPaths = append(commonPaths, filepath.Join(userHome, ".go", "bin", "go"))
		} else {
			userHome := filepath.Join("/home", sudoUser)
			commonPaths = append(commonPaths, filepath.Join(userHome, "go", "bin", "go"))
			commonPaths = append(commonPaths, filepath.Join(userHome, ".go", "bin", "go"))
		}
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("go binary not found in PATH or common locations")
}

func isNewerVersion(current, latest string) bool {
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	if current == latest {
		return false
	}

	currentParts := parseVersion(current)
	latestParts := parseVersion(latest)

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

func parseVersion(version string) [3]int {
	var parts [3]int
	segments := strings.Split(version, ".")
	for i := 0; i < len(segments) && i < 3; i++ {
		var num int
		fmt.Sscanf(segments[i], "%d", &num)
		parts[i] = num
	}
	return parts
}

func performUpdate(targetVersion string) error {
	LogInfo("=== Starting update to %s ===", targetVersion)

	currentVersion, err := getInstalledVersion()
	if err != nil {
		LogWarning("Could not get current version: %v", err)
		LogWarning("This may indicate the binary is not properly installed")
		currentVersion = "unknown"
		if currentVersion == "unknown" {
			LogError("Cannot proceed with update - current binary not detected")
			LogError("Please ensure sentinel is properly installed before updating")
			return fmt.Errorf("cannot update: current binary not detected: %w", err)
		}
	}

	LogInfo("Creating backup before update...")
	backup, err := createBackup(currentVersion)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	updateErr := func() error {
		LogInfo("Step 1: Stopping main agent service...")
		if err := serviceManager.Stop(MainAgentServiceName); err != nil {
			return fmt.Errorf("failed to stop main agent: %w", err)
		}
		LogInfo("Main agent service stopped successfully")

		LogInfo("Step 2: Uninstalling main agent service...")
		if err := serviceManager.Uninstall(MainAgentServiceName); err != nil {
			return fmt.Errorf("failed to uninstall main agent: %w", err)
		}
		LogInfo("Main agent service uninstalled successfully")

		LogInfo("Step 3: Cleaning up old files...")
		if err := cleanupOldFiles(); err != nil {
			LogWarning("Cleanup failed: %v", err)
		}
		LogInfo("Cleanup completed")

		LogInfo("Step 4: Downloading and compiling version %s...", targetVersion)
		newBinaryPath, err := downloadAndCompile(targetVersion)
		if err != nil {
			return fmt.Errorf("failed to compile: %w", err)
		}
		LogInfo("Compilation successful, binary at: %s", newBinaryPath)

		LogInfo("Step 5: Installing new binary...")
		if err := installBinary(newBinaryPath); err != nil {
			return fmt.Errorf("failed to install binary: %w", err)
		}
		LogInfo("Binary installed successfully")

		LogInfo("Step 6: Reinstalling main agent service...")
		installedBinaryPath, detectionMethod, detectErr := getMainAgentBinaryPathWithDetails()
		if detectErr != nil {
			LogError("Failed to detect newly installed binary: %v", detectErr)
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

		LogInfo("Step 7: Starting main agent service...")
		if err := serviceManager.Start(MainAgentServiceName); err != nil {
			return fmt.Errorf("failed to start service: %w", err)
		}
		LogInfo("Service started successfully")

		LogInfo("Step 8: Verifying main agent is running...")
		if err := verifyMainAgentRunning(); err != nil {
			LogError("Service verification failed: %v", err)
			return fmt.Errorf("service not running after update: %w", err)
		}
		LogInfo("Main agent verified running")

		return nil
	}()

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

	LogInfo("Update completed successfully, cleaning up backup file...")
	if err := cleanupBackupFile(backup.BackupPath); err != nil {
		LogWarning("Failed to clean up backup file: %v", err)
		LogWarning("Backup file may need to be manually deleted: %s", backup.BackupPath)
	}

	LogInfo("=== Update completed successfully ===")
	return nil
}

func cleanupOldFiles() error {
	var errors []string

	binaryPath := paths.GetMainAgentBinaryPath()
	LogInfo("Deleting main agent binary: %s", binaryPath)
	if err := os.Remove(binaryPath); err != nil && !os.IsNotExist(err) {
		errors = append(errors, fmt.Sprintf("failed to delete binary %s: %v", binaryPath, err))
	} else if err == nil {
		LogInfo("Deleted: %s", binaryPath)
	}

	backupOldPath := binaryPath + ".old"
	LogInfo("Checking for legacy backup file: %s", backupOldPath)
	if err := os.Remove(backupOldPath); err != nil && !os.IsNotExist(err) {
		errors = append(errors, fmt.Sprintf("failed to delete legacy backup %s: %v", backupOldPath, err))
	} else if err == nil {
		LogInfo("Deleted legacy backup: %s", backupOldPath)
	} else if os.IsNotExist(err) {
		LogInfo("No legacy backup file found (this is normal)")
	}

	backupPath := binaryPath + ".backup"
	LogInfo("Checking for current backup file: %s", backupPath)
	if _, err := os.Stat(backupPath); err == nil {
		LogInfo("Preserving backup file for potential rollback: %s", backupPath)
	} else if os.IsNotExist(err) {
		LogWarning("Backup file not found at: %s", backupPath)
		LogWarning("Rollback will not be possible if update fails")
	}

	dbPath := paths.GetDatabasePath()
	if _, err := os.Stat(dbPath); err == nil {
		LogInfo("Database preserved at: %s", dbPath)
	} else if os.IsNotExist(err) {
		LogInfo("Database does not exist yet at: %s", dbPath)
	}

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

func downloadAndCompile(version string) (string, error) {
	LogInfo("Setting up Go environment for compilation...")

	goBinary, err := findGoBinary()
	if err != nil {
		return "", fmt.Errorf("go command not found: %w", err)
	}
	LogInfo("Using go binary: %s", goBinary)

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		homeDir, err := ensureHomeDirectory()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		gopath = filepath.Join(homeDir, "go")
		LogInfo("GOPATH not set, using default: %s", gopath)
	}

	goroot := os.Getenv("GOROOT")
	if goroot == "" {
		cmd := exec.Command(goBinary, "env", "GOROOT")
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

	env := os.Environ()
	env = append(env, "CGO_ENABLED=1")
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

	moduleWithVersion := fmt.Sprintf("%s/cmd/sentinel@%s", MainAgentModule, version)
	LogInfo("Executing: %s install %s", goBinary, moduleWithVersion)

	cmd := exec.Command(goBinary, "install", moduleWithVersion)
	cmd.Env = env

	output, err := cmd.CombinedOutput()

	if len(output) > 0 {
		LogInfo("Compilation output:\n%s", string(output))
	}

	if err != nil {
		LogError("Compilation failed: %v", err)
		LogError("Output: %s", string(output))
		return "", fmt.Errorf("compilation failed: %w\nOutput: %s", err, string(output))
	}

	binaryName := "sentinel"
	if runtime.GOOS == "windows" {
		binaryName = "sentinel.exe"
	}
	compiledBinaryPath := filepath.Join(gopath, "bin", binaryName)

	if _, err := os.Stat(compiledBinaryPath); os.IsNotExist(err) {
		LogError("Compiled binary not found at expected location: %s", compiledBinaryPath)
		return "", fmt.Errorf("compiled binary not found at expected location: %s", compiledBinaryPath)
	}

	LogInfo("Compilation successful, binary located at: %s", compiledBinaryPath)
	return compiledBinaryPath, nil
}

func installBinary(sourcePath string) error {
	targetPath := paths.GetMainAgentBinaryPath()
	LogInfo("Installing binary from %s to %s", sourcePath, targetPath)

	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source binary: %w", err)
	}

	if err := os.WriteFile(targetPath, sourceData, 0755); err != nil {
		return fmt.Errorf("failed to write target binary: %w", err)
	}

	LogInfo("Binary written to: %s", targetPath)

	if runtime.GOOS != "windows" {
		if err := os.Chmod(targetPath, 0755); err != nil {
			return fmt.Errorf("failed to set executable permissions: %w", err)
		}
		LogInfo("Set executable permissions (0755) on: %s", targetPath)

		if os.Geteuid() == 0 {
			if err := os.Chown(targetPath, 0, 0); err != nil {
				LogWarning("Failed to set ownership to root: %v", err)
			} else {
				LogInfo("Set ownership to root:root on: %s", targetPath)
			}
		}
	}

	fileInfo, err := os.Stat(targetPath)
	if err != nil {
		return fmt.Errorf("failed to verify installed binary: %w", err)
	}

	if runtime.GOOS != "windows" {
		if fileInfo.Mode()&0111 == 0 {
			return fmt.Errorf("binary is not executable")
		}
	}

	LogInfo("Binary installation verified successfully")
	return nil
}

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

type BackupInfo struct {
	Version    string
	BackupPath string
	BinaryPath string
	Timestamp  time.Time
}

func createBackup(currentVersion string) (*BackupInfo, error) {
	LogInfo("Creating backup of current binary...")

	binaryPath := paths.GetMainAgentBinaryPath()

	// Check if binary exists at system location
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// If not found at system location, try platform-specific paths
		possiblePaths := getPossibleBinaryPaths()
		LogInfo("Binary not found at system location, checking %d possible locations...", len(possiblePaths))
		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				LogInfo("Found binary for backup at: %s", path)
				binaryPath = path
				break
			}
		}

		// If still not found, return error
		if binaryPath == paths.GetMainAgentBinaryPath() {
			return nil, fmt.Errorf("current binary not found at %s or any fallback location", binaryPath)
		}
	}

	backupPath := binaryPath + ".backup"

	LogInfo("Reading current binary from: %s", binaryPath)
	binaryData, err := os.ReadFile(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read current binary: %w", err)
	}

	LogInfo("Writing backup to: %s", backupPath)
	if err := os.WriteFile(backupPath, binaryData, 0755); err != nil {
		return nil, fmt.Errorf("failed to write backup file: %w", err)
	}

	backupInfo, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to verify backup file: %w", err)
	}

	backup := &BackupInfo{
		Version:    currentVersion,
		BackupPath: backupPath,
		BinaryPath: binaryPath,
		Timestamp:  time.Now(),
	}

	LogInfo("Backup created successfully:")
	LogInfo("  Version: %s", backup.Version)
	LogInfo("  Path: %s", backup.BackupPath)
	LogInfo("  Binary Path: %s", backup.BinaryPath)
	LogInfo("  Size: %d bytes", backupInfo.Size())
	LogInfo("  Timestamp: %s", backup.Timestamp.Format(time.RFC3339))

	return backup, nil
}

func rollback(backup *BackupInfo) error {
	LogInfo("=== Starting rollback process ===")
	LogInfo("Rolling back to version: %s", backup.Version)
	LogInfo("Backup path: %s", backup.BackupPath)

	LogInfo("Step 1: Verifying backup file exists...")
	if _, err := os.Stat(backup.BackupPath); os.IsNotExist(err) {
		LogCritical("Backup file not found at %s", backup.BackupPath)
		return fmt.Errorf("backup file not found at %s - manual recovery required", backup.BackupPath)
	}
	LogInfo("Backup file verified")

	LogInfo("Step 2: Restoring binary from backup...")
	binaryPath := backup.BinaryPath
	LogInfo("Restoring to original binary path: %s", binaryPath)

	backupData, err := os.ReadFile(backup.BackupPath)
	if err != nil {
		LogCritical("Failed to read backup file: %v", err)
		return fmt.Errorf("failed to read backup file: %w - manual recovery may be required", err)
	}

	targetDir := filepath.Dir(binaryPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		LogCritical("Failed to create target directory: %v", err)
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	if err := os.WriteFile(binaryPath, backupData, 0755); err != nil {
		LogCritical("Failed to restore binary: %v", err)
		return fmt.Errorf("failed to restore binary: %w - manual recovery required", err)
	}
	LogInfo("Binary restored to: %s", binaryPath)

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

	LogInfo("Step 3: Reinstalling service...")
	if err := serviceManager.Install(MainAgentServiceName, binaryPath); err != nil {
		LogError("Failed to reinstall service: %v", err)
		return fmt.Errorf("failed to reinstall service: %w - manual service installation required", err)
	}
	LogInfo("Service reinstalled successfully")

	LogInfo("Step 4: Starting service...")
	if err := serviceManager.Start(MainAgentServiceName); err != nil {
		LogError("Failed to start service: %v", err)
		return fmt.Errorf("failed to start service: %w - manual service start required", err)
	}
	LogInfo("Service started successfully")

	LogInfo("Step 5: Verifying service is running...")
	if err := verifyMainAgentRunning(); err != nil {
		LogError("Service not running after rollback: %v", err)
		return fmt.Errorf("service not running after rollback: %w - manual verification required", err)
	}
	LogInfo("Service verified running")

	LogInfo("=== Rollback completed successfully to version %s ===", backup.Version)
	LogInfo("Backup file preserved for manual inspection at: %s", backup.BackupPath)
	return nil
}

func cleanupBackupFile(backupPath string) error {
	LogInfo("Cleaning up backup file after successful update...")
	LogInfo("Backup file path: %s", backupPath)

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		LogWarning("Backup file not found at: %s (may have been already deleted)", backupPath)
		return nil
	}

	if err := os.Remove(backupPath); err != nil {
		LogError("Failed to delete backup file: %v", err)
		return fmt.Errorf("failed to delete backup file: %w", err)
	}

	LogInfo("Backup file deleted successfully: %s", backupPath)
	return nil
}
