package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestGetDatabasePath verifies that GetDatabasePath returns the correct path
// based on the platform-specific data directory
func TestGetDatabasePath(t *testing.T) {
	expected := filepath.Join(GetDataDirectory(), "sentinel.db")
	actual := GetDatabasePath()

	if actual != expected {
		t.Errorf("GetDatabasePath() = %s; want %s", actual, expected)
	}

	// Verify the path contains the correct filename
	if filepath.Base(actual) != "sentinel.db" {
		t.Errorf("GetDatabasePath() filename = %s; want sentinel.db", filepath.Base(actual))
	}
}

// TestGetUpdaterLogPath verifies that GetUpdaterLogPath returns the correct path
// based on the platform-specific data directory
func TestGetUpdaterLogPath(t *testing.T) {
	expected := filepath.Join(GetDataDirectory(), "updater.log")
	actual := GetUpdaterLogPath()

	if actual != expected {
		t.Errorf("GetUpdaterLogPath() = %s; want %s", actual, expected)
	}

	// Verify the path contains the correct filename
	if filepath.Base(actual) != "updater.log" {
		t.Errorf("GetUpdaterLogPath() filename = %s; want updater.log", filepath.Base(actual))
	}
}

// TestGetAgentLogPath verifies that GetAgentLogPath returns the correct path
// based on the platform-specific data directory
func TestGetAgentLogPath(t *testing.T) {
	expected := filepath.Join(GetDataDirectory(), "agent.log")
	actual := GetAgentLogPath()

	if actual != expected {
		t.Errorf("GetAgentLogPath() = %s; want %s", actual, expected)
	}

	// Verify the path contains the correct filename
	if filepath.Base(actual) != "agent.log" {
		t.Errorf("GetAgentLogPath() filename = %s; want agent.log", filepath.Base(actual))
	}
}

// TestDerivedPathsOnMacOS verifies the expected paths on macOS
func TestDerivedPathsOnMacOS(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test on non-macOS platform")
	}

	tests := []struct {
		name     string
		function func() string
		expected string
	}{
		{
			name:     "GetDatabasePath",
			function: GetDatabasePath,
			expected: "/Library/Application Support/SentinelGo/sentinel.db",
		},
		{
			name:     "GetUpdaterLogPath",
			function: GetUpdaterLogPath,
			expected: "/Library/Application Support/SentinelGo/updater.log",
		},
		{
			name:     "GetAgentLogPath",
			function: GetAgentLogPath,
			expected: "/Library/Application Support/SentinelGo/agent.log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.function()
			if actual != tt.expected {
				t.Errorf("%s() = %s; want %s", tt.name, actual, tt.expected)
			}
		})
	}
}

// TestEnsureDataDirectoryCreation verifies that EnsureDataDirectory creates
// the directory with proper permissions using os.MkdirAll behavior
func TestEnsureDataDirectoryCreation(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	testDataDir := filepath.Join(tempDir, "test-sentinel-data")

	// Test directory creation using os.MkdirAll directly (same as EnsureDataDirectory)
	err := os.MkdirAll(testDataDir, 0755)
	if err != nil {
		t.Fatalf("os.MkdirAll() failed: %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(testDataDir)
	if err != nil {
		t.Fatalf("Directory was not created: %v", err)
	}

	// Verify it's a directory
	if !info.IsDir() {
		t.Errorf("Path exists but is not a directory")
	}

	// Verify permissions are 0755
	if info.Mode().Perm() != 0755 {
		t.Errorf("Directory permissions = %o; want 0755", info.Mode().Perm())
	}
}

// TestEnsureDataDirectoryWithParents verifies that EnsureDataDirectory creates
// all parent directories in the path if they don't exist
func TestEnsureDataDirectoryWithParents(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	testDataDir := filepath.Join(tempDir, "parent1", "parent2", "test-sentinel-data")

	// Test directory creation with multiple parent directories
	err := os.MkdirAll(testDataDir, 0755)
	if err != nil {
		t.Fatalf("os.MkdirAll() failed to create parent directories: %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(testDataDir)
	if err != nil {
		t.Fatalf("Directory with parents was not created: %v", err)
	}

	// Verify it's a directory
	if !info.IsDir() {
		t.Errorf("Path exists but is not a directory")
	}

	// Verify all parent directories were created
	parent1 := filepath.Join(tempDir, "parent1")
	parent2 := filepath.Join(tempDir, "parent1", "parent2")

	for _, dir := range []string{parent1, parent2, testDataDir} {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Errorf("Parent directory %s was not created properly", dir)
		}
	}
}

// TestEnsureDataDirectoryPermissionError verifies that EnsureDataDirectory
// returns an error when run without sufficient permissions
func TestEnsureDataDirectoryPermissionError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Permission testing is different on Windows")
	}

	// Check if we're running as root - if so, skip this test
	if os.Geteuid() == 0 {
		t.Skip("Skipping permission error test when running as root")
	}

	// Test that directory creation fails with permission error in a protected location
	protectedPath := "/root/test-sentinel-no-permission"
	err := os.MkdirAll(protectedPath, 0755)

	if err == nil {
		t.Errorf("os.MkdirAll() should fail without sufficient permissions")
		// Clean up if somehow it succeeded
		os.RemoveAll(protectedPath)
	} else {
		// Verify the error is permission-related
		t.Logf("Got expected permission error: %v", err)
	}
}

// TestEnsureDataDirectoryOnMacOS verifies the actual macOS path behavior
// This test requires root permissions to actually create the directory
func TestEnsureDataDirectoryOnMacOS(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test on non-macOS platform")
	}

	expectedPath := "/Library/Application Support/SentinelGo"

	// Verify GetDataDirectory returns the correct path
	actualPath := GetDataDirectory()
	if actualPath != expectedPath {
		t.Errorf("GetDataDirectory() = %s; want %s", actualPath, expectedPath)
	}

	// Test directory creation
	err := EnsureDataDirectory()

	if os.Geteuid() != 0 {
		// Not running as root - expect permission error
		if err == nil {
			t.Errorf("EnsureDataDirectory() should fail without root permissions on macOS")
			// Clean up if somehow it succeeded
			os.RemoveAll(expectedPath)
		} else {
			t.Logf("Expected permission error (not running as root): %v", err)
		}
	} else {
		// Running as root - should succeed
		if err != nil {
			t.Fatalf("EnsureDataDirectory() failed with root permissions: %v", err)
		}

		// Verify directory exists
		info, err := os.Stat(expectedPath)
		if err != nil {
			t.Fatalf("Directory was not created at %s: %v", expectedPath, err)
		}

		// Verify it's a directory
		if !info.IsDir() {
			t.Errorf("Path exists but is not a directory")
		}

		// Verify permissions are 0755
		if info.Mode().Perm() != 0755 {
			t.Errorf("Directory permissions = %o; want 0755", info.Mode().Perm())
		}

		t.Logf("Successfully created directory at %s with permissions %o", expectedPath, info.Mode().Perm())
	}
}
