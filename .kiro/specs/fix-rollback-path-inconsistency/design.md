# Design Document: Fix Rollback Path Inconsistency

## Overview

This design addresses a critical bug where the rollback process uses a different binary path than the one detected during the update, causing rollback failures. The root cause is that the update process detects the binary path dynamically (e.g., `C:\WINDOWS\system32\config\systemprofile\go\bin\sentinel.exe`), but the rollback process calls `paths.GetMainAgentBinaryPath()` which may return a different path (e.g., `C:\Program Files\SentinelGo\sentinel.exe`), resulting in a "path not found" error.

### Current Behavior (Buggy)

1. **Update starts**: Binary detected at `C:\WINDOWS\system32\config\systemprofile\go\bin\sentinel.exe`
2. **Backup created**: Backup file at `C:\WINDOWS\system32\config\systemprofile\go\bin\sentinel.exe.backup`
3. **Update fails**: Compilation error (missing GCC)
4. **Rollback triggered**: Calls `paths.GetMainAgentBinaryPath()` which returns `C:\Program Files\SentinelGo\sentinel.exe`
5. **Rollback fails**: Tries to write to `C:\Program Files\SentinelGo\sentinel.exe` but directory doesn't exist

### Root Cause Analysis

The bug occurs because:
- The binary path is detected fresh at the start of the update
- The backup is created using the detected path
- During rollback, `paths.GetMainAgentBinaryPath()` is called again, which may:
  - Use a cached value from a previous detection
  - Use a fallback hardcoded path if detection fails
  - Return a different path due to changed system state (service uninstalled)

## Architecture

### Update Context Structure

Introduce an `UpdateContext` structure that maintains all path information throughout the update lifecycle:

```go
type UpdateContext struct {
    // Detected paths
    BinaryPath      string    // Original binary location
    BinaryDir       string    // Directory containing the binary
    BackupPath      string    // Backup file location
    
    // Version information
    CurrentVersion  string    // Version before update
    TargetVersion   string    // Version to update to
    
    // Timestamps
    StartTime       time.Time // When update started
    
    // Detection metadata
    DetectionMethod string    // How the binary was detected
}
```

### Component Changes

#### 1. Update Context Creation

**Location**: `internal/updater/updater.go`

Add a new function to create the update context at the start of the update:

```go
func createUpdateContext(targetVersion string) (*UpdateContext, error)
```

This function will:
- Detect the binary path using the existing detection logic
- Validate that the binary exists and is accessible
- Construct the backup path from the detected binary path
- Store all metadata for use throughout the update

#### 2. Backup Creation

**Location**: `internal/updater/updater.go`

Modify `createBackup()` to accept and use the update context:

```go
func createBackup(ctx *UpdateContext) error
```

Changes:
- Use `ctx.BinaryPath` instead of calling `paths.GetMainAgentBinaryPath()`
- Use `ctx.BackupPath` for the backup file location
- Store the backup path in the context

#### 3. Rollback Process

**Location**: `internal/updater/updater.go`

Modify `rollback()` to accept and use the update context:

```go
func rollback(ctx *UpdateContext) error
```

Changes:
- Use `ctx.BinaryPath` as the restore target instead of calling `paths.GetMainAgentBinaryPath()`
- Use `ctx.BackupPath` to locate the backup file
- Ensure the target directory exists before attempting restoration
- Log both paths for debugging

#### 4. Cleanup Process

**Location**: `internal/updater/updater.go`

Modify `cleanupOldFiles()` to accept the update context:

```go
func cleanupOldFiles(ctx *UpdateContext) error
```

Changes:
- Use `ctx.BinaryPath` instead of calling `paths.GetMainAgentBinaryPath()`
- Use `ctx.BackupPath` for backup file operations

#### 5. Install Binary

**Location**: `internal/updater/updater.go`

Modify `installBinary()` to accept the update context:

```go
func installBinary(sourcePath string, ctx *UpdateContext) error
```

Changes:
- Use `ctx.BinaryPath` as the installation target
- Ensure the target directory exists before installation

### Data Flow

```
performUpdate(targetVersion)
    ↓
createUpdateContext(targetVersion)
    ├─ Detect binary path → ctx.BinaryPath
    ├─ Construct backup path → ctx.BackupPath
    └─ Validate paths
    ↓
createBackup(ctx)
    ├─ Read from ctx.BinaryPath
    └─ Write to ctx.BackupPath
    ↓
cleanupOldFiles(ctx)
    └─ Delete ctx.BinaryPath
    ↓
downloadAndCompile(targetVersion)
    ↓
installBinary(newBinaryPath, ctx)
    └─ Write to ctx.BinaryPath
    ↓
[If error occurs]
    ↓
rollback(ctx)
    ├─ Read from ctx.BackupPath
    └─ Write to ctx.BinaryPath
```

## Error Handling

### Path Validation

