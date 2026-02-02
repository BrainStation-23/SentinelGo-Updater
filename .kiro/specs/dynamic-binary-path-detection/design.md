# Design Document: Dynamic Binary Path Detection for SentinelGo-Updater

## Overview

This document outlines the design for implementing dynamic binary path detection in the SentinelGo-Updater service. The current implementation uses hardcoded paths (`/usr/local/bin/sentinel` on Linux, `C:\Program Files\SentinelGo\sentinel.exe` on Windows) which fail when the binary is installed in different locations. This design enables the updater to automatically discover the binary location across 800+ computers with varying installation paths and operating systems.

### Current Problem

The `internal/paths/paths.go` file contains hardcoded binary paths:
- Linux/macOS: `/usr/local/bin/sentinel`
- Windows: `C:\Program Files\SentinelGo\sentinel.exe`

However, the actual binary can be located in various places:
- User-specific Go bin: `~/go/bin/sentinel`
- Custom installation paths
- Different drive letters on Windows
- Non-standard directories

The updater fails with: `"main agent binary not found at /usr/local/bin/sentinel"`

### Solution Approach

Implement a multi-strategy detection system that:
1. Queries the service configuration (most reliable)
2. Falls back to PATH environment variable search
3. Falls back to common installation directories
4. Caches the detected path for performance
5. Supports manual override via configuration

## Architecture

### Detection Strategy Priority

The updater will attempt detection methods in this order:

```
1. Service Configuration Query (highest priority)
   ├─ Linux: Parse systemd unit file ExecStart
   ├─ macOS: Parse launchd plist ProgramArguments
   └─ Windows: Query Windows Service registry/API

2. Running Process Query
   └─ Query OS for running sentinel process and extract binary path

3. PATH Environment Variable Search
   └─ Search all directories in PATH for sentinel binary

4. Common Installation Directories
   ├─ /usr/local/bin
   ├─ /usr/bin
   ├─ ~/go/bin
   ├─ C:\Program Files\SentinelGo
   └─ Other platform-specific paths

5. Manual Configuration Override
   └─ Read from config file if specified
```

### Caching Strategy

Once a valid path is detected:
- Cache it in memory for the lifetime of the updater process
- Validate cached path before each use
- Invalidate cache if validation fails
- Re-detect using full strategy chain

## Components and Interfaces

### 1. Binary Path Detector (`internal/paths/detector.go`)

New file that implements the detection logic.

```go
package paths

type BinaryDetector struct {
    cachedPath string
    configPath string // manual override from config
}

// DetectBinaryPath attempts to find the main agent binary using multiple strategies
func (d *BinaryDetector) DetectBinaryPath() (string, error)

// Strategy methods (called in order)
func (d *BinaryDetector) detectFromServiceConfig() (string, error)
func (d *BinaryDetector) detectFromRunningProcess() (string, error)
func (d *BinaryDetector) detectFromPATH() (string, error)
func (d *BinaryDetector) detectFromCommonPaths() (string, error)

// Validation
func (d *BinaryDetector) validateBinaryPath(path string) bool

// Cache management
func (d *BinaryDetector) getCachedPath() (string, bool)
func (d *BinaryDetector) setCachedPath(path string)
func (d *BinaryDetector) invalidateCache()
```

### 2. Platform-Specific Service Parsers

#### Linux Service Parser (`internal/paths/detector_linux.go`)

```go
// parseSystemdUnitFile reads the systemd unit file and extracts ExecStart path
func parseSystemdUnitFile(serviceName string) (string, error) {
    // Read /etc/systemd/system/{serviceName}.service
    // Parse ExecStart= line
    // Extract binary path (first argument)
    // Handle both absolute and relative paths
}

// Example systemd unit file:
// [Service]
// ExecStart=/home/bs00927/go/bin/sentinel
```

#### macOS Service Parser (`internal/paths/detector_darwin.go`)

```go
// parseLaunchdPlist reads the launchd plist and extracts ProgramArguments
func parseLaunchdPlist(serviceName string) (string, error) {
    // Read /Library/LaunchDaemons/com.{serviceName}.plist
    // Parse ProgramArguments array
    // Extract first element (binary path)
}
```

#### Windows Service Parser (`internal/paths/detector_windows.go`)

```go
// queryWindowsService uses Windows API to get service executable path
func queryWindowsService(serviceName string) (string, error) {
    // Use golang.org/x/sys/windows to query service
    // Get ImagePath from service configuration
    // Handle quoted paths and arguments
}
```

### 3. Process Query Utilities

```go
// findRunningProcess searches for running sentinel process
func findRunningProcess(processName string) (string, error) {
    // Use platform-specific process enumeration
    // Linux: parse /proc filesystem
    // macOS: use ps command
    // Windows: use Windows API
}
```

