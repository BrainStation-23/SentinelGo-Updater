# Requirements Document

## Introduction

This document outlines the requirements for fixing a critical bug in the SentinelGo-Updater where backup files are deleted prematurely during the cleanup phase, preventing rollback when updates fail. This leaves the system in an unrecoverable state with no binary available.

## Glossary

- **SentinelGo-Updater**: The standalone service responsible for updating the SentinelGo binary
- **Backup File**: A copy of the current binary created before an update, used for rollback if the update fails
- **Cleanup Phase**: Step 3 of the update process where old files are removed
- **Rollback Process**: The recovery mechanism that restores the previous binary version when an update fails
- **Update Process**: The multi-step procedure for updating the SentinelGo binary (backup, stop, uninstall, cleanup, download, install, start)

## Requirements

### Requirement 1

**User Story:** As a system administrator, I want the backup file to be preserved during the cleanup phase, so that rollback is possible if the update fails

#### Acceptance Criteria

1. WHEN the cleanup phase executes, THE SentinelGo-Updater SHALL NOT delete the backup file created in the backup phase
2. THE SentinelGo-Updater SHALL delete old backup files (sentinel.old) if they exist from previous updates
3. THE SentinelGo-Updater SHALL preserve the current backup file (sentinel.backup) until the update completes successfully
4. THE cleanup phase SHALL log which files are being preserved and which are being deleted

### Requirement 2

**User Story:** As a system administrator, I want the backup file to be deleted only after a successful update, so that disk space is managed efficiently without compromising rollback capability

#### Acceptance Criteria

1. WHEN the update completes successfully, THE SentinelGo-Updater SHALL delete the backup file
2. IF the update fails at any step, THEN THE SentinelGo-Updater SHALL preserve the backup file for rollback
3. THE SentinelGo-Updater SHALL log when the backup file is deleted after successful update
4. THE backup file deletion SHALL occur after the new binary is verified and the service is restarted successfully

### Requirement 3

**User Story:** As a system administrator, I want the rollback process to succeed when updates fail, so that the system can recover to a working state automatically

#### Acceptance Criteria

1. WHEN an update fails at any step after cleanup, THE SentinelGo-Updater SHALL successfully locate the backup file
2. THE rollback process SHALL restore the backup file to the original binary location
3. THE rollback process SHALL reinstall and restart the service with the restored binary
4. IF the backup file is missing during rollback, THEN THE SentinelGo-Updater SHALL log a critical error with recovery instructions

### Requirement 4

**User Story:** As a developer, I want clear logging throughout the backup lifecycle, so that I can diagnose issues and verify correct behavior

#### Acceptance Criteria

1. THE SentinelGo-Updater SHALL log when a backup file is created with its path and size
2. THE SentinelGo-Updater SHALL log when a backup file is preserved during cleanup
3. THE SentinelGo-Updater SHALL log when a backup file is deleted after successful update
4. THE SentinelGo-Updater SHALL log when a backup file is used during rollback

### Requirement 5

**User Story:** As a system administrator, I want the update process to handle missing environment variables gracefully, so that updates don't fail due to systemd service context limitations

#### Acceptance Criteria

1. WHEN the $HOME environment variable is not defined, THE SentinelGo-Updater SHALL use `os.UserHomeDir()` as a fallback to determine the home directory
2. WHEN `os.UserHomeDir()` fails, THE SentinelGo-Updater SHALL construct the home directory path from the `/etc/passwd` file or equivalent system information
3. THE SentinelGo-Updater SHALL set the $HOME environment variable for child processes if it is not already set
4. THE SentinelGo-Updater SHALL log which method was used to determine the home directory
5. IF the home directory cannot be determined by any method, THEN THE SentinelGo-Updater SHALL fail the update before the cleanup phase with a clear error message
