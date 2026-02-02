# GCC Auto-Installation Test Suite

This directory contains comprehensive tests for the GCC auto-installation feature on Windows.

## Files

- **gcc_windows_test.go** - Automated test suite (Windows only)
- **GCC_TESTING_GUIDE.md** - Comprehensive manual testing guide
- **run_gcc_tests.ps1** - PowerShell script to run all tests
- **TEST_README.md** - This file

## Quick Start

### Running Automated Tests (Windows Only)

#### Option 1: Using PowerShell Script (Recommended)
```powershell
.\internal\updater\run_gcc_tests.ps1
```

#### Option 2: Using Go Test Command
```powershell
# Run all GCC tests
go test -v ./internal/updater -run TestGCC

# Run specific test
go test -v ./internal/updater -run TestCheckGCCInPath

# Run with timeout
go test -v -timeout 15m ./internal/updater -run TestEnsureGCCAvailable
```

### Manual Testing

For comprehensive manual testing procedures, see [GCC_TESTING_GUIDE.md](./GCC_TESTING_GUIDE.md)

## Test Coverage

The test suite covers all requirements from the specification:

### Automated Tests

1. **TestCheckGCCInPath** - Tests GCC detection in system PATH
2. **TestCheckGCCInCommonLocations** - Tests GCC detection in standard directories
3. **TestVerifyWingetAvailable** - Tests winget availability verification
4. **TestDetectGCCInstallPath** - Tests GCC installation path detection
5. **TestUpdatePATHEnvironment** - Tests PATH environment variable updates
6. **TestEnsureGCCAvailable** - Integration test for full GCC availability check
7. **TestGCCInstallationScenarios** - Tests various installation scenarios
8. **TestGCCVersionDetection** - Tests GCC version detection
9. **TestPATHUpdatePersistence** - Tests PATH updates persist to child processes

### Manual Test Scenarios

1. GCC already installed (skip installation)
2. GCC in common location but not in PATH
3. GCC not installed (automatic installation)
4. Winget not available (error handling)
5. Installation timeout (timeout handling)
6. PATH update verification
7. Compilation success after installation
8. Logging verification
9. Rollback on installation failure

## Requirements Coverage

All 40 acceptance criteria from the requirements document are covered:

- Requirements 1.1-1.4: Automatic GCC installation ✓
- Requirements 2.1-2.4: GCC availability check ✓
- Requirements 3.1-3.4: Winget verification ✓
- Requirements 4.1-4.4: Logging ✓
- Requirements 5.1-5.4: PATH environment update ✓
- Requirements 6.1-6.4: Idempotent installation ✓
- Requirements 7.1-7.4: Error handling ✓
- Requirements 8.1-8.4: Specific GCC version ✓
- Requirements 9.1-9.4: Installation timeout ✓
- Requirements 10.1-10.4: Non-interactive installation ✓

## Platform Requirements

- **Operating System**: Windows 10 or Windows 11
- **Go Version**: 1.25.6 or later
- **PowerShell**: 5.1 or later (for test script)
- **Winget**: Required for installation tests (optional for detection tests)

## Test Execution Notes

### Automated Tests

- Tests use `// +build windows` constraint and only run on Windows
- Some tests will skip if prerequisites are not met (e.g., GCC not installed)
- Integration tests may install GCC if not present and winget is available
- Tests are designed to be non-destructive

### Manual Tests

- Require actual Windows environment
- Some scenarios require GCC to be uninstalled
- Installation tests require internet connection
- Full test cycle takes approximately 30-60 minutes

## Verifying Test File Syntax (Non-Windows)

On Linux/macOS, you can verify the test file compiles correctly:

```bash
GOOS=windows go build -o /dev/null ./internal/updater/gcc_windows_test.go
```

## Troubleshooting

### Tests Not Running

If tests don't run, ensure:
1. You're on a Windows system
2. Go is installed and in PATH
3. You're in the project root directory

### Tests Skipping

Tests will skip if:
- GCC is required but not installed (for detection tests)
- GCC is installed but test requires it to be absent
- Winget is not available (for installation tests)

This is expected behavior - tests adapt to the current system state.

### Integration Test Warnings

The `TestEnsureGCCAvailable` integration test may:
- Install GCC if not present (requires winget)
- Take several minutes to complete
- Require internet connection

## CI/CD Integration

To integrate these tests into CI/CD:

```yaml
# Example GitHub Actions workflow
- name: Run GCC Tests
  if: runner.os == 'Windows'
  run: |
    go test -v ./internal/updater -run TestGCC
```

## Contributing

When adding new GCC-related functionality:

1. Add corresponding test cases to `gcc_windows_test.go`
2. Update manual test scenarios in `GCC_TESTING_GUIDE.md`
3. Update requirements coverage matrix
4. Run full test suite before submitting PR

## Support

For issues or questions:
1. Review the [GCC_TESTING_GUIDE.md](./GCC_TESTING_GUIDE.md)
2. Check test output for specific error messages
3. Verify all prerequisites are met
4. Consult the main project documentation
