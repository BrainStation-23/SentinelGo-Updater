# Design Document: Auto-Install GCC on Windows

## Overview

This design addresses the compilation failure on Windows due to missing GCC compiler. The updater currently detects that GCC is missing and logs a warning, but proceeds with compilation anyway, resulting in a failure. The solution will automatically install GCC using winget before attempting compilation, ensuring that CGO-enabled Go code can be compiled successfully.

### Current Behavior (Buggy)

1. **Compilation starts**: Updater checks for GCC
2. **GCC not found**: Logs warning "GCC not found: GCC not found in common locations or PATH"
3. **Compilation attempted**: Proceeds with `go install` anyway
4. **Compilation fails**: "cgo: C compiler "gcc" not found: exec: "gcc": executable file not found in %PATH%"
5. **Rollback triggered**: Update fails and rollback is attempted

### Desired Behavior

1. **Compilation starts**: Updater checks for GCC
2. **GCC not found**: Logs "GCC not found, installing automatically..."
3. **GCC installation**: Executes `winget install BrechtSanders.WinLibs.POSIX.UCRT --silent --accept-source-agreements --accept-package-agreements`
4. **PATH updated**: Adds GCC bin directory to PATH
5. **GCC verified**: Confirms GCC is now accessible
6. **Compilation proceeds**: `go install` succeeds with CGO support

## Architecture

### Component Overview

```
downloadAndCompile()
    ↓
ensureGCCAvailable()
    ├─ checkGCCInPath()
    │   └─ Returns true if found
    ├─ checkGCCInCommonLocations()
    │   └─ Returns path if found
    └─ installGCCWithWinget()
        ├─ verifyWingetAvailable()
        ├─ executeWingetInstall()
        ├─ detectGCCInstallPath()
        └─ updatePATHEnvironment()
    ↓
setupGoEnvironment()
    └─ Includes GCC in PATH
    ↓
executeGoInstall()
```

### New Functions

#### 1. ensureGCCAvailable()

**Location**: `internal/updater/updater.go`

```go
func ensureGCCAvailable() error
```

This is the main orchestration function that:
- Checks if GCC is already available
- Triggers installation if not found
- Verifies GCC is accessible after installation
- Returns error if GCC cannot be made available

#### 2. checkGCCInPath()

**Location**: `internal/updater/updater.go`

```go
func checkGCCInPath() bool
```

Uses `exec.LookPath("gcc")` to check if GCC is in PATH.

#### 3. checkGCCInCommonLocations()

**Location**: `internal/updater/updater.go`

```go
func checkGCCInCommonLocations() (string, error)
```

Searches common GCC installation directories:
- `C:\Program Files\WinLibs\mingw64\bin`
- `C:\Program Files\WinLibs\mingw32\bin`
- `C:\MinGW\bin`
- `C:\MinGW64\bin`
- `C:\TDM-GCC-64\bin`
- `C:\msys64\mingw64\bin`
- `C:\msys64\ucrt64\bin`

Returns the path if found, error otherwise.

#### 4. installGCCWithWinget()

**Location**: `internal/updater/updater.go`

```go
func installGCCWithWinget() error
```

Orchestrates the GCC installation:
1. Verify winget is available
2. Execute winget install command
3. Detect installation path
4. Update PATH environment variable
5. Verify GCC is now accessible

#### 5. verifyWingetAvailable()

**Location**: `internal/updater/updater.go`

```go
func verifyWingetAvailable() error
```

Checks if winget is available by executing `winget --version`.
Returns error with installation instructions if not found.

#### 6. executeWingetInstall()

**Location**: `internal/updater/updater.go`

```go
func executeWingetInstall() error
```

Executes the winget command:
```
winget install BrechtSanders.WinLibs.POSIX.UCRT --silent --accept-source-agreements --accept-package-agreements
```

With timeout of 10 minutes.

#### 7. detectGCCInstallPath()

**Location**: `internal/updater/updater.go`

```go
func detectGCCInstallPath() (string, error)
```

