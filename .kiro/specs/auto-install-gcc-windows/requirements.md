# Requirements Document

## Introduction

This document outlines the requirements for automatically installing GCC on Windows systems when it's not available during the update process. Currently, the updater fails to compile the new binary because CGO requires a C compiler (GCC), and the compilation fails with "gcc: executable file not found in %PATH%". The solution should automatically install GCC using winget before attempting compilation.

## Glossary

- **GCC**: GNU Compiler Collection, a C/C++ compiler required for CGO compilation
- **CGO**: Go's mechanism for calling C code, required by SQLite and other C dependencies
- **WinLibs**: A Windows build of GCC and related tools
- **winget**: Windows Package Manager, a command-line tool for installing applications on Windows
- **Compilation Phase**: Step 4 of the update process where the new binary is downloaded and compiled
- **PATH Environment Variable**: System variable containing directories to search for executables

## Requirements

### Requirement 1

**User Story:** As a system administrator, I want the updater to automatically install GCC on Windows if it's missing, so that updates don't fail due to missing compiler dependencies

#### Acceptance Criteria

1. WHEN the updater detects that GCC is not available on Windows, THE SentinelGo-Updater SHALL automatically install GCC using winget
2. THE SentinelGo-Updater SHALL use the WinLibs POSIX UCRT build (BrechtSanders.WinLibs.POSIX.UCRT)
3. AFTER installing GCC, THE SentinelGo-Updater SHALL verify that GCC is accessible in the PATH
4. IF GCC installation succeeds, THEN THE SentinelGo-Updater SHALL proceed with compilation

### Requirement 2

**User Story:** As a developer, I want the updater to check for GCC availability before attempting compilation, so that we can install it proactively rather than failing during compilation

#### Acceptance Criteria

1. WHEN the compilation phase begins, THE SentinelGo-Updater SHALL check if GCC is available in the PATH
2. IF GCC is not found in PATH, THEN THE SentinelGo-Updater SHALL search common GCC installation directories
3. IF GCC is not found in common directories, THEN THE SentinelGo-Updater SHALL trigger automatic installation
4. THE GCC availability check SHALL complete before any compilation attempts

### Requirement 3

**User Story:** As a system administrator, I want the updater to verify that winget is available before attempting GCC installation, so that I receive clear error messages if winget is not installed

#### Acceptance Criteria

1. WHEN automatic GCC installation is triggered, THE SentinelGo-Updater SHALL verify that winget is available
2. IF winget is not available, THEN THE SentinelGo-Updater SHALL log a clear error message with installation instructions
3. THE SentinelGo-Updater SHALL provide a link to winget installation documentation
4. IF winget is not available, THEN THE SentinelGo-Updater SHALL fail the update with actionable recovery instructions

### Requirement 4

**User Story:** As a system administrator, I want the GCC installation process to be logged in detail, so that I can troubleshoot installation failures

#### Acceptance Criteria

1. THE SentinelGo-Updater SHALL log when GCC is not found and automatic installation is triggered
2. THE SentinelGo-Updater SHALL log the winget command being executed
3. THE SentinelGo-Updater SHALL log the output from the winget installation process
4. THE SentinelGo-Updater SHALL log when GCC installation completes successfully or fails

### Requirement 5

**User Story:** As a developer, I want the updater to update the PATH environment variable after installing GCC, so that the compiler is immediately available for compilation

#### Acceptance Criteria

1. AFTER installing GCC via winget, THE SentinelGo-Updater SHALL detect the GCC installation directory
2. THE SentinelGo-Updater SHALL add the GCC bin directory to the PATH environment variable for the current process
3. THE SentinelGo-Updater SHALL verify that GCC is accessible after updating PATH
4. IF GCC is still not accessible after PATH update, THEN THE SentinelGo-Updater SHALL log an error with manual recovery instructions

### Requirement 6

**User Story:** As a system administrator, I want the GCC installation to be idempotent, so that the updater doesn't attempt to reinstall GCC if it's already installed

#### Acceptance Criteria

1. WHEN checking for GCC availability, THE SentinelGo-Updater SHALL first check if GCC is already in PATH
2. IF GCC is found in PATH, THEN THE SentinelGo-Updater SHALL skip automatic installation
3. THE SentinelGo-Updater SHALL log that GCC is already available and skip installation
4. THE GCC availability check SHALL complete within 2 seconds

### Requirement 7

**User Story:** As a system administrator, I want the updater to handle GCC installation failures gracefully, so that I can manually install GCC and retry the update

#### Acceptance Criteria

1. IF GCC installation fails, THEN THE SentinelGo-Updater SHALL log the failure reason
2. THE SentinelGo-Updater SHALL provide manual installation instructions including the exact winget command
3. THE SentinelGo-Updater SHALL preserve the backup file for rollback
4. THE SentinelGo-Updater SHALL trigger rollback to restore the previous binary version

### Requirement 8

**User Story:** As a developer, I want the GCC installation to use a specific version that's known to work, so that we avoid compatibility issues with different GCC builds

#### Acceptance Criteria

1. THE SentinelGo-Updater SHALL use the WinLibs POSIX UCRT build (BrechtSanders.WinLibs.POSIX.UCRT)
2. THE SentinelGo-Updater SHALL NOT specify a version number, allowing winget to install the latest stable version
3. THE SentinelGo-Updater SHALL log the GCC version after successful installation
4. THE GCC version information SHALL be included in update logs for troubleshooting

### Requirement 9

**User Story:** As a system administrator, I want the GCC installation to complete within a reasonable time, so that updates don't hang indefinitely

#### Acceptance Criteria

1. THE GCC installation process SHALL have a timeout of 10 minutes
2. IF the installation exceeds the timeout, THEN THE SentinelGo-Updater SHALL terminate the installation process
3. THE SentinelGo-Updater SHALL log a timeout error with recovery instructions
4. THE SentinelGo-Updater SHALL trigger rollback after installation timeout

### Requirement 10

**User Story:** As a developer, I want the updater to work on systems where winget requires user interaction, so that updates can proceed in automated environments

#### Acceptance Criteria

1. THE SentinelGo-Updater SHALL use winget with the --accept-source-agreements flag to avoid interactive prompts
2. THE SentinelGo-Updater SHALL use winget with the --accept-package-agreements flag to avoid license prompts
3. THE SentinelGo-Updater SHALL use winget with the --silent flag to minimize output
4. THE winget command SHALL be non-interactive and suitable for automated execution
