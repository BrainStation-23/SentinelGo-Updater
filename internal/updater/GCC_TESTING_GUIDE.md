# GCC Auto-Installation Testing Guide

This document provides comprehensive testing instructions for the GCC auto-installation feature on Windows.

## Overview

The GCC auto-installation feature ensures that the C compiler (GCC) is available before attempting CGO compilation on Windows systems. This guide covers both automated tests and manual testing procedures.

## Automated Tests

### Test File Location
- `internal/updater/gcc_windows_test.go`

### Running Tests on Windows

```powershell
# Run all GCC-related tests
go test -v ./internal/updater -run TestGCC

# Run specific test
go test -v ./internal/updater -run TestCheckGCCInPath

# Run with verbose logging
go test -v ./internal/updater -run TestEnsureGCCAvailable
```

### Test Coverage

The automated test suite covers:

1. **TestCheckGCCInPath** - Verifies GCC detection in PATH
2. **TestCheckGCCInCommonLocations** - Tests GCC detection in standard installation directories
3. **TestVerifyWingetAvailable** - Validates winget availability check
4. **TestDetectGCCInstallPath** - Tests GCC installation path detection
5. **TestUpdatePATHEnvironment** - Verifies PATH environment variable updates
6. **TestEnsureGCCAvailable** - Integration test for the main orchestration function
7. **TestGCCInstallationScenarios** - Tests various installation scenarios
8. **TestGCCVersionDetection** - Verifies GCC version detection
9. **TestPATHUpdatePersistence** - Tests PATH updates persist to child processes

## Manual Testing Procedures

### Prerequisites

- Windows 10 or Windows 11
- PowerShell or Command Prompt with Administrator privileges
- Internet connection (for winget installation tests)

### Test Scenario 1: GCC Already Installed

**Objective**: Verify the updater skips installation when GCC is already available

**Steps**:
1. Ensure GCC is installed and in PATH:
   ```powershell
   gcc --version
   ```
2. Run the updater or trigger an update
3. Check the logs for:
   ```
   [INFO] GCC found in PATH: <path>
   [INFO] GCC is already available in PATH
   [INFO] Skipping installation - GCC is ready for use
   ```

**Expected Result**:
- No installation attempt
- Compilation proceeds immediately
- Update completes successfully

**Requirements Verified**: 6.1, 6.2, 6.3

---

### Test Scenario 2: GCC Not in PATH but in Common Location

**Objective**: Verify the updater finds GCC in common locations and adds it to PATH

**Setup**:
1. Install GCC manually (e.g., WinLibs to `C:\Program Files\WinLibs`)
2. Remove GCC from system PATH:
   ```powershell
   $env:PATH = $env:PATH -replace 'C:\\Program Files\\WinLibs\\mingw64\\bin;', ''
   ```

**Steps**:
1. Run the updater or trigger an update
2. Check the logs for:
   ```
   [INFO] GCC not found in PATH
   [INFO] Searching common installation directories...
   [INFO] GCC found at: C:\Program Files\WinLibs\mingw64\bin
   [INFO] Adding GCC to PATH for current process
   ```

**Expected Result**:
- GCC found in common location
- PATH updated for current process
- Compilation proceeds successfully
- No installation attempt

**Requirements Verified**: 2.1, 2.2, 5.2, 5.3

---

### Test Scenario 3: GCC Not Installed - Automatic Installation

**Objective**: Verify automatic GCC installation via winget

**Setup**:
1. Ensure GCC is NOT installed:
   ```powershell
   gcc --version  # Should fail
   winget list | findstr WinLibs  # Should return nothing
   ```
2. Ensure winget is available:
   ```powershell
   winget --version
   ```

**Steps**:
1. Run the updater or trigger an update
2. Monitor the logs for the installation process:
   ```
   [INFO] GCC not found in PATH or common locations
   [INFO] Automatic GCC installation will be attempted
   [INFO] Verifying winget is available...
   [INFO] winget version: <version>
   [INFO] Executing: winget install BrechtSanders.WinLibs.POSIX.UCRT --silent --accept-source-agreements --accept-package-agreements
   [INFO] GCC installation in progress...
   ```