Before starting the update, validate:
1. Binary path exists and is readable
2. Binary directory is writable (for backup creation)
3. Binary can be executed (version check succeeds)

If validation fails, abort the update before any modifications.

### Rollback Path Verification

During rollback:
1. Verify backup file exists at `ctx.BackupPath`
2. Verify target directory exists (create if missing)
3. Verify target directory is writable
4. Log both source and target paths before restoration

### Directory Creation

If the target directory doesn't exist during rollback:
- Create it with appropriate permissions (0755 on Unix, default on Windows)
- Log the directory creation
- Proceed with restoration

## Logging Strategy

### Update Start
```
[INFO] === Starting update to v1.6.121 ===
[INFO] Creating update context...
[INFO] Binary path detected: C:\WINDOWS\system32\config\systemprofile\go\bin\sentinel.exe
[INFO] Detection method: common_installation_directory
[INFO] Backup path: C:\WINDOWS\system32\config\systemprofile\go\bin\sentinel.exe.backup
[INFO] Binary directory: C:\WINDOWS\system32\config\systemprofile\go\bin
```

### Backup Creation
```
[INFO] Creating backup using context paths...
[INFO] Source: C:\WINDOWS\system32\config\systemprofile\go\bin\sentinel.exe
[INFO] Destination: C:\WINDOWS\system32\config\systemprofile\go\bin\sentinel.exe.backup
[INFO] Backup created successfully (26973794 bytes)
```

### Rollback Start
```
[INFO] === Starting rollback process ===
[INFO] Using update context for consistent paths
[INFO] Backup file: C:\WINDOWS\system32\config\systemprofile\go\bin\sentinel.exe.backup
[INFO] Restore target: C:\WINDOWS\system32\config\systemprofile\go\bin\sentinel.exe
[INFO] Paths are consistent ✓
```

### Path Inconsistency Warning (if detected)
```
[WARNING] Path inconsistency detected!
[WARNING] Backup source: C:\WINDOWS\system32\config\systemprofile\go\bin\sentinel.exe
[WARNING] Restore target: C:\Program Files\SentinelGo\sentinel.exe
[WARNING] This may indicate a configuration issue
```

## Testing Strategy

### Unit Tests

1. **Update Context Creation**
   - Test context creation with valid binary path
   - Test context creation with invalid binary path
   - Test backup path construction

2. **Path Consistency**
   - Test that backup and restore use the same path
   - Test that cleanup uses the correct path
   - Test that install uses the correct path

3. **Directory Creation**
   - Test rollback when target directory doesn't exist
   - Test rollback when target directory exists but is empty
   - Test rollback when target directory exists with files

### Integration Tests

1. **Full Update Cycle**
   - Test update with binary in GOPATH location
   - Test update with binary in Program Files location
   - Test update with binary in custom location

2. **Rollback Scenarios**
   - Test rollback after compilation failure
   - Test rollback after installation failure
   - Test rollback after service start failure

3. **Platform-Specific Tests**
   - Test on Windows with various installation paths
   - Test on Linux with various installation paths
   - Test on macOS with various installation paths

## Migration Strategy

### Backward Compatibility

The changes are backward compatible because:
- The `UpdateContext` is an internal structure
- External APIs remain unchanged
- The path detection logic remains the same
- Only the internal flow of path usage changes

### Deployment

1. Deploy the updated updater binary
2. The next update cycle will use the new context-based approach
3. No manual intervention required

## Platform-Specific Considerations

### Windows

- Handle paths with drive letters (e.g., `C:\`)
- Handle paths in system profile directories
- Handle paths in Program Files directories
- Use `filepath.Dir()` for directory extraction
- Use `os.MkdirAll()` for directory creation

### Linux

- Handle paths in `/usr/local/bin`, `/usr/bin`, `/opt`
- Handle paths in user home directories
- Set proper permissions (0755) and ownership (root:root if running as root)

### macOS

- Handle paths in `/usr/local/bin`, `/Applications`
- Handle paths in user home directories
- Set proper permissions (0755)

## Security Considerations

1. **Path Validation**: Ensure paths don't contain directory traversal attempts
2. **Permission Checks**: Verify write permissions before attempting file operations
3. **Atomic Operations**: Use atomic file operations where possible
4. **Backup Integrity**: Verify backup file integrity before rollback

## Performance Considerations

1. **Single Detection**: Binary path is detected once at the start of the update
2. **No Re-detection**: Rollback uses cached path from context, avoiding re-detection overhead
3. **Minimal Overhead**: Context structure is lightweight (< 1KB)

## Future Enhancements

1. **Context Persistence**: Optionally persist the update context to disk for recovery after crashes
2. **Multi-Version Rollback**: Support rolling back multiple versions
3. **Backup Rotation**: Implement backup rotation to keep N previous versions
4. **Health Checks**: Add pre-update health checks to validate system state
