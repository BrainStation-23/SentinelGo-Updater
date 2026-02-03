# Design Document

## Overview

This design addresses the platform-specific path handling in the SentinelGo updater by modifying the `internal/paths` package to use macOS-native directory conventions. The change is minimal and focused: separating the `darwin` case from `linux` in the switch statement to return `/Library/Application Support/SentinelGo` for macOS while maintaining `/var/lib/sentinelgo` for Linux.

## Architecture

The architecture remains unchanged - the `paths` package continues to provide a centralized location for all path-related logic. The modification is isolated to the `GetDataDirectory()` function's platform detection logic.

### Current Architecture
```
internal/paths/paths.go
├── GetDataDirectory() - Returns platform-specific data directory
├── GetDatabasePath() - Builds on GetDataDirectory()
├── GetUpdaterLogPath() - Builds on GetDataDirectory()
├── GetAgentLogPath() - Builds on GetDataDirectory()
├── GetBinaryDirectory() - Returns platform-specific binary directory
├── GetMainAgentBinaryPath() - Builds on GetBinaryDirectory()
└── EnsureDataDirectory() - Creates data directory with proper permissions
```

All derived paths (database, logs) automatically inherit the correct platform-specific base directory.

## Components and Interfaces

### Modified Component: GetDataDirectory()

**Current Implementation:**
```go
func GetDataDirectory() string {
	switch runtime.GOOS {
	case "windows":
		// Windows logic
	case "darwin", "linux":  // ← Problem: treats macOS and Linux the same
		return "/var/lib/sentinelgo"
	default:
		return "/var/lib/sentinelgo"
	}
}
```

**New Implementation:**
```go
func GetDataDirectory() string {
	switch runtime.GOOS {
	case "windows":
		programData := os.Getenv("ProgramData")
		if programData == "" {
			programData = "C:\\ProgramData"
		}
		return filepath.Join(programData, "SentinelGo")
	case "darwin":
		return "/Library/Application Support/SentinelGo"
	case "linux":
		return "/var/lib/sentinelgo"
	default:
		return "/var/lib/sentinelgo"
	}
}
```

### Platform-Specific Paths

| Platform | Data Directory | Requires Root | Convention |
|----------|---------------|---------------|------------|
| macOS | `/Library/Application Support/SentinelGo` | Yes | macOS standard |
| Linux | `/var/lib/sentinelgo` | Yes | FHS standard |
| Windows | `%ProgramData%\SentinelGo` | Yes | Windows standard |

### Unchanged Components

- `GetBinaryDirectory()` - Already correctly uses `/usr/local/bin` for both macOS and Linux
- `EnsureDataDirectory()` - No changes needed; works with any path
- All derived path functions - Automatically use the corrected base directory

## Data Models

No data model changes required. This is purely a path configuration change.

## Error Handling

### Existing Error Handling (Preserved)

The `EnsureDataDirectory()` function already handles:
- Permission errors when creating directories
- Returns `os.MkdirAll()` errors directly to caller
- Caller is responsible for interpreting and handling errors

### Expected Behavior

On macOS:
- If run without root: `EnsureDataDirectory()` will fail with permission error
- If run with root: Directory will be created at `/Library/Application Support/SentinelGo`

On Linux:
- If run without root: `EnsureDataDirectory()` will fail with permission error  
- If run with root: Directory will be created at `/var/lib/sentinelgo`

This matches the existing behavior - the change only affects which path is used, not how errors are handled.

## Testing Strategy

### Manual Testing

1. **macOS Testing:**
   - Run without sudo: Verify error message indicates permission issue for `/Library/Application Support/SentinelGo`
   - Run with sudo: Verify directory is created at `/Library/Application Support/SentinelGo`
   - Verify database and log files are created in the correct location

2. **Linux Testing:**
   - Verify existing behavior is unchanged
   - Confirm `/var/lib/sentinelgo` is still used

3. **Cross-Platform Verification:**
   - Print `GetDataDirectory()` output on each platform
   - Verify all derived paths (database, logs) use correct base directory

### Code Review Checklist

- [ ] Verify `darwin` case is separate from `linux` case
- [ ] Confirm macOS path uses `/Library/Application Support/SentinelGo`
- [ ] Confirm Linux path remains `/var/lib/sentinelgo`
- [ ] Verify comment documentation is updated
- [ ] Check that no other code depends on the old macOS path

## Implementation Notes

### Minimal Change Approach

This design intentionally keeps changes minimal:
- Only modifies the switch statement in `GetDataDirectory()`
- No new functions or interfaces
- No changes to error handling
- No changes to permission model
- All other functions automatically benefit from the fix

### Documentation Updates

Update the function comment for `GetDataDirectory()` to reflect the new behavior:

```go
// GetDataDirectory returns the platform-specific data directory
// macOS: /Library/Application Support/SentinelGo
// Linux: /var/lib/sentinelgo
// Windows: %ProgramData%\SentinelGo
```

### Backward Compatibility

This is a breaking change for existing macOS installations that may have created `/var/lib/sentinelgo`. However:
- `/var/lib/sentinelgo` is non-standard on macOS
- Most macOS systems won't have this directory
- The fix aligns with platform conventions
- Migration is not in scope for this change
