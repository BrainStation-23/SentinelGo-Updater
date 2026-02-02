package main

import (
	"fmt"
	"log"
	"os"

	"github.com/BrainStation-23/SentinelGo-Updater/internal/updater"
	"github.com/kardianos/service"
)

var (
	// Version of the updater service (can be overridden at build time with -ldflags)
	Version = "dev"
	// BuildTime is the time when the binary was built
	BuildTime = "unknown"
	// GitCommit is the git commit hash
	GitCommit = "unknown"
)

// updaterProgram implements the service.Interface
type updaterProgram struct {
	exit chan struct{}
}

// Start is called when the service starts
func (p *updaterProgram) Start(s service.Service) error {
	// Start the updater in a goroutine
	p.exit = make(chan struct{})
	go p.run()
	return nil
}

// run executes the main updater logic
func (p *updaterProgram) run() {
	// Run the updater loop
	updater.Run()
}

// Stop is called when the service stops
func (p *updaterProgram) Stop(s service.Service) error {
	// Signal the updater to stop
	close(p.exit)
	return nil
}

func main() {
	// Service configuration
	svcConfig := &service.Config{
		Name:        "sentinelgo-updater",
		DisplayName: "SentinelGo Updater Service",
		Description: "Manages updates for SentinelGo Agent",
	}

	prg := &updaterProgram{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Handle command-line arguments
	if len(os.Args) > 1 {
		command := os.Args[1]

		// Handle --version flag
		if command == "--version" || command == "-v" {
			fmt.Printf("sentinelgo-updater version %s\n", Version)
			fmt.Printf("Build time: %s\n", BuildTime)
			fmt.Printf("Git commit: %s\n", GitCommit)
			return
		}

		// Handle service control commands
		switch command {
		case "install":
			err = s.Install()
			if err != nil {
				log.Fatalf("Failed to install service: %v", err)
			}
			fmt.Println("Service installed successfully")
			fmt.Println("Run 'sentinel-updater start' to start the service")
			return

		case "uninstall":
			err = s.Uninstall()
			if err != nil {
				log.Fatalf("Failed to uninstall service: %v", err)
			}
			fmt.Println("Service uninstalled successfully")
			return

		case "start":
			err = s.Start()
			if err != nil {
				log.Fatalf("Failed to start service: %v", err)
			}
			fmt.Println("Service started successfully")
			return

		case "stop":
			err = s.Stop()
			if err != nil {
				log.Fatalf("Failed to stop service: %v", err)
			}
			fmt.Println("Service stopped successfully")
			return

		case "restart":
			err = s.Restart()
			if err != nil {
				log.Fatalf("Failed to restart service: %v", err)
			}
			fmt.Println("Service restarted successfully")
			return

		default:
			fmt.Printf("Unknown command: %s\n", command)
			fmt.Println("\nUsage:")
			fmt.Println("  sentinel-updater install    - Install the updater service")
			fmt.Println("  sentinel-updater uninstall  - Uninstall the updater service")
			fmt.Println("  sentinel-updater start      - Start the updater service")
			fmt.Println("  sentinel-updater stop       - Stop the updater service")
			fmt.Println("  sentinel-updater restart    - Restart the updater service")
			fmt.Println("  sentinel-updater --version  - Show version information")
			os.Exit(1)
		}
	}

	// No command specified, run as service
	logger, err := s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}

	err = s.Run()
	if err != nil {
		logger.Error(err)
	}
}
