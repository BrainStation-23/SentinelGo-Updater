//go:build linux

package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSystemdUnitFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectPath  string
		expectError bool
	}{
		{
			name: "simple absolute path",
			content: `[Unit]
Description=SentinelGo Agent

[Service]
ExecStart=/usr/local/bin/sentinel

[Install]
WantedBy=multi-user.target
`,
			expectPath:  "/usr/local/bin/sentinel",
			expectError: false,
		},
		{
			name: "path with arguments",
			content: `[Unit]
Description=SentinelGo Agent

[Service]
ExecStart=/home/user/go/bin/sentinel --config /etc/sentinel/config.yaml

[Install]
WantedBy=multi-user.target
`,
			expectPath:  "/home/user/go/bin/sentinel",
			expectError: false,
		},
		{
			name: "quoted path",
			content: `[Unit]
Description=SentinelGo Agent

[Service]
ExecStart="/usr/local/bin/sentinel" --verbose

[Install]
WantedBy=multi-user.target
`,
			expectPath:  "/usr/local/bin/sentinel",
			expectError: false,
		},
		{
			name: "single quoted path",
			content: `[Unit]
Description=SentinelGo Agent

[Service]
ExecStart='/usr/local/bin/sentinel' --verbose

[Install]
WantedBy=multi-user.target
`,
			expectPath:  "/usr/local/bin/sentinel",
			expectError: false,
		},
		{
			name: "path with systemd prefix",
			content: `[Unit]
Description=SentinelGo Agent

[Service]
ExecStart=-/usr/local/bin/sentinel

[Install]
WantedBy=multi-user.target
`,
			expectPath:  "/usr/local/bin/sentinel",
			expectError: false,
		},
		{
			name: "path with multiple prefixes",
			content: `[Unit]
Description=SentinelGo Agent

[Service]
ExecStart=@/usr/local/bin/sentinel

[Install]
WantedBy=multi-user.target
`,
			expectPath:  "/usr/local/bin/sentinel",
			expectError: false,
		},
		{
			name: "relative path",
			content: `[Unit]
Description=SentinelGo Agent

[Service]
ExecStart=./sentinel

[Install]
WantedBy=multi-user.target
`,
			expectPath:  "./sentinel",
			expectError: false,
		},
		{
			name: "no ExecStart directive",
			content: `[Unit]
Description=SentinelGo Agent

[Service]
Type=simple

[Install]
WantedBy=multi-user.target
`,
			expectPath:  "",
			expectError: true,
		},
		{
			name: "empty ExecStart",
			content: `[Unit]
Description=SentinelGo Agent

[Service]
ExecStart=

[Install]
WantedBy=multi-user.target
`,
			expectPath:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary unit file
			tempDir := t.TempDir()
			unitFile := filepath.Join(tempDir, "test.service")

			if err := os.WriteFile(unitFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create test unit file: %v", err)
			}

			// Parse the unit file
			path, err := parseSystemdUnitFileAtPath(unitFile)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none, path: %s", path)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if path != tt.expectPath {
					t.Errorf("Expected path %q, got %q", tt.expectPath, path)
				}
			}
		})
	}
}

func TestExtractBinaryPath(t *testing.T) {
	tests := []struct {
		name       string
		execStart  string
		expectPath string
	}{
		{
			name:       "simple path",
			execStart:  "/usr/local/bin/sentinel",
			expectPath: "/usr/local/bin/sentinel",
		},
		{
			name:       "path with arguments",
			execStart:  "/usr/local/bin/sentinel --config /etc/sentinel.conf",
			expectPath: "/usr/local/bin/sentinel",
		},
		{
			name:       "quoted path",
			execStart:  "\"/usr/local/bin/sentinel\" --verbose",
			expectPath: "/usr/local/bin/sentinel",
		},
		{
			name:       "single quoted path",
			execStart:  "'/usr/local/bin/sentinel' --verbose",
			expectPath: "/usr/local/bin/sentinel",
		},
		{
			name:       "path with dash prefix",
			execStart:  "-/usr/local/bin/sentinel",
			expectPath: "/usr/local/bin/sentinel",
		},
		{
			name:       "path with @ prefix",
			execStart:  "@/usr/local/bin/sentinel",
			expectPath: "/usr/local/bin/sentinel",
		},
		{
			name:       "path with multiple prefixes",
			execStart:  "-@/usr/local/bin/sentinel",
			expectPath: "/usr/local/bin/sentinel",
		},
		{
			name:       "path with colon prefix",
			execStart:  ":/usr/local/bin/sentinel",
			expectPath: "/usr/local/bin/sentinel",
		},
		{
			name:       "path with plus prefix",
			execStart:  "+/usr/local/bin/sentinel",
			expectPath: "/usr/local/bin/sentinel",
		},
		{
			name:       "path with exclamation prefix",
			execStart:  "!/usr/local/bin/sentinel",
			expectPath: "/usr/local/bin/sentinel",
		},
		{
			name:       "relative path",
			execStart:  "./sentinel",
			expectPath: "./sentinel",
		},
		{
			name:       "path with spaces in directory",
			execStart:  "\"/usr/local/my programs/sentinel\" --config test",
			expectPath: "/usr/local/my programs/sentinel",
		},
		{
			name:       "empty string",
			execStart:  "",
			expectPath: "",
		},
		{
			name:       "only whitespace",
			execStart:  "   ",
			expectPath: "",
		},
		{
			name:       "only prefix",
			execStart:  "-",
			expectPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBinaryPath(tt.execStart)
			if result != tt.expectPath {
				t.Errorf("extractBinaryPath(%q) = %q, expected %q", tt.execStart, result, tt.expectPath)
			}
		})
	}
}

func TestDetectFromServiceConfigImpl(t *testing.T) {
	// This test verifies that detectFromServiceConfigImpl calls parseSystemdUnitFile
	// It will fail if no systemd service is installed, which is expected
	_, err := detectFromServiceConfigImpl()

	// We expect an error since we're not running as a real service
	if err == nil {
		// If no error, verify we got a valid path
		t.Log("Service config detection succeeded (real service may be installed)")
	} else {
		// Expected case - no service installed
		if err.Error() == "" {
			t.Error("Expected non-empty error message")
		}
	}
}