### 4. PATH Search Utilities

```go
// searchPATH looks for binary in PATH environment variable
func searchPATH(binaryName string) (string, error) {
    // Get PATH environment variable
    // Split by path separator (: on Unix, ; on Windows)
    // Check each directory for binary
    // Return first match
}
```

### 5. Modified `paths.go`

Update existing file to use the detector:

```go
var (
    detector *BinaryDetector
    once     sync.Once
)

func initDetector() {
    once.Do(func() {
        detector = &BinaryDetector{
            configPath: loadConfigPath(), // from config file if exists
        }
    })
}

// GetMainAgentBinaryPath now uses dynamic detection
func GetMainAgentBinaryPath() string {
    initDetector()
    
    path, err := detector.DetectBinaryPath()
    if err != nil {
        // Log error but return fallback path for backward compatibility
        log.Printf("Failed to detect binary path: %v, using fallback", err)
        return getFallbackBinaryPath()
    }
    
    return path
}

// getFallbackBinaryPath returns the old hardcoded paths as last resort
func getFallbackBinaryPath() string {
    // Same logic as current GetBinaryDirectory()
}
```

### 6. Configuration File Support

Add optional configuration file: `/var/lib/sentinelgo/updater-config.json`

```json
{
  "binaryPath": "/custom/path/to/sentinel",
  "enableAutoDetection": true
}
```

```go
type UpdaterConfig struct {
    BinaryPath           string `json:"binaryPath"`
    EnableAutoDetection  bool   `json:"enableAutoDetection"`
}

func loadConfigPath() string {
    configPath := filepath.Join(GetDataDirectory(), "updater-config.json")
    data, err := os.ReadFile(configPath)
    if err != nil {
        return "" // No config file, use auto-detection
    }
    
    var config UpdaterConfig
    if err := json.Unmarshal(data, &config); err != nil {
        return ""
    }
    
    return config.BinaryPath
}
```

## Data Models

### BinaryDetector State

```go
type BinaryDetector struct {
    cachedPath     string        // Cached binary path
    configPath     string        // Manual override from config
    lastValidation time.Time     // Last time path was validated
    mu             sync.RWMutex  // Thread-safe access
}
```

### Detection Result

```go
type DetectionResult struct {
    Path     string
    Method   string // "service_config", "running_process", "path", "common_paths", "config"
    Error    error
}
```

## Error Handling

### Detection Failures

Each detection method returns an error if it fails. The detector logs each failure and tries the next method:

```go
func (d *BinaryDetector) DetectBinaryPath() (string, error) {
    // Try cached path first
    if cached, ok := d.getCachedPath(); ok {
        if d.validateBinaryPath(cached) {
            return cached, nil
        }
        d.invalidateCache()
    }
    
    // Try manual config
    if d.configPath != "" {
        if d.validateBinaryPath(d.configPath) {
            d.setCachedPath(d.configPath)
            return d.configPath, nil
        }
        log.Printf("Configured path invalid: %s", d.configPath)
    }
    
    // Try each detection strategy
    strategies := []struct{
        name string
        fn   func() (string, error)
    }{
        {"service_config", d.detectFromServiceConfig},
        {"running_process", d.detectFromRunningProcess},
        {"path_search", d.detectFromPATH},
        {"common_paths", d.detectFromCommonPaths},
    }
    
    var errors []string
    for _, strategy := range strategies {
        path, err := strategy.fn()
        if err != nil {
            errors = append(errors, fmt.Sprintf("%s: %v", strategy.name, err))
            continue
        }
        
        if d.validateBinaryPath(path) {
            log.Printf("Binary detected using %s: %s", strategy.name, path)
            d.setCachedPath(path)
            return path, nil
        }
        
        errors = append(errors, fmt.Sprintf("%s: path invalid: %s", strategy.name, path))
    }
    
    return "", fmt.Errorf("all detection methods failed: %s", strings.Join(errors, "; "))
}
```

### Detailed Error Logging

When all methods fail, log comprehensive diagnostics:

```
[ERROR] Failed to detect sentinel binary path
[ERROR] Attempted methods:
[ERROR]   1. Service config: failed to read /etc/systemd/system/sentinelgo.service: file not found
[ERROR]   2. Running process: no process named 'sentinel' found
[ERROR]   3. PATH search: searched 12 directories, binary not found
[ERROR]   4. Common paths: checked /usr/local/bin, /usr/bin, /home/user/go/bin - not found
[ERROR] 
[ERROR] Troubleshooting steps:
[ERROR]   - Verify sentinel binary is installed
[ERROR]   - Check service configuration file exists
[ERROR]   - Ensure binary is in PATH or standard location
[ERROR]   - Set manual path in /var/lib/sentinelgo/updater-config.json
```

