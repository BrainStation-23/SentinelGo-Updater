# Requirements Document

## Introduction

This document outlines the requirements for implementing dynamic binary path detection in the SentinelGo-Updater service. The updater currently fails to locate the main agent binary because it uses a hardcoded path (`/usr/local/bin/sentinel`), which doesn't work across different installation locations and operating systems. The solution must work reliably across 800+ computers running Linux, Windows, and macOS with varying installation paths.

## Glossary

- **SentinelGo-Updater**: The standalone service responsible for updating the SentinelGo agent
- **Main Agent Binary**: The executable file for the SentinelGo agent (named `sentinel` or `sentinel.exe`)
- **Binary Path**: The absolute file system path where the main agent binary is located
- **Installation Path**: The directory where the SentinelGo agent is installed
- **System Service**: A background process managed by the operating system (systemd on Linux, launchd on macOS, Windows Service on Windows)
- **PATH Environment Variable**: The system environment variable containing directories to search for executables
- **Service Configuration**: The configuration file that defines how a system service runs (e.g., systemd unit file)

## Requirements

### Requirement 1

**User Story:** As a system administrator, I want the updater to automatically find the sentinel binary regardless of installation location, so that it works across different deployment scenarios

#### Acceptance Criteria

1. WHEN the SentinelGo-Updater starts, THE SentinelGo-Updater SHALL detect the main agent binary path without requiring manual configuration
2. THE SentinelGo-Updater SHALL support detection of binaries installed in standard system paths (e.g., `/usr/local/bin`, `/usr/bin`, `C:\Program Files`)
3. THE SentinelGo-Updater SHALL support detection of binaries installed in user-specific paths (e.g., `~/go/bin`, `%USERPROFILE%\go\bin`)
4. THE SentinelGo-Updater SHALL support detection of binaries installed in custom paths specified during installation

### Requirement 2

**User Story:** As a developer, I want the updater to use multiple detection strategies, so that it can reliably find the binary across different operating systems and configurations

#### Acceptance Criteria

1. THE SentinelGo-Updater SHALL attempt to detect the binary path by querying the running service configuration
2. IF the service configuration query fails, THEN THE SentinelGo-Updater SHALL search the PATH environment variable for the binary
3. IF the PATH search fails, THEN THE SentinelGo-Updater SHALL search common installation directories for the binary
4. IF the running process can be queried, THEN THE SentinelGo-Updater SHALL detect the binary path from the running process information
5. THE SentinelGo-Updater SHALL use the first successfully detected path and cache it for subsequent operations

### Requirement 3

**User Story:** As a system administrator, I want the updater to work correctly on Linux systems, so that updates can be deployed across our Linux fleet

#### Acceptance Criteria

1. WHEN running on Linux, THE SentinelGo-Updater SHALL query the systemd service unit file to extract the binary path
2. WHEN running on Linux, THE SentinelGo-Updater SHALL parse the `ExecStart` directive from the service configuration
3. IF systemd is not available, THEN THE SentinelGo-Updater SHALL fall back to alternative detection methods
4. THE SentinelGo-Updater SHALL handle both absolute and relative paths in systemd unit files

### Requirement 4

**User Story:** As a system administrator, I want the updater to work correctly on macOS systems, so that updates can be deployed across our Mac fleet

#### Acceptance Criteria

1. WHEN running on macOS, THE SentinelGo-Updater SHALL query the launchd plist file to extract the binary path
2. WHEN running on macOS, THE SentinelGo-Updater SHALL parse the `ProgramArguments` array from the plist configuration
3. IF launchd configuration is not accessible, THEN THE SentinelGo-Updater SHALL fall back to alternative detection methods
4. THE SentinelGo-Updater SHALL handle both absolute paths and paths with environment variable expansion

### Requirement 5

**User Story:** As a system administrator, I want the updater to work correctly on Windows systems, so that updates can be deployed across our Windows fleet

#### Acceptance Criteria

1. WHEN running on Windows, THE SentinelGo-Updater SHALL query the Windows Service configuration to extract the binary path
2. WHEN running on Windows, THE SentinelGo-Updater SHALL use Windows API calls or registry queries to retrieve service executable path
3. IF Windows Service query fails, THEN THE SentinelGo-Updater SHALL fall back to alternative detection methods
4. THE SentinelGo-Updater SHALL handle Windows path formats including drive letters and UNC paths

### Requirement 6

**User Story:** As a developer, I want the updater to provide clear error messages when binary detection fails, so that I can troubleshoot deployment issues

#### Acceptance Criteria

1. IF all detection methods fail, THEN THE SentinelGo-Updater SHALL log detailed error messages for each attempted method
2. THE SentinelGo-Updater SHALL log the directories searched and the detection strategies attempted
3. THE SentinelGo-Updater SHALL provide actionable error messages that guide users toward resolution
4. THE SentinelGo-Updater SHALL continue retrying detection on subsequent update checks rather than failing permanently

### Requirement 7

**User Story:** As a system administrator, I want the option to manually specify the binary path, so that I can override automatic detection if needed

#### Acceptance Criteria

1. THE SentinelGo-Updater SHALL support a configuration option to manually specify the main agent binary path
2. WHEN a manual path is configured, THE SentinelGo-Updater SHALL use the configured path without attempting automatic detection
3. IF the manually configured path is invalid, THEN THE SentinelGo-Updater SHALL log an error and fall back to automatic detection
4. THE SentinelGo-Updater SHALL validate that the manually configured path points to a valid executable file

### Requirement 8

**User Story:** As a developer, I want the binary path detection to be performant, so that it doesn't slow down the updater service

#### Acceptance Criteria

1. THE SentinelGo-Updater SHALL cache the detected binary path after the first successful detection
2. THE SentinelGo-Updater SHALL reuse the cached path for subsequent update checks without re-detecting
3. IF an update operation fails due to an invalid cached path, THEN THE SentinelGo-Updater SHALL invalidate the cache and re-detect the path
4. THE binary path detection process SHALL complete within 5 seconds on all supported platforms
