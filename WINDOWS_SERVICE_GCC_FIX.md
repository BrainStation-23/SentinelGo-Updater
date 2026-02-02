# Windows Service GCC PATH Fix

## Problem

The updater service was failing to find GCC on Windows even though GCC was installed and accessible from the command line. This occurred because:

1. **Windows services run in a different environment** - They don't inherit the user's PATH environment variable
2. **GCC was installed but not in system PATH** - It was only in the user's PATH
3. **The service couldn't find gcc.exe** - Leading to compilation failures

## Root Cause

When a Windows service is created using `sc.exe create`, it runs under the SYSTEM account with a minimal environment that doesn't include the user's PATH. This means:

- User can run `gcc --version` successfully from PowerShell
- Service cannot find `gcc` because it doesn't have access to the user's PATH
- The updater service fails during compilation with "gcc not found" errors

## Solution

The fix involves two changes:

### 1. Service Installation with PATH Configuration

**File**: `internal/service/manager_windows.go`

When installing the service, we now:
1. Retrieve the system PATH from the Windows registry
2. Configure the service with an Environment registry value that includes PATH
3. This makes GCC and other system tools accessible to the service

```go
// Get the system PATH to include in service environment
systemPath := getSystemPATH()

// Configure service environment to include system PATH
if systemPath != "" {
    regKey := fmt.Sprintf("HKLM\\SYSTEM\\CurrentControlSet\\Services\\%s", serviceName)
    regCmd := exec.Command("reg.exe", "add", regKey,
        "/v", "Environment",
        "/t", "REG_MULTI_SZ",
        "/d", fmt.Sprintf("PATH=%s", systemPath),
        "/f",
    )
    regCmd.Run()
}
```

The `getSystemPATH()` function:
- Queries the system PATH from registry: `HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment`
- Falls back to the current process PATH if registry query fails
- Returns the complete PATH string for the service

### 2. Enhanced GCC Detection

**File**: `internal/updater/updater.go`

Expanded the list of common GCC installation paths to include:

**WinLibs installations**:
- `C:\Program Files\WinLibs\mingw64\bin`
- `C:\Program Files\WinLibs\mingw32\bin`
- `C:\Program Files (x86)\WinLibs\mingw64\bin`
- `C:\Program Files (x86)\WinLibs\mingw32\bin`

**MinGW installations**:
- `C:\MinGW\bin`
- `C:\MinGW64\bin`
- `C:\mingw64\bin`
- `C:\mingw32\bin`

**TDM-GCC**:
- `C:\TDM-GCC-64\bin`
- `C:\TDM-GCC-32\bin`

**MSYS2 installations**:
- `C:\msys64\mingw64\bin`
- `C:\msys64\mingw32\bin`
- `C:\msys64\ucrt64\bin`
- `C:\msys64\clang64\bin`
- `C:\msys32\mingw64\bin`
- `C:\msys32\mingw32\bin`

**mingw-w64 installations**:
- `C:\Program Files\mingw-w64\bin`
- `C:\Program Files (x86)\mingw-w64\bin`
- `C:\mingw-w64\bin`

**User-specific paths** (when USERPROFILE is set):
- `%USERPROFILE%\mingw64\bin`
- `%USERPROFILE%\mingw32\bin`
- `%USERPROFILE%\.mingw\bin`
- `%USERPROFILE%\scoop\apps\mingw\current\bin`
- `%USERPROFILE%\scoop\apps\gcc\current\bin`

## How It Works

### Update Process Flow

1. **Service starts update process**
2. **GCC availability check**:
   - First checks if `gcc` is in PATH (will fail for service initially)
   - Then searches common installation directories
   - Finds GCC at one of the common paths (e.g., `C:\MinGW64\bin`)
   - Adds that path to the current process PATH
3. **Compilation proceeds** with GCC now accessible
4. **Service reinstallation**:
   - Service is reinstalled with the system PATH configured
   - Next time the service starts, it will have PATH access from the beginning

