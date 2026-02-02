//go:build darwin

package paths

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"howett.net/plist"
)

// detectFromServiceConfigImpl implements macOS-specific service configuration detection
// It parses the launchd plist file to extract the binary path from ProgramArguments
func detectFromServiceConfigImpl() (string, error) {
	return parseLaunchdPlist("sentinelgo")
}

// detectFromRunningProcessImpl implements macOS-specific running process detection
// It uses the ps command to find running sentinel processes
func detectFromRunningProcessImpl() (string, error) {
	return findRunningProcessDarwin("sentinel")
}

// findRunningProcessDarwin searches for a running process by name using ps command
func findRunningProcessDarwin(processName string) (string, error) {
	// Use ps to list all processes with full command line
	// ps -ax -o pid,command lists all processes with their full command
	cmd := fmt.Sprintf("ps -ax -o command | grep -E '/%s(\\s|$)' | grep -v grep | head -n 1", processName)

	// Execute the command
	output, err := executeCommand("sh", "-c", cmd)
	if err != nil {
		return "", fmt.Errorf("failed to execute ps command: %w", err)
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return "", fmt.Errorf("process %s not found", processName)
	}

	// Extract the binary path from the command line
	// The output is the full command line, we need the first token
	binaryPath := extractBinaryPathFromCommandLine(output)
	if binaryPath == "" {
		return "", fmt.Errorf("failed to extract binary path from ps output: %s", output)
	}

	return binaryPath, nil
}

// extractBinaryPathFromCommandLine extracts the binary path from a command line string
func extractBinaryPathFromCommandLine(cmdLine string) string {
	cmdLine = strings.TrimSpace(cmdLine)
	if cmdLine == "" {
		return ""
	}

	// Handle quoted paths
	if strings.HasPrefix(cmdLine, "\"") {
		endQuote := strings.Index(cmdLine[1:], "\"")
		if endQuote > 0 {
			return cmdLine[1 : endQuote+1]
		}
	}

	if strings.HasPrefix(cmdLine, "'") {
		endQuote := strings.Index(cmdLine[1:], "'")
		if endQuote > 0 {
			return cmdLine[1 : endQuote+1]
		}
	}

	// Unquoted path - take first token
	parts := strings.Fields(cmdLine)
	if len(parts) > 0 {
		return parts[0]
	}

	return ""
}

// executeCommand executes a shell command and returns its output
func executeCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command failed: %w", err)
	}
	return string(output), nil
}

// parseLaunchdPlist reads a launchd plist file and extracts the binary path from ProgramArguments
// It handles both system-level and user-level LaunchDaemons/LaunchAgents
func parseLaunchdPlist(serviceName string) (string, error) {
	// Try common launchd plist locations
	plistPaths := []string{
		fmt.Sprintf("/Library/LaunchDaemons/com.%s.plist", serviceName),
		fmt.Sprintf("/Library/LaunchAgents/com.%s.plist", serviceName),
		fmt.Sprintf("/System/Library/LaunchDaemons/com.%s.plist", serviceName),
		fmt.Sprintf("/System/Library/LaunchAgents/com.%s.plist", serviceName),
	}

	// Also check user-specific locations if HOME is set
	if home := os.Getenv("HOME"); home != "" {
		plistPaths = append(plistPaths,
			filepath.Join(home, "Library", "LaunchAgents", fmt.Sprintf("com.%s.plist", serviceName)),
		)
	}

	var lastErr error
	for _, plistPath := range plistPaths {
		path, err := parseLaunchdPlistAtPath(plistPath)
		if err == nil {
			return path, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return "", fmt.Errorf("failed to parse launchd plist: %w", lastErr)
	}
	return "", fmt.Errorf("launchd plist not found for service %s", serviceName)
}

// parseLaunchdPlistAtPath reads and parses a specific launchd plist file
func parseLaunchdPlistAtPath(plistPath string) (string, error) {
	data, err := os.ReadFile(plistPath)
	if err != nil {
		return "", fmt.Errorf("failed to read plist file %s: %w", plistPath, err)
	}

	// Parse the plist file
	var plistData map[string]interface{}
	_, err = plist.Unmarshal(data, &plistData)
	if err != nil {
		return "", fmt.Errorf("failed to parse plist file %s: %w", plistPath, err)
	}

	// Extract ProgramArguments array
	programArgs, ok := plistData["ProgramArguments"]
	if !ok {
		// Try Program key as fallback (older plist format)
		if program, ok := plistData["Program"]; ok {
			if programStr, ok := program.(string); ok && programStr != "" {
				return programStr, nil
			}
		}
		return "", fmt.Errorf("ProgramArguments not found in plist file %s", plistPath)
	}

	// ProgramArguments should be an array
	argsArray, ok := programArgs.([]interface{})
	if !ok {
		return "", fmt.Errorf("ProgramArguments is not an array in plist file %s", plistPath)
	}

	// Extract the first element (binary path)
	if len(argsArray) == 0 {
		return "", fmt.Errorf("ProgramArguments array is empty in plist file %s", plistPath)
	}

	binaryPath, ok := argsArray[0].(string)
	if !ok {
		return "", fmt.Errorf("first element of ProgramArguments is not a string in plist file %s", plistPath)
	}

	if binaryPath == "" {
		return "", fmt.Errorf("binary path is empty in plist file %s", plistPath)
	}

	return binaryPath, nil
}