## Testing Strategy

### Unit Tests

1. **Service Config Parsing Tests**
   - Test parsing valid systemd unit files
   - Test parsing launchd plist files
   - Test handling malformed config files
   - Test extracting paths with arguments
   - Test handling quoted paths

2. **PATH Search Tests**
   - Test finding binary in PATH
   - Test handling multiple PATH entries
   - Test handling non-existent directories in PATH
   - Test platform-specific path separators

3. **Common Paths Tests**
   - Test searching common directories
   - Test platform-specific paths
   - Test handling permission errors

4. **Caching Tests**
   - Test cache hit/miss scenarios
   - Test cache invalidation
   - Test concurrent access

5. **Validation Tests**
   - Test validating existing files
   - Test validating executable permissions
   - Test validating non-existent files

### Integration Tests

1. **End-to-End Detection**
   - Install binary in various locations
   - Verify detector finds it
   - Test on Linux, macOS, Windows

2. **Service Integration**
   - Create test service configurations
   - Verify detector extracts correct path
   - Test with running services

3. **Fallback Chain**
   - Disable each detection method
   - Verify fallback to next method
   - Test complete failure scenario

### Manual Testing Checklist

- [ ] Test on Linux with systemd service
- [ ] Test on Linux with binary in ~/go/bin
- [ ] Test on Linux with binary in /usr/local/bin
- [ ] Test on macOS with launchd service
- [ ] Test on Windows with Windows Service
- [ ] Test with binary in PATH
- [ ] Test with binary not in PATH
- [ ] Test with manual config override
- [ ] Test with invalid manual config
- [ ] Test cache invalidation on binary move
- [ ] Test concurrent detection calls
- [ ] Test with 800+ different installations (sampling)

## Implementation Plan

### Phase 1: Core Detection Framework

1. Create `internal/paths/detector.go` with base structure
2. Implement caching mechanism
3. Implement validation logic
4. Add configuration file support

### Phase 2: Platform-Specific Parsers

1. Implement Linux systemd parser (`detector_linux.go`)
2. Implement macOS launchd parser (`detector_darwin.go`)
3. Implement Windows service query (`detector_windows.go`)

### Phase 3: Fallback Strategies

1. Implement running process detection
2. Implement PATH search
3. Implement common paths search

### Phase 4: Integration

1. Modify `paths.go` to use detector
2. Update `updater.go` to handle detection errors gracefully
3. Add comprehensive logging

### Phase 5: Testing

1. Write unit tests for each component
2. Write integration tests
3. Test on all platforms
4. Test with various installation scenarios

## Performance Considerations

### Detection Performance

- **Service config parsing**: ~1-5ms (file I/O)
- **Running process query**: ~10-50ms (process enumeration)
- **PATH search**: ~5-20ms (depends on PATH length)
- **Common paths search**: ~10-30ms (multiple stat calls)

**Total worst-case**: ~100ms for first detection

### Caching Benefits

After first detection:
- **Cached path validation**: ~1ms (single stat call)
- **99% reduction in detection time**

### Memory Footprint

- Detector instance: ~200 bytes
- Cached path string: ~100 bytes
- Total: <1KB additional memory

## Security Considerations

### Path Validation

Always validate detected paths:
1. File exists
2. File is executable
3. File is not a symlink to unexpected location (optional)
4. File has expected permissions

### Configuration File Security

- Config file should be readable only by updater service user
- Validate config file permissions before reading
- Sanitize paths from config file

### Service Config Parsing

- Validate service config file permissions
- Handle malicious config files gracefully
- Don't execute or eval any content from config files

## Backward Compatibility

### Fallback to Hardcoded Paths

If all detection methods fail, fall back to current hardcoded paths:
- Maintains existing behavior for standard installations
- Prevents breaking existing deployments

### Gradual Rollout

1. Deploy with detection enabled but non-blocking
2. Log detection results for monitoring
3. After validation period, make detection mandatory
4. Remove hardcoded fallbacks in future version

## Monitoring and Observability

### Metrics to Track

- Detection method success rate (per method)
- Detection time (p50, p95, p99)
- Cache hit rate
- Detection failures by error type

### Logging

Log at INFO level:
- Successful detection with method used
- Cached path reuse
- Manual config override usage

Log at WARNING level:
- Detection method failures (non-fatal)
- Cache invalidation events
- Fallback to hardcoded paths

Log at ERROR level:
- Complete detection failure
- Invalid manual configuration
- Validation failures

