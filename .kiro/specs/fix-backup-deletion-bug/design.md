# Design Document

## Overview

This design addresses a critical bug in the SentinelGo-Updater where backup files are prematurely deleted during the cleanup phase (Step 3), preventing successful rollback when updates fail in subsequent steps (Step 4+). The current implementation deletes `sentinel.backup` during cleanup, but then attempts to use it for rollback if the update fails, resulting in a "backup file not found" error and leaving the system in an unrecoverable state.

The fix involves restructuring the backup lifecycle to ensure backup files are preserved until the update completes successfully, and only then deleted as part of post-update cleanup.

## Architecture

### Current Flow (Problematic)

```
1. Create backup (sentinel.backup)
2. Stop service
3. Uninstall service
4. Cleanup old files → DELETES sentinel.backup ❌
5. Download & compile → FAILS
6. Rollback → Cannot find sentinel.backup ❌
```

### Proposed Flow (Fixed)

```
1. Create backup (sentinel.backup)
2. Stop service
3. Uninstall service
4. Cleanup old files → PRESERVES sentinel.backup ✓
5. Download & compile → FAILS
6. Rollback → Uses sentinel.backup ✓
7. (On success) Delete sentinel.backup ✓
```

## Components and Interfaces

### 1. Backup Lifecycle Management

**Modified Functions:**
- `cleanupOldFiles()` - Remove backup deletion logic
- `performUpdate()` - Add post-success backup cleanup
- `createBackup()` - No changes needed (already working correctly)
- `rollback()` - Add cleanup of backup after successful rollback

**Backup File States:**
- **Active Backup**: `sentinel.backup` - Created before update, preserved during cleanup
- **Legacy Backup**: `sentinel.old` - From previous updates, safe to delete during cleanup
- **Post-Update**: Backup deleted only after successful update completion

### 2. Cleanup Phase Modifications

The `cleanupOldFiles()` function currently deletes three items:
1. Main binary (`sentinel`) - ✓ Correct
2. Legacy backup (`sentinel.old`) - ✓ Correct  
3. Current backup (`sentinel.backup`) - ❌ **PROBLEM**

**Design Change:**
```go
func cleanupOldFiles() error {
    // Delete main agent binary - KEEP
    // Delete sentinel.old - KEEP
    // Delete sentinel.backup - REMOVE THIS
    // Preserve database - KEEP
    // Preserve logs - KEEP
}
```

### 3. Post-Update Cleanup

Add new cleanup logic after successful update:

```go
func performUpdate(targetVersion string) error {
    backup, err := createBackup(currentVersion)
    // ... existing update steps ...
    
    updateErr := func() error {
        // All update steps
    }()
    
    if updateErr != nil {
        // Rollback (backup still exists)
        rollback(backup)
        return updateErr
    }
    
    // NEW: Delete backup only after successful update
    cleanupBackupFile(backup.BackupPath)
    
    return nil
}
```

### 4. Environment Variable Handling

The logs show the root cause: `$HOME is not defined` causing compilation failure. This happens because the updater runs as a systemd service without a user session, so $HOME is not set in the environment.

**Root Cause Analysis:**
- Systemd services run without $HOME set by default
- The `downloadAndCompile()` function calls `os.UserHomeDir()` which fails when $HOME is not set
- This causes compilation to fail after destructive operations (cleanup) have already occurred
- The backup has been deleted, so rollback fails

**Design Solution:**
Implement robust home directory detection with multiple fallback strategies:

