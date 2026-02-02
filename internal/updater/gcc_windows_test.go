//go:build windows
// +build windows

package updater

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckGCCInPath tests the checkGCCInPath function
func TestCheckGCCInPath(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() func()
		expected bool
	}{
		{
			name: "GCC in PATH",
			setup: func() func() {
				// Check if GCC is actually in PATH
				_, err := exec.LookPath("gcc")
				if err != nil {
					t.Skip("GCC not in PATH, skipping test")
				}
				return func() {}
			},
			expected: true,
		},
		{
			name: "GCC not in PATH",
			setup: func() func() {
				// Save original PATH
				originalPath := os.Getenv("PATH")
				// Set PATH to empty to simulate GCC not being available
				os.Setenv("PATH", "")
				return func() {
					// Restore original PATH
					os.Setenv("PATH", originalPath)
				}
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setup()
			defer cleanup()

			result := checkGCCInPath()
			if result != tt.expected {
				t.Errorf("checkGCCInPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestCheckGCCInCommonLocations tests the checkGCCInCommonLocations function
func TestCheckGCCInCommonLocations(t *testing.T) {
	// This test checks if the function can find GCC in common locations
	// We'll test with actual system state
	path, err := checkGCCInCommonLocations()

	// If GCC is found, verify the path is valid
	if err == nil {
		if path == "" {
			t.Error("checkGCCInCommonLocations() returned empty path with no error")
		}

		// Verify gcc.exe exists at the returned path
		gccExe := filepath.Join(path, "gcc.exe")
		if _, statErr := os.Stat(gccExe); statErr != nil {
			t.Errorf("checkGCCInCommonLocations() returned path %s, but gcc.exe not found: %v", path, statErr)
		}

		t.Logf("GCC found at: %s", path)
	} else {
		// GCC not found in common locations - this is acceptable
		t.Logf("GCC not found in common locations (this is expected if GCC is not installed): %v", err)
	}
}

// TestVerifyWingetAvailable tests the verifyWingetAvailable function
func TestVerifyWingetAvailable(t *testing.T) {
	err := verifyWingetAvailable()

	// Check if winget is actually available on the system
	_, lookupErr := exec.LookPath("winget")

	if lookupErr != nil {
		// winget not available - function should return error
		if err == nil {
			t.Error("verifyWingetAvailable() returned nil, but winget is not in PATH")
		} else {
			t.Logf("winget not available (expected): %v", err)
		}
	} else {
		// winget is available - function should succeed
		if err != nil {
			t.Errorf("verifyWingetAvailable() returned error, but winget is available: %v", err)
		} else {
			t.Log("winget is available and verified successfully")
		}
	}
}

// TestDetectGCCInstallPath tests the detectGCCInstallPath function
func TestDetectGCCInstallPath(t *testing.T) {
	// This test only runs if GCC is actually installed
	_, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("GCC not installed, skipping detectGCCInstallPath test")
	}

	path, err := detectGCCInstallPath()
	if err != nil {
		t.Errorf("detectGCCInstallPath() failed: %v", err)
		return
	}

	if path == "" {
		t.Error("detectGCCInstallPath() returned empty path")
		return
	}

	// Verify gcc.exe exists at the detected path
	gccExe := filepath.Join(path, "gcc.exe")
	if _, statErr := os.Stat(gccExe); statErr != nil {
		t.Errorf("detectGCCInstallPath() returned path %s, but gcc.exe not found: %v", path, statErr)
	}

	t.Logf("GCC installation path detected: %s", path)
}

// TestUpdatePATHEnvironment tests the updatePATHEnvironment function
func TestUpdatePATHEnvironment(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	tests := []struct {
		name        string
		gccBinPath  string
		initialPATH string
		wantErr     bool
		checkFunc   func(t *testing.T, newPath string)
	}{
		{
			name:        "Add new path to empty PATH",
			gccBinPath:  "C:\\TestGCC\\bin",
			initialPATH: "",
			wantErr:     false,
			checkFunc: func(t *testing.T, newPath string) {
				if !strings.Contains(newPath, "C:\\TestGCC\\bin") {
					t.Errorf("PATH does not contain added GCC path: %s", newPath)
				}
			},
		},
		{
			name:        "Add new path to existing PATH",
			gccBinPath:  "C:\\TestGCC\\bin",
			initialPATH: "C:\\Windows\\System32;C:\\Windows",
			wantErr:     false,
			checkFunc: func(t *testing.T, newPath string) {
				if !strings.HasPrefix(newPath, "C:\\TestGCC\\bin;") {
					t.Errorf("GCC path not prepended to PATH: %s", newPath)
				}
				if !strings.Contains(newPath, "C:\\Windows\\System32") {
					t.Errorf("Original PATH entries lost: %s", newPath)
				}
			},
		},
		{
			name:        "Skip duplicate path",
			gccBinPath:  "C:\\TestGCC\\bin",
			initialPATH: "C:\\TestGCC\\bin;C:\\Windows\\System32",
			wantErr:     false,
			checkFunc: func(t *testing.T, newPath string) {
				// Should not add duplicate
				count := strings.Count(newPath, "C:\\TestGCC\\bin")
				if count != 1 {
					t.Errorf("PATH contains duplicate GCC path (count=%d): %s", count, newPath)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set initial PATH
			os.Setenv("PATH", tt.initialPATH)

			// Call updatePATHEnvironment
			err := updatePATHEnvironment(tt.gccBinPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("updatePATHEnvironment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkFunc != nil {
				newPath := os.Getenv("PATH")
				tt.checkFunc(t, newPath)
			}
		})
	}
}

// TestEnsureGCCAvailable tests the main ensureGCCAvailable orchestration function
func TestEnsureGCCAvailable(t *testing.T) {
	// This is an integration test that tests the full GCC availability check

	// Check current GCC state
	gccInPath := checkGCCInPath()

	t.Logf("Initial GCC state - In PATH: %v", gccInPath)

	// Run ensureGCCAvailable
	err := ensureGCCAvailable()

	if err != nil {
		// If error occurred, check if it's due to missing winget
		_, wingetErr := exec.LookPath("winget")
		if wingetErr != nil {
			t.Logf("ensureGCCAvailable() failed as expected (winget not available): %v", err)
			t.Skip("Skipping test - winget not available for automatic installation")
		} else {
			t.Errorf("ensureGCCAvailable() failed: %v", err)
		}
		return
	}

	// After ensureGCCAvailable, GCC should be available
	if !checkGCCInPath() {
		t.Error("After ensureGCCAvailable(), GCC is still not in PATH")
	} else {
		t.Log("GCC is now available in PATH")
	}
}

// TestGCCInstallationScenarios tests various GCC installation scenarios
func TestGCCInstallationScenarios(t *testing.T) {
	tests := []struct {
		name        string
		description string
		testFunc    func(t *testing.T)
	}{
		{
			name:        "Scenario 1: GCC already installed",
			description: "Should skip installation if GCC is already available",
			testFunc: func(t *testing.T) {
				// Check if GCC is available
				if !checkGCCInPath() {
					gccPath, err := checkGCCInCommonLocations()
					if err != nil {
						t.Skip("GCC not installed, skipping 'already installed' scenario")
					}
					// Add to PATH for this test
					updatePATHEnvironment(gccPath)
				}

				// Now GCC should be in PATH
				if !checkGCCInPath() {
					t.Fatal("Failed to setup test - GCC not in PATH")
				}

				// ensureGCCAvailable should succeed without installation
				err := ensureGCCAvailable()
				if err != nil {
					t.Errorf("ensureGCCAvailable() failed even though GCC is available: %v", err)
				}

				t.Log("Successfully verified GCC availability without installation")
			},
		},
		{
			name:        "Scenario 2: GCC in common location but not in PATH",
			description: "Should find GCC and add to PATH without installation",
			testFunc: func(t *testing.T) {
				// Save original PATH
				originalPath := os.Getenv("PATH")
				defer os.Setenv("PATH", originalPath)

				// Check if GCC exists in common locations
				gccPath, err := checkGCCInCommonLocations()
				if err != nil {
					t.Skip("GCC not in common locations, skipping this scenario")
				}

				// Remove GCC from PATH to simulate scenario
				os.Setenv("PATH", "C:\\Windows\\System32;C:\\Windows")

				// Verify GCC is not in PATH
				if checkGCCInPath() {
					t.Skip("Cannot simulate scenario - GCC still in PATH after removal")
				}

				// ensureGCCAvailable should find it and add to PATH
				err = ensureGCCAvailable()
				if err != nil {
					t.Errorf("ensureGCCAvailable() failed: %v", err)
					return
				}

				// Verify GCC is now in PATH
				if !checkGCCInPath() {
					t.Error("GCC not in PATH after ensureGCCAvailable()")
				}

				// Verify PATH contains the GCC path
				newPath := os.Getenv("PATH")
				if !strings.Contains(newPath, gccPath) {
					t.Errorf("PATH does not contain GCC path %s: %s", gccPath, newPath)
				}

				t.Log("Successfully found GCC in common location and added to PATH")
			},
		},
		{
			name:        "Scenario 3: Winget not available",
			description: "Should fail with clear instructions if winget is not available",
			testFunc: func(t *testing.T) {
				// Check if winget is available
				_, err := exec.LookPath("winget")
				if err == nil {
					t.Skip("winget is available, cannot test 'winget not available' scenario")
				}

				// Check if GCC is available
				if checkGCCInPath() {
					t.Skip("GCC is in PATH, cannot test installation failure scenario")
				}

				gccPath, err := checkGCCInCommonLocations()
				if err == nil {
					t.Skipf("GCC found at %s, cannot test installation failure scenario", gccPath)
				}

				// ensureGCCAvailable should fail with clear error
				err = ensureGCCAvailable()
				if err == nil {
					t.Error("ensureGCCAvailable() should have failed when winget is not available")
					return
				}

				// Verify error message contains helpful information
				errMsg := err.Error()
				if !strings.Contains(errMsg, "winget") {
					t.Errorf("Error message should mention winget: %s", errMsg)
				}

				t.Logf("Correctly failed with error: %v", err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)
			tt.testFunc(t)
		})
	}
}

// TestGCCVersionDetection tests that GCC version can be detected after installation
func TestGCCVersionDetection(t *testing.T) {
	// This test verifies that we can detect GCC version
	_, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("GCC not installed, skipping version detection test")
	}

	cmd := exec.Command("gcc", "--version")
	output, err := cmd.Output()
	if err != nil {
		t.Errorf("Failed to get GCC version: %v", err)
		return
	}

	versionOutput := strings.TrimSpace(string(output))
	if versionOutput == "" {
		t.Error("GCC version output is empty")
		return
	}

	t.Logf("GCC version output:\n%s", versionOutput)

	// Verify output contains "gcc"
	if !strings.Contains(strings.ToLower(versionOutput), "gcc") {
		t.Errorf("GCC version output does not contain 'gcc': %s", versionOutput)
	}
}

// TestPATHUpdatePersistence tests that PATH updates persist for child processes
func TestPATHUpdatePersistence(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	testPath := "C:\\TestGCC\\bin"

	// Update PATH
	err := updatePATHEnvironment(testPath)
	if err != nil {
		t.Fatalf("updatePATHEnvironment() failed: %v", err)
	}

	// Verify PATH is updated in current process
	newPath := os.Getenv("PATH")
	if !strings.Contains(newPath, testPath) {
		t.Errorf("PATH not updated in current process: %s", newPath)
	}

	// Verify PATH is available to child processes
	cmd := exec.Command("cmd", "/c", "echo", "%PATH%")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to execute child process: %v", err)
	}

	childPath := strings.TrimSpace(string(output))
	if !strings.Contains(childPath, testPath) {
		t.Errorf("PATH not inherited by child process: %s", childPath)
	}

	t.Log("PATH update persists to child processes")
}
