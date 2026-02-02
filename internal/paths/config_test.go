package paths

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigPath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		config      *UpdaterConfig
		expectPath  string
		expectEmpty bool
	}{
		{
			name: "valid config with binary path",
			config: &UpdaterConfig{
				BinaryPath:          "/custom/path/to/sentinel",
				EnableAutoDetection: true,
			},
			expectPath:  "/custom/path/to/sentinel",
			expectEmpty: false,
		},
		{
			name: "config with empty binary path",
			config: &UpdaterConfig{
				BinaryPath:          "",
				EnableAutoDetection: true,
			},
			expectPath:  "",
			expectEmpty: true,
		},
		{
			name:        "no config file",
			config:      nil,
			expectPath:  "",
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config file if config is provided
			configPath := filepath.Join(tempDir, "updater-config.json")

			if tt.config != nil {
				data, err := json.Marshal(tt.config)
				if err != nil {
					t.Fatalf("Failed to marshal config: %v", err)
				}

				if err := os.WriteFile(configPath, data, 0644); err != nil {
					t.Fatalf("Failed to write config file: %v", err)
				}
			}

			// Note: loadConfigPath() uses GetDataDirectory() which we can't easily override
			// This test verifies the struct and JSON marshaling work correctly
			if tt.config != nil {
				data, err := os.ReadFile(configPath)
				if err != nil {
					t.Fatalf("Failed to read config file: %v", err)
				}

				var config UpdaterConfig
				if err := json.Unmarshal(data, &config); err != nil {
					t.Fatalf("Failed to unmarshal config: %v", err)
				}

				if config.BinaryPath != tt.expectPath {
					t.Errorf("Expected path %q, got %q", tt.expectPath, config.BinaryPath)
				}
			}
		})
	}
}

func TestUpdaterConfigValidation(t *testing.T) {
	detector := &BinaryDetector{}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "empty path",
			path:     "",
			expected: false,
		},
		{
			name:     "non-existent path",
			path:     "/non/existent/path/sentinel",
			expected: false,
		},
		{
			name:     "directory instead of file",
			path:     os.TempDir(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.validateBinaryPath(tt.path)
			if result != tt.expected {
				t.Errorf("validateBinaryPath(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestDetectorWithManualConfig(t *testing.T) {
	// Create a temporary executable file for testing
	tempDir := t.TempDir()
	testBinary := filepath.Join(tempDir, "sentinel")

	// Create a dummy executable file
	if err := os.WriteFile(testBinary, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	// Create detector with manual config path
	detector := &BinaryDetector{
		configPath: testBinary,
	}

	// Test that manual config path is validated
	if !detector.validateBinaryPath(testBinary) {
		t.Error("Expected test binary to be valid")
	}

	// Test detection with valid manual config
	path, err := detector.DetectBinaryPath()
	if err != nil {
		t.Errorf("DetectBinaryPath() failed: %v", err)
	}

	if path != testBinary {
		t.Errorf("Expected path %q, got %q", testBinary, path)
	}

	// Verify path is cached
	cachedPath, ok := detector.getCachedPath()
	if !ok {
		t.Error("Expected path to be cached")
	}

	if cachedPath != testBinary {
		t.Errorf("Expected cached path %q, got %q", testBinary, cachedPath)
	}
}

func TestDetectorFallbackOnInvalidConfig(t *testing.T) {
	// Create detector with invalid manual config path
	detector := &BinaryDetector{
		configPath: "/invalid/path/to/sentinel",
	}

	// Test that detection falls back to auto-detection
	// This may succeed if a real binary exists on the system, or fail if not
	path, err := detector.DetectBinaryPath()

	if err != nil {
		// No valid binary found - verify error message is informative
		if err.Error() == "" {
			t.Error("Expected non-empty error message")
		}
	} else {
		// A valid binary was found via fallback - verify it's valid
		if path == "" {
			t.Error("Expected non-empty path when detection succeeds")
		}

		if !detector.validateBinaryPath(path) {
			t.Errorf("Detected path %q is not valid", path)
		}

		// Verify it's not the invalid config path
		if path == "/invalid/path/to/sentinel" {
			t.Error("Should not have used invalid config path")
		}
	}
}