```go
func ensureHomeDirectory() (string, error) {
    // Strategy 1: Check $HOME environment variable
    if home := os.Getenv("HOME"); home != "" {
        return home, nil
    }
    
    // Strategy 2: Use os.UserHomeDir() (works on most systems)
    if home, err := os.UserHomeDir(); err == nil {
        return home, nil
    }
    
    // Strategy 3: Get current user and lookup home from passwd
    if currentUser, err := user.Current(); err == nil {
        return currentUser.HomeDir, nil
    }
    
    // Strategy 4: Construct from /etc/passwd (Linux/Unix)
    // Read /etc/passwd and find home directory for current UID
    
    return "", fmt.Errorf("unable to determine home directory")
}

func setEnvironmentVariables() error {
    // Ensure $HOME is set for child processes
    homeDir, err := ensureHomeDirectory()
    if err != nil {
        return err
    }
    
    if os.Getenv("HOME") == "" {
        os.Setenv("HOME", homeDir)
        LogInfo("Set $HOME to: %s", homeDir)
    }
    
    // Ensure GOPATH is set
    if os.Getenv("GOPATH") == "" {
        gopath := filepath.Join(homeDir, "go")
        os.Setenv("GOPATH", gopath)
        LogInfo("Set $GOPATH to: %s", gopath)
    }
    
    return nil
}
```

**Integration Point:**
Call `setEnvironmentVariables()` at the start of `performUpdate()` before any destructive operations.

## Data Models

### BackupInfo Structure (Existing)

```go
type BackupInfo struct {
    Version    string    // Version being backed up
    BackupPath string    // Path to backup file
    Timestamp  time.Time // When backup was created
}
```

No changes needed to the data model.

## Error Handling

### Current Error Handling Issues

1. **Backup deleted before use**: Cleanup runs before risky operations complete
2. **No pre-flight checks**: Environment issues discovered after destructive operations
3. **Defer cleanup timing**: The defer in `performUpdate()` doesn't execute on error paths

### Improved Error Handling Strategy

**Phase 1: Environment Setup**
```go
// Before any destructive operations
- Detect and set $HOME if not defined
- Set GOPATH if not defined
- Verify Go toolchain availability
- Ensure sufficient disk space
```

**Phase 2: Pre-flight Validation**
```go
// Validate environment is ready
- Verify current binary exists
- Check network connectivity
- Validate target version format
```

**Phase 3: Backup Creation**
```go
// Create backup before any changes
- Create sentinel.backup
- Verify backup integrity
- Store BackupInfo
```

**Phase 4: Destructive Operations**
```go
// Stop, uninstall, cleanup (preserve backup)
- Stop service
- Uninstall service
- Delete main binary
- Delete sentinel.old (legacy)
- PRESERVE sentinel.backup
```

**Phase 5: Risky Operations**
```go
// Download, compile, install
- Download source
- Compile binary (with $HOME now set)
- Install binary
- Reinstall service
- Start service
- Verify running
```

**Phase 6: Cleanup**
```go
// Only on success
- Delete sentinel.backup
- Log successful update
```

**Phase 7: Rollback (on any error)**
```go
// Restore from backup
- Verify sentinel.backup exists
- Restore binary
- Reinstall service
- Start service
- Verify running
- PRESERVE sentinel.backup for manual inspection
```

### Error Recovery Matrix

| Failure Point | Backup State | Recovery Action |
|--------------|--------------|-----------------|
| Environment setup | Not created | Abort, no changes made |
| Pre-flight validation | Not created | Abort, no changes made |
| Backup creation | Failed | Abort, no changes made |
| Service stop/uninstall | Exists | Rollback possible |
| Cleanup | Exists (preserved) | Rollback possible |
| Download/compile | Exists | Rollback possible (now succeeds with $HOME set) |
| Install/start | Exists | Rollback possible |
| Verification | Exists | Rollback possible |
| Rollback itself | Exists | Manual intervention needed |

## Testing Strategy

### Unit Tests

1. **Test `cleanupOldFiles()` preserves backup**
   - Create sentinel, sentinel.old, sentinel.backup
   - Run cleanup
   - Verify sentinel and sentinel.old deleted
   - Verify sentinel.backup preserved

2. **Test post-update backup cleanup**
   - Simulate successful update
   - Verify backup deleted after success
   - Verify backup preserved after failure