3. Wait for installation to complete (may take 2-10 minutes)
4. Verify installation success:
   ```
   [INFO] GCC installation completed successfully
   [INFO] GCC installed at: C:\Program Files\WinLibs\mingw64\bin
   [INFO] GCC version: gcc (GCC) <version>
   ```

**Expected Result**:
- Winget verification succeeds
- GCC installation completes within 10 minutes
- PATH updated with GCC location
- Compilation proceeds successfully
- Update completes successfully

**Requirements Verified**: 1.1, 1.2, 1.3, 1.4, 3.1, 5.1, 5.2, 5.3, 8.1, 8.2, 9.1, 9.2, 10.1, 10.2, 10.3, 10.4

---

### Test Scenario 4: Winget Not Available

**Objective**: Verify clear error messages when winget is not available

**Setup**:
1. Ensure GCC is NOT installed
2. Temporarily rename or remove winget (or test on system without winget)

**Steps**:
1. Run the updater or trigger an update
2. Check the logs for clear error messages:
   ```
   [ERROR] winget is not available on this system
   [ERROR] GCC installation requires winget (Windows Package Manager)
   [ERROR] INSTALLATION INSTRUCTIONS:
   [ERROR]   1. Install winget from: https://aka.ms/getwinget
   [ERROR]   2. Or install 'App Installer' from Microsoft Store
   ```

**Expected Result**:
- Update fails with clear error message
- Installation instructions provided
- Manual installation steps included
- Rollback triggered successfully

**Requirements Verified**: 3.1, 3.2, 3.3, 3.4, 7.1, 7.2, 7.3, 7.4

---

### Test Scenario 5: GCC Installation Timeout

**Objective**: Verify timeout handling for slow installations

**Setup**:
1. Simulate slow network (optional: use network throttling tools)
2. Ensure GCC is NOT installed

**Steps**:
1. Run the updater or trigger an update
2. Monitor installation progress
3. If installation takes longer than 10 minutes, verify timeout:
   ```
   [ERROR] GCC installation timed out after 10 minutes
   [ERROR] TIMEOUT CAUSES:
   [ERROR]   - Slow internet connection
   [ERROR]   - Network interruptions
   ```

**Expected Result**:
- Installation terminates after 10 minutes
- Clear timeout error message
- Recovery instructions provided
- Rollback triggered successfully

**Requirements Verified**: 9.1, 9.2, 9.3, 9.4

---

### Test Scenario 6: PATH Update Verification

**Objective**: Verify PATH is correctly updated and persists

**Setup**:
1. Install GCC in a non-standard location
2. Remove from PATH

**Steps**:
1. Run the updater
2. After GCC is found/installed, verify PATH:
   ```powershell
   echo $env:PATH
   ```
3. Verify GCC is accessible:
   ```powershell
   gcc --version
   ```
4. Start a new PowerShell session and verify GCC is still accessible

**Expected Result**:
- PATH contains GCC bin directory
- GCC is accessible from command line
- No duplicate PATH entries
- Child processes inherit updated PATH

**Requirements Verified**: 5.1, 5.2, 5.3, 5.4

---

### Test Scenario 7: Compilation Success After Installation

**Objective**: Verify compilation succeeds after GCC installation

**Setup**:
1. Ensure GCC is NOT installed initially
2. Ensure winget is available

**Steps**:
1. Trigger an update that requires compilation
2. Monitor the full update process:
   - GCC detection
   - GCC installation
   - PATH update
   - Compilation
3. Verify compilation succeeds:
   ```
   [INFO] GCC is available and ready for compilation
   [INFO] Compilation successful, binary at: <path>
   ```

**Expected Result**:
- GCC installed successfully
- Compilation completes without errors
- New binary is functional
- Service starts successfully

**Requirements Verified**: 1.1, 1.2, 1.3, 1.4, 2.4

---

### Test Scenario 8: Logging Verification

**Objective**: Verify comprehensive logging throughout the process

**Steps**:
1. Run any update scenario
2. Review the updater log file
3. Verify the following are logged:
   - GCC detection attempts
   - Winget version
   - Installation command
   - Installation output
   - GCC version after installation
   - PATH updates
   - Any errors with recovery instructions

**Expected Result**:
- All steps are logged with INFO level
- Errors include detailed recovery instructions
- GCC version is logged
- Installation output is captured

