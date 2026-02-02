package service

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type linuxManager struct{}

func newPlatformManager() Manager {
	return &linuxManager{}
}

// Stop stops the service using systemctl
func (m *linuxManager) Stop(serviceName string) error {
	cmd := exec.Command("systemctl", "stop", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop service %s: %w, output: %s", serviceName, err, string(output))
	}
	return nil
}

// Uninstall disables the service and removes the service file
func (m *linuxManager) Uninstall(serviceName string) error {
	// Disable the service
	cmd := exec.Command("systemctl", "disable", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to disable service %s: %w, output: %s", serviceName, err, string(output))
	}

	// Remove the service file
	serviceFile := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
	if err := os.Remove(serviceFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file %s: %w", serviceFile, err)
	}

	// Reload systemd daemon
	cmd = exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	return nil
}

// Install creates a service file, reloads systemd, and enables the service
func (m *linuxManager) Install(serviceName, binaryPath string) error {
	// Create systemd service file content
	serviceContent := fmt.Sprintf(`[Unit]
Description=SentinelGo Agent
After=network.target

[Service]
Type=simple
ExecStart=%s
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
`, binaryPath)

	// Write service file
	serviceFile := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
	if err := os.WriteFile(serviceFile, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file %s: %w", serviceFile, err)
	}

	// Reload systemd daemon
	cmd := exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Enable the service
	cmd = exec.Command("systemctl", "enable", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to enable service %s: %w, output: %s", serviceName, err, string(output))
	}

	return nil
}

// Start starts the service using systemctl
func (m *linuxManager) Start(serviceName string) error {
	cmd := exec.Command("systemctl", "start", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start service %s: %w, output: %s", serviceName, err, string(output))
	}
	return nil
}

// IsRunning checks if the service is active using systemctl
func (m *linuxManager) IsRunning(serviceName string) (bool, error) {
	cmd := exec.Command("systemctl", "is-active", serviceName)
	output, err := cmd.Output()
	if err != nil {
		// Service is not active, but this is not an error condition
		return false, nil
	}
	return strings.TrimSpace(string(output)) == "active", nil
}

// GetServiceBinaryPath parses the service file to extract the binary path
func (m *linuxManager) GetServiceBinaryPath(serviceName string) (string, error) {
	serviceFile := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)

	file, err := os.Open(serviceFile)
	if err != nil {
		return "", fmt.Errorf("failed to open service file %s: %w", serviceFile, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "ExecStart=") {
			// Extract the binary path from ExecStart=
			binaryPath := strings.TrimPrefix(line, "ExecStart=")
			// Handle potential arguments by taking only the first part
			parts := strings.Fields(binaryPath)
			if len(parts) > 0 {
				return parts[0], nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading service file %s: %w", serviceFile, err)
	}

	return "", fmt.Errorf("ExecStart not found in service file %s", serviceFile)
}
