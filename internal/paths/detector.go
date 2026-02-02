package paths

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// BinaryDetector handles dynamic detection of the main agent binary path
// with thread-safe caching and multiple detection strategies
type BinaryDetector struct {
	cachedPath     string
	configPath     string
	lastValidation time.Time
	mu             sync.RWMutex
}

// DetectionResult contains information about a detection attempt
type DetectionResult struct {
	Path   string
	Method string
	Error  error
}

// DetectionError contains detailed information about a failed detection attempt
type DetectionError struct {
	Method      string // Detection method name (e.g., "service_config")
	Description string // Human-readable description
	Error       error  // The actual error that occurred
	Attempted   bool   // Whether this method was attempted
	PathFound   string // Path that was found but failed validation (if any)
}

// UpdaterConfig represents the optional configuration file for manual path override
type UpdaterConfig struct {
	BinaryPath          string `json:"binaryPath"`
	EnableAutoDetection bool   `json:"enableAutoDetection"`
}

var (
	detector     *BinaryDetector
	detectorOnce sync.Once
)

// GetDetector returns the singleton BinaryDetector instance
func GetDetector() *BinaryDetector {
	detectorOnce.Do(func() {
		detector = &BinaryDetector{
			configPath: loadConfigPath(),
		}
	})
	return detector
}

// DetectBinaryPath attempts to find the main agent binary using multiple strategies
// It tries methods in priority order and caches the result for performance
func (d *BinaryDetector) DetectBinaryPath() (string, error) {
	// Try cached path first
	if cached, ok := d.getCachedPath(); ok {
		fmt.Printf("[INFO] Using cached binary path: %s\n", cached)
		if d.validateBinaryPath(cached) {
			fmt.Println("[INFO] Cached path is valid")
			return cached, nil
		}
		// Cache is stale, invalidate it
		fmt.Println("[WARN] Cached path is no longer valid, invalidating cache")
		d.invalidateCache()
	}

	// Try manual config override
	if d.configPath != "" {
		fmt.Printf("[INFO] Attempting to use manually configured path: %s\n", d.configPath)
		if d.validateBinaryPath(d.configPath) {
			fmt.Println("[INFO] Manually configured path is valid")
			d.setCachedPath(d.configPath)
			return d.configPath, nil
		}
		// Log warning but continue to auto-detection
		fmt.Printf("[WARN] Configured path invalid: %s, falling back to auto-detection\n", d.configPath)
	}

	// Try each detection strategy in order
	strategies := []struct {
		name        string
		description string
		fn          func() (string, error)
	}{
		{"service_config", "System service configuration", d.detectFromServiceConfig},
		{"running_process", "Running process detection", d.detectFromRunningProcess},
		{"path_search", "PATH environment variable", d.detectFromPATH},
		{"common_paths", "Common installation directories", d.detectFromCommonPaths},
	}

	var detectionErrors []DetectionError
	fmt.Println("[INFO] Starting binary path detection...")
	fmt.Printf("[INFO] Platform: %s, Architecture: %s\n", runtime.GOOS, runtime.GOARCH)

	for _, strategy := range strategies {
		fmt.Printf("[INFO] Attempting detection method: %s (%s)\n", strategy.name, strategy.description)

		path, err := strategy.fn()
		if err != nil {
			detectionErr := DetectionError{
				Method:      strategy.name,
				Description: strategy.description,
				Error:       err,
				Attempted:   true,
			}
			detectionErrors = append(detectionErrors, detectionErr)
			fmt.Printf("[WARN] Detection method %s failed: %v\n", strategy.name, err)
			continue
		}

		fmt.Printf("[INFO] Detection method %s returned path: %s\n", strategy.name, path)

		// Validate the returned path
		validationErr := d.validateBinaryPathWithDetails(path)
		if validationErr != nil {
			detectionErr := DetectionError{
				Method:      strategy.name,
				Description: strategy.description,
				Error:       fmt.Errorf("path validation failed: %w", validationErr),
				Attempted:   true,
				PathFound:   path,
			}
			detectionErrors = append(detectionErrors, detectionErr)
			fmt.Printf("[WARN] Detection method %s returned invalid path: %s (reason: %v)\n", strategy.name, path, validationErr)
			continue
		}

		fmt.Printf("[INFO] Binary successfully detected using %s: %s\n", strategy.name, path)
		d.setCachedPath(path)
		return path, nil
	}

	// All methods failed - generate comprehensive error report
	return "", d.generateDetailedError(detectionErrors)
}

