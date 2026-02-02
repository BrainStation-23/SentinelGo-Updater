# Requirements Document

## Introduction

This document outlines the requirements for fixing a critical bug in the SentinelGo-Updater where the rollback process uses a different binary path than the one detected during the update, causing rollback failures on Windows. The logs show the updater detects the binary at `C:\WINDOWS\system32\config\systemprofile\go\bin\sentinel.exe` but attempts to restore it to `C:\Program Files\SentinelGo\sentinel.exe`, resulting in a "path not found" error and leaving the system in an unrecoverable state.

## Glossary

- **SentinelGo-Updater**: The standalone service responsible for updating the SentinelGo binary
- **Binary Path**: The absolute file system path where the main agent binary is located
- **Backup File**: A copy of the current binary created before an update, stored with a `.backup` extension
- **Rollback Process**: The recovery mechanism that restores the previous binary version when an update fails
- **Update Context**: The state information maintained throughout an update operation, including detected paths
- **Restore Path**: The target location where the backup file should be restored during rollback

## Requirements

### Requirement 1

**User Story:** As a system administrator, I want the rollback process to restore the binary to the same path where it was originally detected, so that rollback succeeds and the system can recover from failed updates

#### Acceptance Criteria

1. WHEN the update process detects the binary path, THE SentinelGo-Updater SHALL store this path in the update context
2. WHEN the rollback process executes, THE SentinelGo-Updater SHALL use the stored binary path from the update context as the restore target
3. THE SentinelGo-Updater SHALL NOT use hardcoded or default paths during rollback
4. THE restore path SHALL be identical to the path where the backup was created

### Requirement 2

**User Story:** As a developer, I want the update context to maintain all critical path information throughout the update lifecycle, so that all update phases use consistent paths

#### Acceptance Criteria

1. THE SentinelGo-Updater SHALL create an update context structure at the start of each update operation
2. THE update context SHALL store the detected binary path, backup file path, and binary directory
3. THE update context SHALL be passed to all update phases (backup, cleanup, download, install, rollback)
4. THE update context SHALL remain valid throughout the entire update operation until completion or rollback

### Requirement 3

**User Story:** As a system administrator, I want the backup file path to be derived from the detected binary path, so that backup and restore operations are always consistent

#### Acceptance Criteria

1. WHEN creating a backup, THE SentinelGo-Updater SHALL construct the backup file path by appending `.backup` to the detected binary path
2. WHEN restoring from backup, THE SentinelGo-Updater SHALL restore to the exact path stored in the update context
3. THE backup file location SHALL be in the same directory as the original binary
4. THE SentinelGo-Updater SHALL log both the backup source path and backup destination path during backup creation

### Requirement 4

**User Story:** As a developer, I want clear logging of all paths used during update and rollback, so that I can diagnose path inconsistency issues

#### Acceptance Criteria

1. THE SentinelGo-Updater SHALL log the detected binary path at the start of the update process
2. THE SentinelGo-Updater SHALL log the backup file path when creating the backup
3. THE SentinelGo-Updater SHALL log the restore target path at the start of rollback
4. IF the restore target path differs from the backup source path, THEN THE SentinelGo-Updater SHALL log a warning before attempting rollback

### Requirement 5

**User Story:** As a system administrator, I want the rollback process to verify path consistency before attempting restoration, so that I receive clear error messages if paths are inconsistent

#### Acceptance Criteria

1. WHEN rollback begins, THE SentinelGo-Updater SHALL verify that the backup file exists at the expected location
2. WHEN rollback begins, THE SentinelGo-Updater SHALL verify that the restore target directory exists
3. IF the restore target directory does not exist, THEN THE SentinelGo-Updater SHALL create it before attempting restoration
4. IF path verification fails, THEN THE SentinelGo-Updater SHALL log detailed error information including both paths

### Requirement 6

**User Story:** As a system administrator, I want the updater to handle Windows-specific path scenarios correctly, so that rollback works reliably on Windows systems

#### Acceptance Criteria

1. WHEN running on Windows, THE SentinelGo-Updater SHALL correctly handle paths with drive letters (e.g., `C:\`)
2. WHEN running on Windows, THE SentinelGo-Updater SHALL correctly handle paths in system profile directories
3. WHEN running on Windows, THE SentinelGo-Updater SHALL correctly handle paths in Program Files directories
4. THE SentinelGo-Updater SHALL use Windows-appropriate path separators and path manipulation functions

### Requirement 7

**User Story:** As a developer, I want the update process to fail fast if path detection is inconsistent, so that we don't proceed with an update that cannot be rolled back

#### Acceptance Criteria

1. WHEN the update process begins, THE SentinelGo-Updater SHALL validate that the detected binary path is accessible
2. WHEN creating a backup, THE SentinelGo-Updater SHALL validate that the backup directory is writable
3. IF path validation fails before cleanup, THEN THE SentinelGo-Updater SHALL abort the update without modifying any files
4. THE SentinelGo-Updater SHALL log validation failures with specific details about which path check failed

### Requirement 8

**User Story:** As a system administrator, I want the rollback process to provide actionable recovery instructions when restoration fails, so that I can manually recover the system

#### Acceptance Criteria

1. IF rollback fails due to path issues, THEN THE SentinelGo-Updater SHALL log the backup file location
2. IF rollback fails due to path issues, THEN THE SentinelGo-Updater SHALL log the intended restore target location
3. THE SentinelGo-Updater SHALL provide step-by-step manual recovery instructions in the error log
4. THE recovery instructions SHALL include commands to verify backup file integrity and manually restore the binary
