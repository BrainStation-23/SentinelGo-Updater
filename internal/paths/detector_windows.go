//go:build windows

package paths

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/mgr"
)

// detectFromServiceConfigImpl implements Windows-specific service configuration detection
// It queries the Windows Service Manager to extract the binary path from ImagePath
func detectFromServiceConfigImpl() (string, error) {
	return queryWindowsService("sentinelgo")
}

// detectFromRunningProcessImpl implements Windows-specific running process detection
// It uses Windows API to enumerate processes and find the sentinel process
func detectFromRunningProcessImpl() (string, error) {
	return findRunningProcessWindows("sentinel.exe")
}

// findRunningProcessWindows searches for a running process by name using Windows API
func findRunningProcessWindows(processName string) (string, error) {
	// Take a snapshot of all processes
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return "", fmt.Errorf("failed to create process snapshot: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	// Prepare the process entry structure
	var procEntry windows.ProcessEntry32
	procEntry.Size = uint32(unsafe.Sizeof(procEntry))

	// Get the first process
	err = windows.Process32First(snapshot, &procEntry)
	if err != nil {
		return "", fmt.Errorf("failed to get first process: %w", err)
	}

	// Iterate through all processes
	for {
		// Convert the process name from [260]uint16 to string
		exeFile := syscall.UTF16ToString(procEntry.ExeFile[:])

		// Check if this is the process we're looking for
		if strings.EqualFold(exeFile, processName) {
			// Found the process, now get its full path
			binaryPath, err := getProcessImagePath(procEntry.ProcessID)
			if err != nil {
				// Try next process if we can't get the path
				goto nextProcess
			}
			return binaryPath, nil
		}

	nextProcess:
		// Get the next process
		err = windows.Process32Next(snapshot, &procEntry)
		if err != nil {
			// No more processes
			break
		}
	}

	return "", fmt.Errorf("process %s not found", processName)
}

// getProcessImagePath retrieves the full path of a process executable by PID
func getProcessImagePath(pid uint32) (string, error) {
	// Open the process with QUERY_LIMITED_INFORMATION access
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", fmt.Errorf("failed to open process %d: %w", pid, err)
	}
	defer windows.CloseHandle(handle)

	// Query the full image path
	var pathBuf [windows.MAX_PATH]uint16
	size := uint32(len(pathBuf))

	// Use QueryFullProcessImageName to get the full path
	err = windows.QueryFullProcessImageName(handle, 0, &pathBuf[0], &size)
	if err != nil {
		return "", fmt.Errorf("failed to query process image name: %w", err)
	}

	// Convert to Go string
	return syscall.UTF16ToString(pathBuf[:size]), nil
}

// queryWindowsService queries the Windows Service Manager for the service executable path
// It handles quoted paths, UNC paths, and paths with arguments
func queryWindowsService(serviceName string) (string, error) {
	// Connect to the Windows Service Manager
	m, err := mgr.Connect()
	if err != nil {
		return "", fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Open the service
	s, err := m.OpenService(serviceName)
	if err != nil {
		return "", fmt.Errorf("failed to open service %s: %w", serviceName, err)
	}
	defer s.Close()

	// Get the service configuration
	config, err := s.Config()
	if err != nil {
		return "", fmt.Errorf("failed to get service config for %s: %w", serviceName, err)
	}

	// Extract binary path from BinaryPathName (ImagePath)
	binaryPath := extractWindowsBinaryPath(config.BinaryPathName)
	if binaryPath == "" {
		return "", fmt.Errorf("failed to extract binary path from ImagePath: %s", config.BinaryPathName)
	}

	return binaryPath, nil
}

// extractWindowsBinaryPath extracts the binary path from a Windows service ImagePath
// Handles quoted paths, UNC paths, and paths with arguments
//
// Examples:
//   - "C:\Program Files\SentinelGo\sentinel.exe" arg1 arg2
//   - C:\SentinelGo\sentinel.exe arg1 arg2
//   - \\server\share\sentinel.exe arg1 arg2
//   - "\\server\share\sentinel.exe" arg1 arg2
func extractWindowsBinaryPath(imagePath string) string {
	imagePath = strings.TrimSpace(imagePath)
	if imagePath == "" {
		return ""
	}

	// Handle quoted paths: "C:\Program Files\SentinelGo\sentinel.exe" arg1 arg2
	if strings.HasPrefix(imagePath, "\"") {
		// Find the closing quote
		endQuote := strings.Index(imagePath[1:], "\"")
		if endQuote > 0 {
			return imagePath[1 : endQuote+1]
		}
		// If no closing quote found, treat the whole string as the path
		return strings.TrimPrefix(imagePath, "\"")
	}

	// Handle unquoted paths with arguments
	// For UNC paths (\\server\share\...), we need to be careful with splitting
	// For regular paths (C:\...), split by space

	// Check if it's a UNC path
	if strings.HasPrefix(imagePath, "\\\\") {
		// UNC path: \\server\share\path\to\sentinel.exe arg1 arg2
		// We need to find the first space that's not part of the path
		// This is tricky because UNC paths can have spaces in them
		// We'll use a heuristic: look for .exe followed by a space
		exeIndex := strings.Index(strings.ToLower(imagePath), ".exe")
		if exeIndex > 0 {
			// Check if there's a space after .exe
			afterExe := imagePath[exeIndex+4:]
			if len(afterExe) == 0 || afterExe[0] == ' ' {
				return imagePath[:exeIndex+4]
			}
		}
		// If no .exe found or no space after it, take the whole path
		parts := strings.Fields(imagePath)
		if len(parts) > 0 {
			return parts[0]
		}
		return imagePath
	}

	// Regular path (C:\...) - split by whitespace and take first token
	parts := strings.Fields(imagePath)
	if len(parts) > 0 {
		return parts[0]
	}

	return imagePath
}