3. **Test environment variable handling**
   - Test with missing $HOME
   - Test with missing GOPATH
   - Test with missing Go toolchain
   - Verify $HOME is set for child processes
   - Verify compilation succeeds after environment setup

### Integration Tests

1. **Test full update cycle with backup preservation**
   - Create initial binary
   - Trigger update
   - Simulate failure at various points
   - Verify rollback succeeds
   - Verify backup exists after rollback

2. **Test successful update cleanup**
   - Complete full successful update
   - Verify backup deleted
   - Verify no orphaned files

3. **Test rollback after cleanup phase**
   - Update fails after cleanup
   - Verify backup still exists
   - Verify rollback succeeds

### Manual Testing Scenarios

1. **Simulate compilation failure**
   - Set invalid GOPATH
   - Trigger update
   - Verify rollback succeeds
   - Verify system returns to working state

2. **Simulate network failure**
   - Disconnect network during download
   - Verify rollback succeeds

3. **Simulate service start failure**
   - Corrupt new binary
   - Verify rollback succeeds

## Implementation Notes

### Key Changes Required

1. **internal/updater/updater.go**
   - Add `ensureHomeDirectory()` function with multiple fallback strategies
   - Add `setEnvironmentVariables()` function to set $HOME and GOPATH
   - Call `setEnvironmentVariables()` at start of `performUpdate()`
   - Modify `cleanupOldFiles()` to skip `sentinel.backup` deletion
   - Add `cleanupBackupFile()` helper function
   - Call `cleanupBackupFile()` after successful update
   - Update rollback to preserve backup for inspection

2. **Logging Improvements**
   - Log home directory detection method used
   - Log when environment variables are set
   - Log when backup is preserved during cleanup
   - Log when backup is deleted after success
   - Log when backup is preserved after rollback
   - Add clear messages about backup file lifecycle

### Backward Compatibility

- No configuration changes required
- No API changes
- Existing backup files will be handled correctly
- Legacy `sentinel.old` files still cleaned up

### Performance Considerations

- Minimal performance impact (one less file deletion during cleanup)
- Backup file typically 10-20MB, negligible disk space impact
- Backup preserved temporarily, cleaned up after success or manual intervention

### Security Considerations

- Backup files contain executable code, should have restricted permissions (0755)
- Backup files should be owned by root on Unix systems
- No sensitive data in backup files (just binary executable)
- Backup files should be in protected directories (same as main binary)

## Rollback Behavior Changes

### Current Behavior
- Rollback fails if backup missing
- System left in broken state
- Manual intervention required

### New Behavior
- Rollback succeeds because backup preserved
- System automatically recovers
- Backup preserved after rollback for inspection
- Admin can manually delete backup after verifying system health

## Logging Enhancements

### New Log Messages

**During Cleanup:**
```
[INFO] Checking for backup file: /path/to/sentinel.backup
[INFO] Preserving backup file for potential rollback: /path/to/sentinel.backup
```

**After Successful Update:**
```
[INFO] Update completed successfully, cleaning up backup file
[INFO] Deleted backup file: /path/to/sentinel.backup
```

**After Successful Rollback:**
```
[INFO] Rollback completed successfully
[INFO] Backup file preserved at: /path/to/sentinel.backup
[INFO] You may manually delete the backup after verifying system health
```

**During Pre-flight Validation:**
```
[INFO] Setting up environment for update...
[INFO] Checking $HOME environment variable...
[INFO] $HOME not set, using fallback method: os.UserHomeDir()
[INFO] Set $HOME to: /home/bs00927
[INFO] Set $GOPATH to: /home/bs00927/go
[INFO] Environment setup completed successfully
```

## Migration Path

1. Deploy updated updater binary
2. Existing systems will automatically use new logic
3. Any existing `sentinel.backup` files will be handled correctly
4. No manual intervention required for migration