After installation, searches for the GCC installation:
1. Check common WinLibs installation paths
2. Search Program Files directories
3. Use `where gcc` command as fallback
4. Return the bin directory path

#### 8. updatePATHEnvironment()

**Location**: `internal/updater/updater.go`

```go
func updatePATHEnvironment(gccBinPath string) error
```

Adds the GCC bin directory to the PATH environment variable for the current process.

### Modified Functions

#### downloadAndCompile()

**Location**: `internal/updater/updater.go`

Add GCC availability check at the beginning:

```go
func downloadAndCompile(version string) (string, error) {
    LogInfo("Setting up Go environment for compilation...")
    
    // On Windows, ensure GCC is available
    if runtime.GOOS == "windows" {
        LogInfo("Windows detected, ensuring GCC is available...")
        if err := ensureGCCAvailable(); err != nil {
            LogError("Failed to ensure GCC availability: %v", err)
            return "", fmt.Errorf("GCC not available and automatic installation failed: %w", err)
        }
        LogInfo("GCC is available and ready for compilation")
    }
    
    // ... rest of existing code
}
```

## Data Flow

```
downloadAndCompile() called
    ↓
[Windows only] ensureGCCAvailable()
    ↓
checkGCCInPath()
    ├─ Found → Return success
    └─ Not found → Continue
    ↓
checkGCCInCommonLocations()
    ├─ Found → Update PATH → Return success
    └─ Not found → Continue
    ↓
installGCCWithWinget()
    ↓
verifyWingetAvailable()
    ├─ Not found → Return error with instructions
    └─ Found → Continue
    ↓
executeWingetInstall()
    ├─ Timeout → Return error
    ├─ Failed → Return error
    └─ Success → Continue
    ↓
detectGCCInstallPath()
    ├─ Not found → Return error
    └─ Found → Continue
    ↓
updatePATHEnvironment()
    ↓
Verify GCC accessible
    ├─ Not accessible → Return error
    └─ Accessible → Return success
    ↓
Continue with compilation
```

## Error Handling

### Winget Not Available

```
[ERROR] winget is not available on this system
[ERROR] GCC installation requires winget (Windows Package Manager)
[ERROR] INSTALLATION INSTRUCTIONS:
[ERROR]   1. Install winget from: https://aka.ms/getwinget
[ERROR]   2. Or install App Installer from Microsoft Store
[ERROR]   3. After installing winget, retry the update
[ERROR] MANUAL GCC INSTALLATION:
[ERROR]   Run: winget install BrechtSanders.WinLibs.POSIX.UCRT
```

### GCC Installation Failed

```
[ERROR] Failed to install GCC via winget: <error details>
[ERROR] MANUAL INSTALLATION INSTRUCTIONS:
[ERROR]   1. Open PowerShell or Command Prompt as Administrator
[ERROR]   2. Run: winget install BrechtSanders.WinLibs.POSIX.UCRT --accept-source-agreements --accept-package-agreements
[ERROR]   3. Verify installation: gcc --version
[ERROR]   4. Retry the update
```

### GCC Installation Timeout

```
[ERROR] GCC installation timed out after 10 minutes
[ERROR] This may indicate network issues or system problems
[ERROR] RECOVERY INSTRUCTIONS:
[ERROR]   1. Check your internet connection
[ERROR]   2. Manually install GCC: winget install BrechtSanders.WinLibs.POSIX.UCRT
[ERROR]   3. Retry the update
```

### GCC Not Found After Installation

```
[ERROR] GCC was installed but cannot be found in PATH
[ERROR] Installation path detection failed
[ERROR] MANUAL RECOVERY:
[ERROR]   1. Find GCC installation: where gcc
[ERROR]   2. Add to PATH manually
[ERROR]   3. Or reinstall: winget uninstall BrechtSanders.WinLibs.POSIX.UCRT && winget install BrechtSanders.WinLibs.POSIX.UCRT
```

## Logging Strategy

### GCC Check Start
```
[INFO] Windows detected, ensuring GCC is available...
[INFO] Checking for GCC in PATH...
```