## Platform-Specific Implementation Details

### Linux (systemd)

**Service file location**: `/etc/systemd/system/sentinelgo.service`

**Parsing logic**:
```go
func parseSystemdUnitFile(serviceName string) (string, error) {
    unitFile := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
    
    data, err := os.ReadFile(unitFile)
    if err != nil {
        return "", err
    }
    
    // Parse INI-style format
    lines := strings.Split(string(data), "\n")
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if strings.HasPrefix(line, "ExecStart=") {
            execStart := strings.TrimPrefix(line, "ExecStart=")
            // Extract first argument (binary path)
            parts := strings.Fields(execStart)
            if len(parts) > 0 {
                return parts[0], nil
            }
        }
    }
    
    return "", fmt.Errorf("ExecStart not found in unit file")
}
```

### macOS (launchd)

**Plist location**: `/Library/LaunchDaemons/com.sentinelgo.plist`

**Parsing logic**:
```go
func parseLaunchdPlist(serviceName string) (string, error) {
    plistFile := fmt.Sprintf("/Library/LaunchDaemons/com.%s.plist", serviceName)
    
    data, err := os.ReadFile(plistFile)
    if err != nil {
        return "", err
    }
    
    // Parse plist XML
    var plist struct {
        ProgramArguments []string `plist:"ProgramArguments"`
    }
    
    if _, err := plist.Unmarshal(data, &plist); err != nil {
        return "", err
    }
    
    if len(plist.ProgramArguments) > 0 {
        return plist.ProgramArguments[0], nil
    }
    
    return "", fmt.Errorf("ProgramArguments not found in plist")
}
```

### Windows (Windows Service)

**Registry location**: `HKLM\SYSTEM\CurrentControlSet\Services\sentinelgo`

**Query logic**:
```go
func queryWindowsService(serviceName string) (string, error) {
    // Open service manager
    m, err := mgr.Connect()
    if err != nil {
        return "", err
    }
    defer m.Disconnect()
    
    // Open service
    s, err := m.OpenService(serviceName)
    if err != nil {
        return "", err
    }
    defer s.Close()
    
    // Get service config
    config, err := s.Config()
    if err != nil {
        return "", err
    }
    
    // Extract binary path from ImagePath
    // Handle quoted paths: "C:\Program Files\SentinelGo\sentinel.exe" arg1 arg2
    imagePath := config.BinaryPathName
    if strings.HasPrefix(imagePath, "\"") {
        // Quoted path
        endQuote := strings.Index(imagePath[1:], "\"")
        if endQuote > 0 {
            return imagePath[1:endQuote+1], nil
        }
    }
    
    // Unquoted path - take first space-separated token
    parts := strings.Fields(imagePath)
    if len(parts) > 0 {
        return parts[0], nil
    }
    
    return "", fmt.Errorf("failed to extract binary path from ImagePath")
}
```

## Common Installation Paths

### Linux
```go
var linuxCommonPaths = []string{
    "/usr/local/bin/sentinel",
    "/usr/bin/sentinel",
    "/opt/sentinelgo/sentinel",
    filepath.Join(os.Getenv("HOME"), "go/bin/sentinel"),
    filepath.Join(os.Getenv("HOME"), ".local/bin/sentinel"),
}
```

### macOS
```go
var macosCommonPaths = []string{
    "/usr/local/bin/sentinel",
    "/usr/bin/sentinel",
    "/opt/sentinelgo/sentinel",
    filepath.Join(os.Getenv("HOME"), "go/bin/sentinel"),
    "/Applications/SentinelGo/sentinel",
}
```

### Windows
```go
var windowsCommonPaths = []string{
    filepath.Join(os.Getenv("ProgramFiles"), "SentinelGo", "sentinel.exe"),
    filepath.Join(os.Getenv("ProgramFiles(x86)"), "SentinelGo", "sentinel.exe"),
    filepath.Join(os.Getenv("USERPROFILE"), "go", "bin", "sentinel.exe"),
    "C:\\SentinelGo\\sentinel.exe",
}
```

## Conclusion

This design provides a robust, cross-platform solution for dynamically detecting the SentinelGo agent binary path. The multi-strategy approach ensures high reliability across diverse deployment scenarios, while caching ensures performance. The implementation is backward compatible and includes comprehensive error handling and logging for troubleshooting.

Key benefits:
- **Reliability**: Multiple detection strategies with fallbacks
- **Performance**: Caching reduces overhead to <1ms after first detection
- **Flexibility**: Supports manual override for edge cases
- **Observability**: Comprehensive logging for troubleshooting
- **Maintainability**: Clean separation of concerns with platform-specific implementations
