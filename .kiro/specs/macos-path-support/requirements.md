# Requirements Document

## Introduction

This feature addresses the cross-platform path handling issue in SentinelGo where macOS currently uses Linux-style paths (`/var/lib/sentinelgo`) instead of macOS-native conventions. The system needs to use platform-appropriate directories for storing application data, logs, and databases while maintaining backward compatibility and proper permission handling.

## Glossary

- **Path Manager**: The system component responsible for determining and providing platform-specific file system paths
- **Data Directory**: The primary directory where the application stores persistent data including databases and logs
- **Application Support Directory**: The macOS-standard location for application data (`/Library/Application Support` or `~/Library/Application Support`)
- **SentinelGo**: The monitoring agent application that requires cross-platform file storage

## Requirements

### Requirement 1

**User Story:** As a macOS user, I want SentinelGo to use standard macOS directories, so that the application follows platform conventions and stores data in the appropriate system location.

#### Acceptance Criteria

1. WHEN the Path Manager detects macOS as the operating system, THE Path Manager SHALL return `/Library/Application Support/SentinelGo` as the data directory path
2. WHEN the Path Manager detects Linux as the operating system, THE Path Manager SHALL return `/var/lib/sentinelgo` as the data directory path
3. WHEN the Path Manager detects Windows as the operating system, THE Path Manager SHALL return `%ProgramData%\SentinelGo` as the data directory path
4. THE Path Manager SHALL NOT use `/var/lib` directory on macOS systems regardless of permission level

### Requirement 2

**User Story:** As a developer, I want the path system to handle directory creation gracefully, so that the application can initialize properly on first run.

#### Acceptance Criteria

1. WHEN the Path Manager attempts to create the data directory, THE Path Manager SHALL create all parent directories in the path if they do not exist
2. IF the Path Manager cannot create the data directory due to permission errors, THEN THE Path Manager SHALL return a descriptive error indicating the permission issue
3. WHEN the Path Manager successfully creates the data directory, THE Path Manager SHALL set permissions to 0755 (rwxr-xr-x)

### Requirement 3

**User Story:** As a system administrator, I want clear separation between macOS and Linux path conventions, so that each platform uses its native directory structure.

#### Acceptance Criteria

1. THE Path Manager SHALL distinguish between darwin (macOS) and linux operating systems in the path selection logic
2. THE Path Manager SHALL NOT use `/var/lib` as the data directory on macOS systems
3. THE Path Manager SHALL maintain the existing binary directory convention of `/usr/local/bin` for both macOS and Linux systems

### Requirement 4

**User Story:** As a developer, I want all derived paths to be based on the correct platform-specific data directory, so that logs and databases are stored in the appropriate location.

#### Acceptance Criteria

1. WHEN the Path Manager provides the database path, THE Path Manager SHALL construct it by joining the platform-specific data directory with `sentinel.db`
2. WHEN the Path Manager provides the updater log path, THE Path Manager SHALL construct it by joining the platform-specific data directory with `updater.log`
3. WHEN the Path Manager provides the agent log path, THE Path Manager SHALL construct it by joining the platform-specific data directory with `agent.log`