### GCC Found in PATH
```
[INFO] GCC found in PATH: C:\Program Files\WinLibs\mingw64\bin\gcc.exe
[INFO] GCC version: gcc (GCC) 13.2.0
[INFO] GCC is available and ready for compilation
```

### GCC Found in Common Location
```
[INFO] GCC not found in PATH
[INFO] Searching common installation directories...
[INFO] GCC found at: C:\Program Files\WinLibs\mingw64\bin\gcc.exe
[INFO] Adding GCC to PATH for current process
[INFO] GCC is now available for compilation
```

### GCC Installation Triggered
```
[INFO] GCC not found in PATH or common locations
[INFO] Automatic GCC installation will be attempted
[INFO] Verifying winget is available...
[INFO] winget version: v1.7.10582
[INFO] Executing: winget install BrechtSanders.WinLibs.POSIX.UCRT --silent --accept-source-agreements --accept-package-agreements
[INFO] GCC installation in progress (this may take several minutes)...
```

### GCC Installation Success
```
[INFO] GCC installation completed successfully
[INFO] Detecting GCC installation path...
[INFO] GCC installed at: C:\Program Files\WinLibs\mingw64\bin
[INFO] Adding GCC to PATH
[INFO] Verifying GCC is accessible...
[INFO] GCC version: gcc (GCC) 13.2.0
[INFO] GCC is available and ready for compilation
```

## Testing Strategy

### Unit Tests

1. **GCC Detection**
   - Test checkGCCInPath() with GCC in PATH
   - Test checkGCCInPath() without GCC in PATH
   - Test checkGCCInCommonLocations() with various installation paths

2. **Winget Verification**
   - Test verifyWingetAvailable() with winget installed
   - Test verifyWingetAvailable() without winget installed

3. **PATH Manipulation**
   - Test updatePATHEnvironment() adds GCC to PATH correctly
   - Test PATH update doesn't duplicate entries

### Integration Tests

1. **Full Installation Flow**
   - Test on Windows without GCC (requires winget)
   - Test on Windows with GCC already installed
   - Test on Windows with GCC in non-standard location

2. **Error Scenarios**
   - Test with winget not available
   - Test with network disconnected (installation timeout)
   - Test with insufficient permissions

3. **Compilation Verification**
   - Test that compilation succeeds after GCC installation
   - Test that CGO-enabled code compiles correctly

## Platform-Specific Considerations

### Windows Only

This feature is Windows-specific and should only execute on Windows:
- Use `runtime.GOOS == "windows"` checks
- All GCC installation logic is Windows-only
- Linux and macOS typically have GCC pre-installed or available via system package managers

### WinLibs POSIX UCRT

The specific GCC build matters:
- **POSIX**: Provides POSIX threading support
- **UCRT**: Uses Universal C Runtime (modern Windows runtime)
- This build is recommended for Go CGO compilation on Windows

### Common Installation Paths

WinLibs typically installs to:
- `C:\Program Files\WinLibs\mingw64\bin` (64-bit)
- `C:\Program Files\WinLibs\mingw32\bin` (32-bit)

## Security Considerations

1. **Command Injection**: Use `exec.Command()` with separate arguments, not shell execution
2. **PATH Manipulation**: Only modify PATH for current process, not system-wide
3. **Timeout**: Prevent indefinite hangs with 10-minute timeout
4. **Verification**: Always verify GCC is accessible after installation

## Performance Considerations

1. **Quick Check**: PATH check is fast (< 100ms)
2. **Installation Time**: GCC installation takes 2-5 minutes typically
3. **One-Time Cost**: Installation only happens once per system
4. **Caching**: Once installed, subsequent updates use existing GCC

## Backward Compatibility

- Existing systems with GCC already installed are unaffected
- The check is fast and non-intrusive
- No breaking changes to existing functionality
- Graceful degradation if winget is not available

## Future Enhancements

1. **Version Pinning**: Optionally specify a specific GCC version
2. **Alternative Installers**: Support chocolatey or scoop as fallbacks
3. **Pre-flight Check**: Check GCC availability before starting update
4. **Offline Installation**: Support installing from local package cache
