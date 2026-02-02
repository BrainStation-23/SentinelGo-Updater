package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type windowsManager struct{}

func newPlatformManager() Manager {
	return &windowsManager{}
}

// Stop stops the service using sc.exe
func (m *windowsManager) Stop(serviceName string) error {
	cmd := exec.Command("sc.exe", "stop", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		// If service doesn't exist, that's okay - it's already "stopped"
		if strings.Contains(outputStr, "1060") || strings.Contains(outputStr, "does not exist") {
			return nil
		}
		return fmt.Errorf("failed to stop service %s: %w, output: %s", serviceName, err, outputStr)
	}
	return nil
}

// Uninstall removes the service using sc.exe delete
func (m *windowsManager) Uninstall(serviceName string) error {
	cmd := exec.Command("sc.exe", "delete", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		// If service doesn't exist, that's okay - it's already uninstalled
		if strings.Contains(outputStr, "1060") || strings.Contains(outputStr, "does not exist") {
			return nil
		}
		return fmt.Errorf("failed to delete service %s: %w, output: %s", serviceName, err, outputStr)
	}
	return nil
}

// Install creates the service using sc.exe create
func (m *windowsManager) Install(serviceName, binaryPath string) error {
	// Get the system PATH to include in service environment
	systemPath := getSystemPATH()

	// Create the service with sc.exe
	// sc.exe create <serviceName> binPath="<binaryPath>" start=auto
	// NOTE: No space after the equals sign - sc.exe is strict about syntax
	cmd := exec.Command("sc.exe", "create", serviceName,
		fmt.Sprintf("binPath=\"%s\"", binaryPath),
		"start=auto",
		"DisplayName=SentinelGo Agent",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create service %s: %w, output: %s", serviceName, err, string(output))
	}

	// Configure service environment to include system PATH
	// This ensures the service can find GCC and other system tools
	if systemPath != "" {
		// Use reg.exe to set the Environment value for the service
		// This makes PATH available to the service process
		regKey := fmt.Sprintf("HKLM\\SYSTEM\\CurrentControlSet\\Services\\%s", serviceName)
		regCmd := exec.Command("reg.exe", "add", regKey,
			"/v", "Environment",
			"/t", "REG_MULTI_SZ",
			"/d", fmt.Sprintf("PATH=%s", systemPath),
			"/f",
		)
		if err := regCmd.Run(); err != nil {
			// Log warning but don't fail installation
			fmt.Printf("Warning: failed to set service environment PATH: %v\n", err)
		} else {
			fmt.Printf("Service environment configured with system PATH\n")
		}
	}

	// Configure service to restart on failure
	cmd = exec.Command("sc.exe", "failure", serviceName,
		"reset=86400",
		"actions=restart/60000/restart/60000/restart/60000",
	)
	if err := cmd.Run(); err != nil {
		// Log warning but don't fail installation
		fmt.Printf("Warning: failed to configure service failure actions: %v\n", err)
	}

	return nil
}

// Start starts the service using sc.exe
func (m *windowsManager) Start(serviceName string) error {
	cmd := exec.Command("sc.exe", "start", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start service %s: %w, output: %s", serviceName, err, string(output))
	}
	return nil
}

// IsRunning checks if the service is running by parsing sc.exe query output
func (m *windowsManager) IsRunning(serviceName string) (bool, error) {
	cmd := exec.Command("sc.exe", "query", serviceName)
	output, err := cmd.Output()
	if err != nil {
		// Service not found or error querying
		return false, nil
	}

	// Parse the output to find STATE line
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "STATE") {
			// STATE line format: "STATE              : 4  RUNNING"
			if strings.Contains(line, "RUNNING") {
				return true, nil
			}
			return false, nil
		}
	}

	return false, nil
}

// GetServiceBinaryPath queries the service configuration and parses BINARY_PATH_NAME
func (m *windowsManager) GetServiceBinaryPath(serviceName string) (string, error) {
	cmd := exec.Command("sc.exe", "qc", serviceName)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to query service %s: %w", serviceName, err)
	}

	// Parse the output to find BINARY_PATH_NAME line
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "BINARY_PATH_NAME") {
			// BINARY_PATH_NAME line format: "BINARY_PATH_NAME   : C:\Path\To\Binary.exe"
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				binaryPath := strings.TrimSpace(parts[1])
				// Remove quotes if present
				binaryPath = strings.Trim(binaryPath, "\"")
				return binaryPath, nil
			}
		}
	}

	return "", fmt.Errorf("BINARY_PATH_NAME not found for service %s", serviceName)
}

// getSystemPATH retrieves the system PATH from registry and current environment
func getSystemPATH() string {
	// Try to get the system PATH from registry first
	cmd := exec.Command("reg.exe", "query",
		"HKLM\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment",
		"/v", "Path",
	)
	output, err := cmd.Output()
	if err == nil {
		// Parse the registry output
		outputStr := string(output)
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Path") && strings.Contains(line, "REG_") {
				// Line format: "Path    REG_EXPAND_SZ    C:\Windows\system32;..."
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					// Join everything after the type field
					systemPath := strings.Join(parts[2:], " ")
					return systemPath
				}
			}
		}
	}

	// Fallback to current process PATH
	// This includes both system and user PATH
	return os.Getenv("PATH")
}
