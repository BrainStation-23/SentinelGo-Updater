# Implementation Plan

- [x] 1. Update GetDataDirectory() function to use macOS-specific path
  - Modify the switch statement in `internal/paths/paths.go` to separate `darwin` from `linux` case
  - Set macOS to return `/Library/Application Support/SentinelGo`
  - Ensure Linux continues to return `/var/lib/sentinelgo`
  - Update the function documentation comment to reflect the new platform-specific paths
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 3.1, 3.2_

- [x] 2. Verify derived path functions work correctly
  - Run the application or write a simple test program to print all paths on macOS
  - Confirm `GetDatabasePath()` returns `/Library/Application Support/SentinelGo/sentinel.db`
  - Confirm `GetUpdaterLogPath()` returns `/Library/Application Support/SentinelGo/updater.log`
  - Confirm `GetAgentLogPath()` returns `/Library/Application Support/SentinelGo/agent.log`
  - _Requirements: 4.1, 4.2, 4.3_

- [x] 3. Test directory creation with proper permissions
  - Test `EnsureDataDirectory()` on macOS with root permissions
  - Verify the directory is created at `/Library/Application Support/SentinelGo`
  - Verify directory permissions are set to 0755
  - Test error handling when run without sufficient permissions
  - _Requirements: 2.1, 2.2, 2.3_