// validateBinaryPath checks if the given path points to a valid executable file
func (d *BinaryDetector) validateBinaryPath(path string) bool {
	return d.validateBinaryPathWithDetails(path) == nil
}

// validateBinaryPathWithDetails checks if the given path points to a valid executable file
// and returns detailed error information if validation fails
func (d *BinaryDetector) validateBinaryPathWithDetails(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}

	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist")
		}
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied accessing file")
		}
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Check if it's a regular file (not a directory)
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file")
	}

	// Check if file is executable (Unix-like systems)
	if runtime.GOOS != "windows" {
		mode := info.Mode()
		if mode&0111 == 0 {
			// No execute permission bits set
			return fmt.Errorf("file is not executable (missing execute permissions)")
		}
	}

	return nil
}

// getCachedPath retrieves the cached binary path if it exists and is still valid
func (d *BinaryDetector) getCachedPath() (string, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.cachedPath == "" {
		return "", false
	}

	return d.cachedPath, true
}

// setCachedPath stores the detected binary path in cache
func (d *BinaryDetector) setCachedPath(path string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.cachedPath = path
	d.lastValidation = time.Now()
}

// invalidateCache clears the cached binary path
func (d *BinaryDetector) invalidateCache() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.cachedPath = ""
	d.lastValidation = time.Time{}
}

// RefreshCache forces a re-detection of the binary path
func (d *BinaryDetector) RefreshCache() error {
	d.invalidateCache()
	_, err := d.DetectBinaryPath()
	return err
}

// loadConfigPath reads the optional configuration file and returns the manual path if specified
func loadConfigPath() string {
	configPath := filepath.Join(GetDataDirectory(), "updater-config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// No config file or can't read it - not an error, just use auto-detection
		return ""
	}

	var config UpdaterConfig
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Printf("[WARN] Failed to parse updater config: %v\n", err)
		return ""
	}

	return config.BinaryPath
}

// detectFromServiceConfig attempts to detect the binary path from service configuration
// Platform-specific implementations are in detector_*.go files
func (d *BinaryDetector) detectFromServiceConfig() (string, error) {
	return detectFromServiceConfigImpl()
}

// detectFromRunningProcess attempts to detect the binary path from a running process
// Platform-specific implementations are in detector_*.go files
func (d *BinaryDetector) detectFromRunningProcess() (string, error) {
	return detectFromRunningProcessImpl()
}

// detectFromPATH searches for the binary in PATH environment variable
func (d *BinaryDetector) detectFromPATH() (string, error) {
	binaryName := "sentinel"
	if runtime.GOOS == "windows" {
		binaryName = "sentinel.exe"
	}

	path, err := searchPATH(binaryName)
	if err != nil {
		// Enhance error with more details
		pathEnv := os.Getenv("PATH")
		if pathEnv == "" {
			return "", fmt.Errorf("PATH environment variable is empty or not set")
		}

		separator := ":"
		if runtime.GOOS == "windows" {
			separator = ";"
		}
		paths := strings.Split(pathEnv, separator)
		dirCount := 0
		for _, p := range paths {
			if p != "" {
				dirCount++
			}
		}

		return "", fmt.Errorf("binary '%s' not found in PATH (searched %d directories)", binaryName, dirCount)
	}

	return path, nil
}

// detectFromCommonPaths searches common installation directories
func (d *BinaryDetector) detectFromCommonPaths() (string, error) {
	commonPaths := getCommonPaths()

	var checkedPaths []string
	var accessErrors []string

	for _, path := range commonPaths {
		checkedPaths = append(checkedPaths, path)

		// Check if path exists and get detailed error
		info, err := os.Stat(path)
		if err != nil {
			if os.IsPermission(err) {
				accessErrors = append(accessErrors, fmt.Sprintf("%s (permission denied)", path))
			}
			continue
		}

		// Path exists, check if it's valid
		if info.IsDir() {
			continue
		}

		if d.validateBinaryPath(path) {
			return path, nil
		}
	}

	errorMsg := fmt.Sprintf("binary not found in %d common installation paths", len(commonPaths))
	if len(accessErrors) > 0 {
		errorMsg += fmt.Sprintf(" (permission denied for: %s)", strings.Join(accessErrors, ", "))
	}

	return "", fmt.Errorf("%s", errorMsg)
}