**Requirements Verified**: 4.1, 4.2, 4.3, 4.4, 7.1, 7.2, 8.3, 8.4

---

### Test Scenario 9: Rollback on Installation Failure

**Objective**: Verify rollback occurs when GCC installation fails

**Setup**:
1. Simulate installation failure (e.g., disconnect network during installation)

**Steps**:
1. Trigger an update
2. Cause GCC installation to fail
3. Verify rollback occurs:
   ```
   [ERROR] Update failed: GCC_INSTALLATION_FAILED
   [INFO] Triggering rollback to previous version...
   [INFO] Rollback successful, restored version <version>
   ```

**Expected Result**:
- Update fails gracefully
- Rollback restores previous binary
- Service is restarted with old version
- Backup file is preserved
- Clear recovery instructions provided

**Requirements Verified**: 7.3, 7.4, 9.4

---

## Test Execution Checklist

Use this checklist to track manual testing progress:

- [ ] Test Scenario 1: GCC Already Installed
- [ ] Test Scenario 2: GCC in Common Location
- [ ] Test Scenario 3: Automatic Installation
- [ ] Test Scenario 4: Winget Not Available
- [ ] Test Scenario 5: Installation Timeout
- [ ] Test Scenario 6: PATH Update Verification
- [ ] Test Scenario 7: Compilation Success
- [ ] Test Scenario 8: Logging Verification
- [ ] Test Scenario 9: Rollback on Failure

## Requirements Coverage Matrix

| Requirement | Test Scenarios | Status |
|-------------|----------------|--------|
| 1.1 | 3, 7 | ✓ |
| 1.2 | 3, 7 | ✓ |
| 1.3 | 3, 7 | ✓ |
| 1.4 | 3, 7 | ✓ |
| 2.1 | 2, 3 | ✓ |
| 2.2 | 2, 3 | ✓ |
| 2.3 | 3 | ✓ |
| 2.4 | 7 | ✓ |
| 3.1 | 3, 4 | ✓ |
| 3.2 | 4 | ✓ |
| 3.3 | 4 | ✓ |
| 3.4 | 4 | ✓ |
| 4.1 | 8 | ✓ |
| 4.2 | 8 | ✓ |
| 4.3 | 8 | ✓ |
| 4.4 | 8 | ✓ |
| 5.1 | 3, 6 | ✓ |
| 5.2 | 2, 3, 6 | ✓ |
| 5.3 | 2, 3, 6 | ✓ |
| 5.4 | 6 | ✓ |
| 6.1 | 1 | ✓ |
| 6.2 | 1 | ✓ |
| 6.3 | 1 | ✓ |
| 6.4 | 1 | ✓ |
| 7.1 | 4, 8 | ✓ |
| 7.2 | 4, 8 | ✓ |
| 7.3 | 4, 9 | ✓ |
| 7.4 | 4, 9 | ✓ |
| 8.1 | 3 | ✓ |
| 8.2 | 3 | ✓ |
| 8.3 | 8 | ✓ |
| 8.4 | 8 | ✓ |
| 9.1 | 3, 5 | ✓ |
| 9.2 | 3, 5 | ✓ |
| 9.3 | 5 | ✓ |
| 9.4 | 5, 9 | ✓ |
| 10.1 | 3 | ✓ |
| 10.2 | 3 | ✓ |
| 10.3 | 3 | ✓ |
| 10.4 | 3 | ✓ |

## Troubleshooting

### Tests Won't Run on Non-Windows Systems

The test file uses `// +build windows` constraint and will only compile on Windows. To verify syntax on other platforms:

```bash
GOOS=windows go build -o /dev/null ./internal/updater/gcc_windows_test.go
```

### GCC Already Installed

Some tests require GCC to NOT be installed. To temporarily remove GCC from PATH:

```powershell
$env:PATH = $env:PATH -replace 'C:\\Program Files\\WinLibs\\mingw64\\bin;', ''
```

### Winget Not Available

Install winget from:
- https://aka.ms/getwinget
- Or install "App Installer" from Microsoft Store

## Notes

- All tests are designed to be non-destructive
- Tests will skip scenarios that cannot be set up (e.g., testing "GCC not installed" when GCC is installed)
- Manual testing is required for full coverage due to the need for actual Windows environment and winget
- Automated tests provide validation of logic and error handling
