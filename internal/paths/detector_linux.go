//go:build linux

package paths

import (
	"fmt"
	"os"
	"strings"
)

// detectFromServiceConfigImpl implements Linux-specific service configuration detection
// It parses the systemd unit file to extract the binary path from ExecStart directive
func detectFromServiceConfigImpl() (string, error) {
	return parseSystemdUnitFile("sentinelgo")
}

// detectFromRunningProcessImpl implements Linux-specific running process detection
// It searches /proc filesystem for running sentinel processes
func detectFromRunningProcessImpl() (string, error) {
	return findRunningProcessLinux("sentinel")
}

// findRunningProcessLinux searches for a running process by name using /proc filesystem
func findRunningProcessLinux(processName string) (string, error) {
	// Read /proc directory
	procDir, err := os.ReadDir("/proc")
	if err != nil {
		return "", fmt.Errorf("failed to read /proc directory: %w", err)
	}

	// Iterate through /proc entries (PIDs)
	for _, entry := range procDir {
		// Skip non-directory entries
		if !entry.IsDir() {
			continue
		}

		// Skip non-numeric directory names (only PIDs are numeric)
		pid := entry.Name()
		if len(pid) == 0 || pid[0] < '0' || pid[0] > '9' {
			continue
		}

		// Read the exe symlink to get the binary path
		exePath := fmt.Sprintf("/proc/%s/exe", pid)
		binaryPath, err := os.Readlink(exePath)
		if err != nil {
			// Process may have exited or we don't have permission
			continue
		}

		// Check if this is the process we're looking for
		// Extract the base name from the path
		baseName := binaryPath
		if lastSlash := strings.LastIndex(binaryPath, "/"); lastSlash >= 0 {
			baseName = binaryPath[lastSlash+1:]
		}

		// Match the process name
		if baseName == processName {
			return binaryPath, nil
		}
	}

	return "", fmt.Errorf("process %s not found in /proc", processName)
}

// parseSystemdUnitFile reads a systemd unit file and extracts the binary path from ExecStart
// It handles both absolute and relative paths, quoted and unquoted formats
func parseSystemdUnitFile(serviceName string) (string, error) {
	// Try common systemd unit file locations
	unitFilePaths := []string{
		fmt.Sprintf("/etc/systemd/system/%s.service", serviceName),
		fmt.Sprintf("/lib/systemd/system/%s.service", serviceName),
		fmt.Sprintf("/usr/lib/systemd/system/%s.service", serviceName),
	}

	var lastErr error
	for _, unitFile := range unitFilePaths {
		path, err := parseSystemdUnitFileAtPath(unitFile)
		if err == nil {
			return path, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return "", fmt.Errorf("failed to parse systemd unit file: %w", lastErr)
	}
	return "", fmt.Errorf("systemd unit file not found for service %s", serviceName)
}

// parseSystemdUnitFileAtPath reads and parses a specific systemd unit file
func parseSystemdUnitFileAtPath(unitFile string) (string, error) {
	data, err := os.ReadFile(unitFile)
	if err != nil {
		return "", fmt.Errorf("failed to read unit file %s: %w", unitFile, err)
	}

	// Parse the INI-style systemd unit file
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for ExecStart directive
		if strings.HasPrefix(line, "ExecStart=") {
			execStart := strings.TrimPrefix(line, "ExecStart=")
			execStart = strings.TrimSpace(execStart)

			if execStart == "" {
				continue
			}

			// Extract the binary path from ExecStart
			binaryPath := extractBinaryPath(execStart)
			if binaryPath != "" {
				return binaryPath, nil
			}
		}
	}

	return "", fmt.Errorf("ExecStart directive not found in unit file %s", unitFile)
}

// extractBinaryPath extracts the binary path from an ExecStart directive value
// Handles quoted paths, unquoted paths, paths with arguments, and special prefixes
func extractBinaryPath(execStart string) string {
	// Handle systemd special prefixes (-, @, :, +, !, etc.)
	// These can appear before the path: ExecStart=-/path/to/binary
	execStart = strings.TrimLeft(execStart, "-@:+!")
	execStart = strings.TrimSpace(execStart)

	if execStart == "" {
		return ""
	}

	// Handle quoted paths: "/path/to/binary" arg1 arg2
	if strings.HasPrefix(execStart, "\"") {
		endQuote := strings.Index(execStart[1:], "\"")
		if endQuote > 0 {
			return execStart[1 : endQuote+1]
		}
	}

	// Handle single-quoted paths: '/path/to/binary' arg1 arg2
	if strings.HasPrefix(execStart, "'") {
		endQuote := strings.Index(execStart[1:], "'")
		if endQuote > 0 {
			return execStart[1 : endQuote+1]
		}
	}

	// Handle unquoted paths: /path/to/binary arg1 arg2
	// Split by whitespace and take the first token
	parts := strings.Fields(execStart)
	if len(parts) > 0 {
		return parts[0]
	}

	return ""
}