// searchPATH looks for the binary in PATH environment variable
func searchPATH(binaryName string) (string, error) {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return "", fmt.Errorf("PATH environment variable is empty")
	}

	// Platform-specific path separator
	separator := ":"
	if runtime.GOOS == "windows" {
		separator = ";"
	}

	paths := strings.Split(pathEnv, separator)
	for _, dir := range paths {
		if dir == "" {
			continue
		}

		fullPath := filepath.Join(dir, binaryName)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			return fullPath, nil
		}
	}

	return "", fmt.Errorf("binary %s not found in PATH", binaryName)
}

// getCommonPaths returns platform-specific common installation paths
func getCommonPaths() []string {
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

// generateDetailedError creates a comprehensive error message with troubleshooting steps
func (d *BinaryDetector) generateDetailedError(errors []DetectionError) error {
	var errorMsg strings.Builder

	errorMsg.WriteString("\n")
	errorMsg.WriteString("================================================================================\n")
	errorMsg.WriteString("SENTINEL BINARY PATH DETECTION FAILED\n")
	errorMsg.WriteString("================================================================================\n")
	errorMsg.WriteString("\n")
	errorMsg.WriteString(fmt.Sprintf("Platform: %s (%s)\n", runtime.GOOS, runtime.GOARCH))
	errorMsg.WriteString(fmt.Sprintf("Timestamp: %s\n", time.Now().Format(time.RFC3339)))
	errorMsg.WriteString("\n")
	errorMsg.WriteString("All detection methods failed to locate the sentinel binary.\n")
	errorMsg.WriteString("\n")
	errorMsg.WriteString("ATTEMPTED DETECTION METHODS:\n")
	errorMsg.WriteString("--------------------------------------------------------------------------------\n")

	for i, err := range errors {
		errorMsg.WriteString(fmt.Sprintf("\n%d. %s (%s)\n", i+1, err.Description, err.Method))
		errorMsg.WriteString("   Status: FAILED\n")
		if err.PathFound != "" {
			errorMsg.WriteString(fmt.Sprintf("   Path Found: %s\n", err.PathFound))
		}
		errorMsg.WriteString(fmt.Sprintf("   Error: %v\n", err.Error))
	}

	errorMsg.WriteString("\n")
	errorMsg.WriteString("TROUBLESHOOTING STEPS:\n")
	errorMsg.WriteString("--------------------------------------------------------------------------------\n")
	errorMsg.WriteString("\n")

	// Platform-specific troubleshooting
	switch runtime.GOOS {
	case "linux":
		errorMsg.WriteString("For Linux systems:\n")
		errorMsg.WriteString("\n")
		errorMsg.WriteString("1. Verify the sentinel binary is installed:\n")
		errorMsg.WriteString("   $ which sentinel\n")
		errorMsg.WriteString("   $ ls -la /usr/local/bin/sentinel\n")
		errorMsg.WriteString("   $ ls -la ~/go/bin/sentinel\n")
		errorMsg.WriteString("\n")
		errorMsg.WriteString("2. Check if the sentinel service is configured:\n")
		errorMsg.WriteString("   $ systemctl status sentinelgo\n")
		errorMsg.WriteString("   $ cat /etc/systemd/system/sentinelgo.service\n")
		errorMsg.WriteString("\n")
		errorMsg.WriteString("3. Verify the binary has execute permissions:\n")
		errorMsg.WriteString("   $ chmod +x /path/to/sentinel\n")
		errorMsg.WriteString("\n")
		errorMsg.WriteString("4. Check if the binary is in your PATH:\n")
		errorMsg.WriteString("   $ echo $PATH\n")
		errorMsg.WriteString("\n")

	case "darwin":
		errorMsg.WriteString("For macOS systems:\n")
		errorMsg.WriteString("\n")
		errorMsg.WriteString("1. Verify the sentinel binary is installed:\n")
		errorMsg.WriteString("   $ which sentinel\n")
		errorMsg.WriteString("   $ ls -la /usr/local/bin/sentinel\n")
		errorMsg.WriteString("   $ ls -la ~/go/bin/sentinel\n")
		errorMsg.WriteString("\n")
		errorMsg.WriteString("2. Check if the sentinel service is configured:\n")
		errorMsg.WriteString("   $ launchctl list | grep sentinel\n")
		errorMsg.WriteString("   $ cat /Library/LaunchDaemons/com.sentinelgo.plist\n")
		errorMsg.WriteString("\n")
		errorMsg.WriteString("3. Verify the binary has execute permissions:\n")
		errorMsg.WriteString("   $ chmod +x /path/to/sentinel\n")
		errorMsg.WriteString("\n")
		errorMsg.WriteString("4. Check if the binary is in your PATH:\n")
		errorMsg.WriteString("   $ echo $PATH\n")
		errorMsg.WriteString("\n")

	case "windows":
		errorMsg.WriteString("For Windows systems:\n")
		errorMsg.WriteString("\n")
		errorMsg.WriteString("1. Verify the sentinel binary is installed:\n")
		errorMsg.WriteString("   > where sentinel.exe\n")
		errorMsg.WriteString("   > dir \"C:\\Program Files\\SentinelGo\\sentinel.exe\"\n")
		errorMsg.WriteString("   > dir \"%USERPROFILE%\\go\\bin\\sentinel.exe\"\n")
		errorMsg.WriteString("\n")
		errorMsg.WriteString("2. Check if the sentinel service is configured:\n")
		errorMsg.WriteString("   > sc query sentinelgo\n")
		errorMsg.WriteString("   > sc qc sentinelgo\n")
		errorMsg.WriteString("\n")
		errorMsg.WriteString("3. Check if the binary is in your PATH:\n")
		errorMsg.WriteString("   > echo %PATH%\n")
		errorMsg.WriteString("\n")
	}

	errorMsg.WriteString("5. Manual configuration option:\n")
	errorMsg.WriteString("   Create a configuration file to manually specify the binary path:\n")
	errorMsg.WriteString(fmt.Sprintf("   File: %s\n", filepath.Join(GetDataDirectory(), "updater-config.json")))
	errorMsg.WriteString("   Content:\n")
	errorMsg.WriteString("   {\n")
	errorMsg.WriteString("     \"binaryPath\": \"/full/path/to/sentinel\",\n")
	errorMsg.WriteString("     \"enableAutoDetection\": true\n")
	errorMsg.WriteString("   }\n")
	errorMsg.WriteString("\n")

	errorMsg.WriteString("6. Reinstall the sentinel binary:\n")
	errorMsg.WriteString("   If the binary is missing, reinstall it using:\n")
	errorMsg.WriteString("   $ go install github.com/BrainStation-23/SentinelGo/cmd/sentinel@latest\n")
	errorMsg.WriteString("\n")

	errorMsg.WriteString("SEARCHED LOCATIONS:\n")
	errorMsg.WriteString("--------------------------------------------------------------------------------\n")
	errorMsg.WriteString("\n")

	// List all searched paths
	commonPaths := getCommonPaths()
	errorMsg.WriteString("Common installation directories checked:\n")
	for _, path := range commonPaths {
		errorMsg.WriteString(fmt.Sprintf("  - %s\n", path))
	}
	errorMsg.WriteString("\n")

	// PATH directories
	pathEnv := os.Getenv("PATH")
	if pathEnv != "" {
		separator := ":"
		if runtime.GOOS == "windows" {
			separator = ";"
		}
		paths := strings.Split(pathEnv, separator)
		errorMsg.WriteString("PATH environment variable directories:\n")
		count := 0
		for _, path := range paths {
			if path != "" && count < 10 { // Limit to first 10 for readability
				errorMsg.WriteString(fmt.Sprintf("  - %s\n", path))
				count++
			}
		}
		if len(paths) > 10 {
			errorMsg.WriteString(fmt.Sprintf("  ... and %d more directories\n", len(paths)-10))
		}
		errorMsg.WriteString("\n")
	}

	errorMsg.WriteString("NEXT STEPS:\n")
	errorMsg.WriteString("--------------------------------------------------------------------------------\n")
	errorMsg.WriteString("\n")
	errorMsg.WriteString("The updater will continue running and retry detection on the next update check.\n")
	errorMsg.WriteString("Please resolve the issue using the troubleshooting steps above.\n")
	errorMsg.WriteString("\n")
	errorMsg.WriteString("For additional support, please contact your system administrator or visit:\n")
	errorMsg.WriteString("https://github.com/BrainStation-23/SentinelGo-Updater/issues\n")
	errorMsg.WriteString("\n")
	errorMsg.WriteString("================================================================================\n")

	// Log the detailed error
	fmt.Print(errorMsg.String())

	// Return a concise error for the caller
	return fmt.Errorf("failed to detect sentinel binary path after trying %d methods (see detailed error above)", len(errors))
}
