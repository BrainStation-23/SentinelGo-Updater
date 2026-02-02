# Implementation Plan

- [x] 1. Fix $HOME environment variable issue to prevent compilation failures
- [x] 1.1 Create ensureHomeDirectory() function with fallback strategies
  - Implement Strategy 1: Check $HOME environment variable
  - Implement Strategy 2: Use os.UserHomeDir()
  - Implement Strategy 3: Use user.Current() to get home directory
  - Implement Strategy 4: Parse /etc/passwd for current UID (Linux/Unix fallback)
  - Return error if all strategies fail
  - _Requirements: 5.1, 5.2_

- [x] 1.2 Create setEnvironmentVariables() function
  - Call ensureHomeDirectory() to get home directory
  - Set $HOME environment variable if not already set
  - Set $GOPATH environment variable if not already set (default to $HOME/go)
  - Add logging for each environment variable that is set
  - _Requirements: 5.3, 5.4_

- [x] 1.3 Call setEnvironmentVariables() at start of performUpdate()
  - Call before backup creation
  - Return error if environment setup fails
  - Log successful environment setup
  - _Requirements: 5.5_

- [x] 2. Modify cleanup phase to preserve backup file
  - Update `cleanupOldFiles()` to skip deletion of `sentinel.backup`
  - Keep deletion of main binary and `sentinel.old`
  - Add logging to indicate backup file is being preserved
  - Verify database and log files remain preserved
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [x] 3. Add post-success backup cleanup
  - Create `cleanupBackupFile()` helper function
  - Call `cleanupBackupFile()` in `performUpdate()` after successful update
  - Add logging when backup is deleted after success
  - Ensure cleanup only happens on success path, not error path
  - _Requirements: 2.1, 2.2, 2.3, 2.4_

- [x] 4. Update rollback to preserve backup for inspection
  - Modify `rollback()` to not delete backup file after successful rollback
  - Add logging message indicating backup is preserved for manual inspection
  - Update rollback error handling to provide clear recovery instructions
  - _Requirements: 3.1, 3.2, 3.3, 3.4_

- [x] 5. Enhance logging throughout backup lifecycle
  - Add log messages when backup is created
  - Add log messages when backup is preserved during cleanup
  - Add log messages when backup is deleted after success
  - Add log messages when backup is used during rollback
  - Add log messages for environment validation steps
  - _Requirements: 4.1, 4.2, 4.3, 4.4_

- [ ]* 6. Add unit tests for backup preservation
- [ ]* 6.1 Test cleanup preserves backup file
  - Create test that verifies `cleanupOldFiles()` preserves `sentinel.backup`
  - Verify main binary and `sentinel.old` are deleted
  - _Requirements: 1.1, 1.2, 1.3_

- [ ]* 6.2 Test post-update cleanup
  - Test that backup is deleted after successful update
  - Test that backup is preserved after failed update
  - _Requirements: 2.1, 2.2_

- [ ]* 6.3 Test environment variable handling
  - Test ensureHomeDirectory() with missing $HOME
  - Test ensureHomeDirectory() with os.UserHomeDir() fallback
  - Test setEnvironmentVariables() sets $HOME correctly
  - Test setEnvironmentVariables() sets $GOPATH correctly
  - Verify compilation succeeds after environment setup
  - _Requirements: 5.1, 5.2, 5.3, 5.4_
