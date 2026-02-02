# Implementation Plan

- [ ] 1. Create UpdateContext structure and initialization
  - Define UpdateContext struct with all required fields (BinaryPath, BackupPath, BinaryDir, CurrentVersion, TargetVersion, StartTime, DetectionMethod)
  - Implement createUpdateContext() function that detects binary path and constructs all context fields
  - Add path validation logic to ensure binary exists and is accessible before proceeding
  - Add logging for context creation with all detected paths
  - _Requirements: 1.1, 1.2, 2.1, 2.2, 4.1, 7.1, 7.2_

- [ ] 2. Refactor backup creation to use UpdateContext
  - Modify createBackup() signature to accept UpdateContext parameter
  - Update backup creation to use ctx.BinaryPath instead of paths.GetMainAgentBinaryPath()
  - Update backup path to use ctx.BackupPath
  - Update BackupInfo struct to store context reference or relevant context fields
  - Add logging to show source and destination paths from context
  - _Requirements: 1.1, 1.2, 3.1, 3.2, 3.3, 4.2_

- [ ] 3. Refactor rollback process to use UpdateContext
  - Modify rollback() signature to accept UpdateContext parameter instead of BackupInfo
  - Update rollback to use ctx.BinaryPath as restore target instead of calling paths.GetMainAgentBinaryPath()
  - Update rollback to use ctx.BackupPath to locate backup file
  - Add directory existence check and creation logic before restoration
  - Add path consistency validation and logging
  - Update error messages to include both backup and restore paths
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 2.3, 4.3, 5.1, 5.2, 5.3, 5.4, 8.1, 8.2, 8.3_

- [ ] 4. Refactor cleanup process to use UpdateContext
  - Modify cleanupOldFiles() signature to accept UpdateContext parameter
  - Update cleanup to use ctx.BinaryPath instead of paths.GetMainAgentBinaryPath()
  - Update cleanup to use ctx.BackupPath for backup file operations
  - Ensure backup file preservation logic uses context paths
  - _Requirements: 1.1, 1.2, 2.3_

- [ ] 5. Refactor install binary to use UpdateContext
  - Modify installBinary() signature to accept UpdateContext parameter
  - Update installation to use ctx.BinaryPath as target instead of paths.GetMainAgentBinaryPath()
  - Add directory creation logic if target directory doesn't exist
  - Add validation that target directory is writable
  - _Requirements: 1.1, 1.2, 5.2, 5.3, 6.1, 6.2, 6.3, 6.4_

- [ ] 6. Update performUpdate() to create and pass context
  - Create UpdateContext at the start of performUpdate() before any operations
  - Pass context to createBackup() call
  - Pass context to cleanupOldFiles() call
  - Pass context to installBinary() call
  - Pass context to rollback() call in error handler
  - Update all function calls to use context-based signatures
  - Add early validation and fail-fast logic if context creation fails
  - _Requirements: 1.1, 2.1, 2.2, 2.3, 2.4, 7.1, 7.2, 7.3, 7.4_

- [ ] 7. Add comprehensive logging throughout the update lifecycle
  - Add context creation logging with all detected paths
  - Add backup creation logging with source and destination paths
  - Add rollback logging with backup and restore paths
  - Add path consistency validation logging
  - Add directory creation logging when creating missing directories
  - Add warning logs if path inconsistencies are detected
  - _Requirements: 4.1, 4.2, 4.3, 4.4, 5.4_

- [ ] 8. Add directory creation and validation logic
  - Implement directory existence check in rollback process
  - Implement directory creation with proper permissions (0755 on Unix)
  - Add validation that directories are writable before file operations
  - Add error handling for directory creation failures
  - _Requirements: 5.2, 5.3, 6.1, 6.2, 6.3, 6.4_

- [ ] 9. Update error messages and recovery instructions
  - Update rollback error messages to include both backup and restore paths
  - Update recovery instructions to reference actual paths from context
  - Add specific instructions for manual directory creation if needed
  - Add backup file integrity verification instructions
  - _Requirements: 5.4, 8.1, 8.2, 8.3, 8.4_

- [ ] 10. Test the fix on Windows with various installation paths
  - Test with binary in GOPATH location (C:\Users\...\go\bin\sentinel.exe)
  - Test with binary in Program Files (C:\Program Files\SentinelGo\sentinel.exe)
  - Test with binary in system profile (C:\WINDOWS\system32\config\systemprofile\go\bin\sentinel.exe)
  - Verify rollback works correctly in all scenarios
  - Verify paths are consistent throughout update lifecycle
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 6.1, 6.2, 6.3, 6.4_
