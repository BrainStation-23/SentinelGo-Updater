package updater

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/BrainStation-23/SentinelGo-Updater/internal/paths"
)

const (
	// MaxLogFileSize is the maximum size of a log file before rotation (10MB)
	MaxLogFileSize = 10 * 1024 * 1024

	// MaxLogFiles is the number of rotated log files to keep
	MaxLogFiles = 5
)

// LogLevel represents the severity of a log message
type LogLevel string

const (
	LogLevelInfo     LogLevel = "INFO"
	LogLevelWarning  LogLevel = "WARNING"
	LogLevelError    LogLevel = "ERROR"
	LogLevelCritical LogLevel = "CRITICAL"
)

var (
	logFile     *os.File
	multiWriter io.Writer
	initialized bool
)

// InitLogger initializes the logging system with file rotation
func InitLogger() error {
	if initialized {
		return nil
	}

	// Ensure data directory exists
	if err := paths.EnsureDataDirectory(); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	logPath := paths.GetUpdaterLogPath()

	// Check if log rotation is needed
	if err := rotateLogIfNeeded(logPath); err != nil {
		return fmt.Errorf("failed to rotate log: %w", err)
	}

	// Open log file for appending
	var err error
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Create multi-writer to write to both file and stderr
	multiWriter = io.MultiWriter(logFile, os.Stderr)

	// Configure standard logger to use our multi-writer
	log.SetOutput(multiWriter)
	log.SetFlags(0) // We'll add our own timestamps and formatting

	initialized = true

	LogInfo("Logging system initialized")
	LogInfo("Log file: %s", logPath)
	LogInfo("Max log file size: %d bytes (%.2f MB)", MaxLogFileSize, float64(MaxLogFileSize)/(1024*1024))
	LogInfo("Max log files to keep: %d", MaxLogFiles)

	return nil
}

// CloseLogger closes the log file
func CloseLogger() error {
	if logFile != nil {
		LogInfo("Closing log file")
		return logFile.Close()
	}
	return nil
}

// rotateLogIfNeeded checks if the log file needs rotation and performs it
func rotateLogIfNeeded(logPath string) error {
	// Check if log file exists
	fileInfo, err := os.Stat(logPath)
	if os.IsNotExist(err) {
		// Log file doesn't exist yet, no rotation needed
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	// Check if file size exceeds limit
	if fileInfo.Size() < MaxLogFileSize {
		// No rotation needed
		return nil
	}

	// Perform rotation
	return rotateLogFiles(logPath)
}

// rotateLogFiles rotates log files, keeping MaxLogFiles versions
func rotateLogFiles(logPath string) error {
	// Close current log file if open
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}

	// Delete the oldest log file if it exists
	oldestLog := fmt.Sprintf("%s.%d", logPath, MaxLogFiles)
	if err := os.Remove(oldestLog); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove oldest log file: %w", err)
	}

	// Rotate existing log files
	for i := MaxLogFiles - 1; i >= 1; i-- {
		oldName := fmt.Sprintf("%s.%d", logPath, i)
		newName := fmt.Sprintf("%s.%d", logPath, i+1)

		if _, err := os.Stat(oldName); err == nil {
			if err := os.Rename(oldName, newName); err != nil {
				return fmt.Errorf("failed to rotate log file %s to %s: %w", oldName, newName, err)
			}
		}
	}

	// Rename current log file to .1
	rotatedName := fmt.Sprintf("%s.1", logPath)
	if err := os.Rename(logPath, rotatedName); err != nil {
		return fmt.Errorf("failed to rotate current log file: %w", err)
	}

	return nil
}

// formatLogMessage formats a log message with timestamp and level
func formatLogMessage(level LogLevel, format string, args ...interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	message := fmt.Sprintf(format, args...)
	return fmt.Sprintf("[%s] [%s] %s", timestamp, level, message)
}

// LogInfo logs an informational message
func LogInfo(format string, args ...interface{}) {
	message := formatLogMessage(LogLevelInfo, format, args...)
	log.Println(message)

	// Check if rotation is needed after each log
	checkAndRotate()
}

// LogWarning logs a warning message
func LogWarning(format string, args ...interface{}) {
	message := formatLogMessage(LogLevelWarning, format, args...)
	log.Println(message)

	checkAndRotate()
}

// LogError logs an error message
func LogError(format string, args ...interface{}) {
	message := formatLogMessage(LogLevelError, format, args...)
	log.Println(message)

	checkAndRotate()
}

// LogCritical logs a critical error message
func LogCritical(format string, args ...interface{}) {
	message := formatLogMessage(LogLevelCritical, format, args...)
	log.Println(message)

	checkAndRotate()
}

// checkAndRotate checks if log rotation is needed and performs it
func checkAndRotate() {
	if !initialized || logFile == nil {
		return
	}

	logPath := paths.GetUpdaterLogPath()

	// Get current file size
	fileInfo, err := os.Stat(logPath)
	if err != nil {
		return
	}

	// Check if rotation is needed
	if fileInfo.Size() >= MaxLogFileSize {
		// Close current file
		logFile.Close()

		// Rotate logs
		if err := rotateLogFiles(logPath); err != nil {
			// Can't log this error since we're in the logging system
			fmt.Fprintf(os.Stderr, "Failed to rotate log files: %v\n", err)
			return
		}

		// Reopen log file
		var err error
		logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to reopen log file after rotation: %v\n", err)
			return
		}

		// Update multi-writer
		multiWriter = io.MultiWriter(logFile, os.Stderr)
		log.SetOutput(multiWriter)

		LogInfo("Log file rotated")
	}
}

// GetLogFilePath returns the current log file path
func GetLogFilePath() string {
	return paths.GetUpdaterLogPath()
}

// GetRotatedLogFiles returns a list of all rotated log files
func GetRotatedLogFiles() []string {
	logPath := paths.GetUpdaterLogPath()
	logDir := filepath.Dir(logPath)
	logBaseName := filepath.Base(logPath)

	var rotatedFiles []string

	for i := 1; i <= MaxLogFiles; i++ {
		rotatedFile := filepath.Join(logDir, fmt.Sprintf("%s.%d", logBaseName, i))
		if _, err := os.Stat(rotatedFile); err == nil {
			rotatedFiles = append(rotatedFiles, rotatedFile)
		}
	}

	return rotatedFiles
}