### After First Successful Update

Once the service has been reinstalled with the PATH configuration:
- The service will have access to the system PATH on startup
- GCC will be found immediately in PATH check (Step 1)
- No need to search common locations
- Faster update process

## Testing

### Verify GCC is Accessible

From PowerShell:
```powershell
gcc --version
```

Expected output:
```
gcc.exe (MinGW-W64 x86_64-ucrt-posix-seh, built by Brecht Sanders, r5) 15.2.0
```

### Verify Service Configuration

After the service is reinstalled, check the registry:
```powershell
reg query "HKLM\SYSTEM\CurrentControlSet\Services\sentinelgo-updater" /v Environment
```

Should show the PATH environment variable configured for the service.

### Check Updater Logs

The updater log will show:
```
[INFO] Checking if GCC is already in PATH...
[INFO] GCC not found in PATH, checking common installation directories...
[INFO] Checking: C:\MinGW64\bin\gcc.exe
[INFO] GCC found at: C:\MinGW64\bin
[INFO] GCC version: gcc.exe (MinGW-W64 ...) 15.2.0
[INFO] Adding GCC to PATH for current process...
[INFO] PATH environment variable updated successfully
[INFO] GCC is now available for compilation
```

## Benefits

1. **No manual PATH configuration required** - The service automatically finds GCC
2. **Works with any GCC installation** - Supports WinLibs, MinGW, MSYS2, TDM-GCC, etc.
3. **Persistent solution** - Once configured, the service always has PATH access
4. **No GCC installation needed** - Uses existing GCC installation
5. **Simpler than automatic installation** - Avoids winget dependency and installation complexity

## Troubleshooting

### If GCC Still Not Found

1. **Check GCC installation location**:
   ```powershell
   where gcc
   ```

2. **If GCC is in a non-standard location**, add it to the system PATH:
   ```powershell
   # Open System Properties > Environment Variables
   # Add GCC bin directory to System PATH (not User PATH)
   ```

3. **Restart the updater service** after PATH changes:
   ```powershell
   sc stop sentinelgo-updater
   sc start sentinelgo-updater
   ```

### If Service Environment Not Configured

Manually configure the service environment:
```powershell
$gccPath = "C:\MinGW64\bin"  # Adjust to your GCC location
$systemPath = [Environment]::GetEnvironmentVariable("PATH", "Machine")
$servicePath = "$gccPath;$systemPath"

reg add "HKLM\SYSTEM\CurrentControlSet\Services\sentinelgo-updater" /v Environment /t REG_MULTI_SZ /d "PATH=$servicePath" /f

sc stop sentinelgo-updater
sc start sentinelgo-updater
```

## Migration Notes

### For Existing Installations

The fix will automatically apply on the next update:
1. Current update may still fail if GCC not in system PATH
2. Service will be reinstalled with PATH configuration
3. Subsequent updates will work correctly

### For New Installations

New installations will have PATH configured from the start if:
- GCC is installed before the first update
- GCC is in one of the common locations

## Comparison: Before vs After

### Before Fix

```
[ERROR] Failed to compile: exec: "gcc": executable file not found in %PATH%
[ERROR] Update failed
[INFO] Triggering rollback...
```

### After Fix

```
[INFO] GCC found at: C:\MinGW64\bin
[INFO] GCC version: gcc.exe (MinGW-W64 ...) 15.2.0
[INFO] Compilation successful
[INFO] Update completed successfully
```

## Related Files

- `internal/service/manager_windows.go` - Service installation with PATH configuration
- `internal/updater/updater.go` - GCC detection and PATH management
- `internal/updater/logging.go` - Logging functions

## References

- Windows Service Environment: https://docs.microsoft.com/en-us/windows/win32/services/service-user-accounts
- Registry Service Configuration: https://docs.microsoft.com/en-us/windows/win32/services/service-configuration
- MinGW-W64: https://www.mingw-w64.org/
- WinLibs: https://winlibs.com/
