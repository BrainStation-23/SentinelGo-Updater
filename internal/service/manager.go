package service

// Manager defines the interface for service management operations
type Manager interface {
	// Stop stops the specified service
	Stop(serviceName string) error

	// Uninstall removes the service from the service manager
	Uninstall(serviceName string) error

	// Install registers the service with the service manager
	Install(serviceName, binaryPath string) error

	// Start starts the specified service
	Start(serviceName string) error

	// IsRunning checks if the service is currently running
	IsRunning(serviceName string) (bool, error)

	// GetServiceBinaryPath returns the path to the service binary
	GetServiceBinaryPath(serviceName string) (string, error)
}

// NewManager creates a platform-specific service manager
func NewManager() Manager {
	return newPlatformManager()
}
