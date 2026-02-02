package service

import (
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type darwinManager struct{}

func newPlatformManager() Manager {
	return &darwinManager{}
}

// plist represents a simplified launchd plist structure
type plist struct {
	XMLName xml.Name `xml:"plist"`
	Dict    dict     `xml:"dict"`
}

type dict struct {
	Keys   []string `xml:"key"`
	Values []string `xml:"string"`
}

// Stop stops the service using launchctl
func (m *darwinManager) Stop(serviceName string) error {
	cmd := exec.Command("launchctl", "stop", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop service %s: %w, output: %s", serviceName, err, string(output))
	}
	return nil
}

// Uninstall unloads the service and removes the plist file
func (m *darwinManager) Uninstall(serviceName string) error {
	plistFile := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", serviceName)

	// Unload the service
	cmd := exec.Command("launchctl", "unload", plistFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Log but don't fail if unload fails (service might not be loaded)
		fmt.Printf("Warning: failed to unload service %s: %v, output: %s\n", serviceName, err, string(output))
	}

	// Remove the plist file
	if err := os.Remove(plistFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist file %s: %w", plistFile, err)
	}

	return nil
}

// Install creates a plist file and loads it with launchctl
func (m *darwinManager) Install(serviceName, binaryPath string) error {
	// Create launchd plist file content
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>/var/log/%s.log</string>
	<key>StandardErrorPath</key>
	<string>/var/log/%s.err</string>
</dict>
</plist>
`, serviceName, binaryPath, serviceName, serviceName)

	// Write plist file
	plistFile := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", serviceName)
	if err := os.WriteFile(plistFile, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write plist file %s: %w", plistFile, err)
	}

	// Load the service
	cmd := exec.Command("launchctl", "load", plistFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to load service %s: %w, output: %s", serviceName, err, string(output))
	}

	return nil
}

// Start starts the service using launchctl
func (m *darwinManager) Start(serviceName string) error {
	cmd := exec.Command("launchctl", "start", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start service %s: %w, output: %s", serviceName, err, string(output))
	}
	return nil
}

// IsRunning checks if the service is running using launchctl list
func (m *darwinManager) IsRunning(serviceName string) (bool, error) {
	cmd := exec.Command("launchctl", "list", serviceName)
	output, err := cmd.Output()
	if err != nil {
		// Service is not running or not found
		return false, nil
	}

	// If we get output without error, the service exists
	// Check if it contains a PID (indicating it's running)
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "PID") && !strings.Contains(line, "PID = 0") {
			return true, nil
		}
	}

	return false, nil
}

// GetServiceBinaryPath parses the plist file to extract the binary path
func (m *darwinManager) GetServiceBinaryPath(serviceName string) (string, error) {
	plistFile := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", serviceName)

	data, err := os.ReadFile(plistFile)
	if err != nil {
		return "", fmt.Errorf("failed to read plist file %s: %w", plistFile, err)
	}

	var p plist
	if err := xml.Unmarshal(data, &p); err != nil {
		return "", fmt.Errorf("failed to parse plist file %s: %w", plistFile, err)
	}

	// Find ProgramArguments key and get the first value (the binary path)
	for _, key := range p.Dict.Keys {
		if key == "ProgramArguments" {
			// The value after ProgramArguments should be the binary path
			// In the simplified structure, we need to parse differently
			// Let's use a simpler string parsing approach
			break
		}
	}

	// Fallback to simple string parsing
	content := string(data)
	programArgsStart := strings.Index(content, "<key>ProgramArguments</key>")
	if programArgsStart == -1 {
		return "", fmt.Errorf("ProgramArguments not found in plist file %s", plistFile)
	}

	// Find the first <string> tag after ProgramArguments
	searchStart := programArgsStart + len("<key>ProgramArguments</key>")
	stringStart := strings.Index(content[searchStart:], "<string>")
	if stringStart == -1 {
		return "", fmt.Errorf("binary path not found in plist file %s", plistFile)
	}

	stringStart += searchStart + len("<string>")
	stringEnd := strings.Index(content[stringStart:], "</string>")
	if stringEnd == -1 {
		return "", fmt.Errorf("malformed plist file %s", plistFile)
	}

	binaryPath := content[stringStart : stringStart+stringEnd]
	return binaryPath, nil
}
