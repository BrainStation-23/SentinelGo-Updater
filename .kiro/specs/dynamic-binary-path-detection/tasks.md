# Implementation Plan

- [x] 1. Create core binary detector framework
  - Create `internal/paths/detector.go` with BinaryDetector struct and core methods
  - Implement thread-safe caching mechanism using sync.RWMutex
  - Implement binary path validation logic (file exists, is executable)
  - Implement cache invalidation and refresh logic
  - _Requirements: 1.1, 2.5, 8.1, 8.2, 8.3_

- [x] 2. Implement configuration file support
  - Define UpdaterConfig struct for JSON configuration
  - Implement loadConfigPath() to read from `/var/lib/sentinelgo/updater-config.json`
  - Add validation for manually configured paths
  - Implement fallback to auto-detection when manual config is invalid
  - _Requirements: 7.1, 7.2, 7.3, 7.4_

- [x] 3. Implement Linux systemd service parser
  - Create `internal/paths/detector_linux.go` with build tag `//go:build linux`
  - Implement parseSystemdUnitFile() to read `/etc/systemd/system/sentinelgo.service`
  - Parse ExecStart directive and extract binary path
  - Handle both absolute and relative paths, quoted and unquoted
  - _Requirements: 2.1, 3.1, 3.2, 3.4_

- [x] 4. Implement macOS launchd plist parser
  - Create `internal/paths/detector_darwin.go` with build tag `//go:build darwin`
  - Implement parseLaunchdPlist() to read launchd plist file
  - Parse ProgramArguments array and extract first element
  - Handle plist XML parsing using encoding/xml or plist library
  - _Requirements: 2.1, 4.1, 4.2, 4.4_

- [x] 5. Implement Windows service query
  - Create `internal/paths/detector_windows.go` with build tag `//go:build windows`
  - Implement queryWindowsService() using golang.org/x/sys/windows/svc/mgr
  - Query service configuration and extract ImagePath
  - Handle quoted paths, UNC paths, and paths with arguments
  - _Requirements: 2.1, 5.1, 5.2, 5.4_

- [x] 6. Implement running process detection
  - Add detectFromRunningProcess() method to BinaryDetector
  - Implement platform-specific process enumeration (Linux: /proc, macOS: ps, Windows: API)
  - Extract binary path from running sentinel process
  - Handle cases where process is not running
  - _Requirements: 2.4_

- [x] 7. Implement PATH environment variable search
  - Add detectFromPATH() method to BinaryDetector
  - Implement searchPATH() utility function
  - Split PATH by platform-specific separator (: on Unix, ; on Windows)
  - Search each directory for sentinel binary
  - _Requirements: 2.2_

- [x] 8. Implement common paths fallback search
  - Add detectFromCommonPaths() method to BinaryDetector
  - Define platform-specific common paths arrays (Linux, macOS, Windows)
  - Search each common path and validate binary existence
  - Handle permission errors gracefully
  - _Requirements: 2.3_

- [x] 9. Implement main detection orchestration
  - Implement DetectBinaryPath() method with strategy chain
  - Call detection methods in priority order: service config → running process → PATH → common paths
  - Log each attempt with success/failure details
  - Return first successful detection result
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_

- [x] 10. Add comprehensive error handling and logging
  - Implement detailed error messages for each detection method failure
  - Log all attempted methods and their results
  - Provide actionable troubleshooting steps in error messages
  - Ensure updater continues retrying on subsequent checks
  - _Requirements: 6.1, 6.2, 6.3, 6.4_

- [x] 11. Integrate detector into paths package
  - Modify `internal/paths/paths.go` to use BinaryDetector
  - Update GetMainAgentBinaryPath() to call detector.DetectBinaryPath()
  - Implement singleton pattern with sync.Once for detector initialization
  - Keep getFallbackBinaryPath() for backward compatibility
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [x] 12. Update updater.go to handle detection errors
  - Modify getInstalledVersion() to handle detection failures gracefully
  - Add retry logic for transient detection failures
  - Log detection method used for successful detections
  - Ensure updater doesn't crash on detection failure
  - _Requirements: 6.4, 8.4_

- [ ]* 13. Write unit tests for core detector
  - Test caching mechanism (hit/miss, invalidation)
  - Test path validation logic
  - Test configuration file loading
  - Test concurrent access to detector
  - _Requirements: 8.1, 8.2, 8.3_

- [ ]* 14. Write unit tests for platform-specific parsers
  - Test systemd unit file parsing with various formats
  - Test launchd plist parsing
  - Test Windows service query
  - Test handling of malformed config files
  - _Requirements: 3.1, 3.2, 3.4, 4.1, 4.2, 5.1, 5.2_

- [ ]* 15. Write unit tests for fallback strategies
  - Test PATH search with multiple directories
  - Test common paths search
  - Test running process detection
  - Test handling of permission errors
  - _Requirements: 2.2, 2.3, 2.4_

- [ ]* 16. Write integration tests
  - Test end-to-end detection with binary in various locations
  - Test fallback chain when methods fail
  - Test with running service on each platform
  - Test complete failure scenario
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 2.1, 2.2, 2.3, 2.4_

- [ ] 17. Test on Linux with systemd
  - Deploy updater on Linux test system
  - Test with binary in ~/go/bin
  - Test with binary in /usr/local/bin
  - Test with binary in custom location
  - Verify service config parsing works correctly
  - _Requirements: 3.1, 3.2, 3.3, 3.4_

- [ ] 18. Test on macOS with launchd
  - Deploy updater on macOS test system
  - Test with binary in various locations
  - Verify launchd plist parsing works correctly
  - Test fallback strategies
  - _Requirements: 4.1, 4.2, 4.3, 4.4_

- [ ] 19. Test on Windows with Windows Service
  - Deploy updater on Windows test system
  - Test with binary in Program Files
  - Test with binary in user directory
  - Verify Windows service query works correctly
  - _Requirements: 5.1, 5.2, 5.3, 5.4_

- [ ] 20. Verify performance requirements
  - Measure detection time for each method
  - Verify total detection time is under 5 seconds
  - Verify cached path validation is under 1ms
  - Test with 100+ concurrent detection calls
  - _Requirements: 8.1, 8.2, 8.4_
