package main

import (
	"fmt"
	"runtime"

	"github.com/BrainStation-23/SentinelGo-Updater/internal/paths"
)

func main() {
	fmt.Println("=== SentinelGo Path Verification ===")
	fmt.Printf("Operating System: %s\n\n", runtime.GOOS)

	fmt.Println("Data Directory:")
	fmt.Printf("  %s\n\n", paths.GetDataDirectory())

	fmt.Println("Database Path:")
	fmt.Printf("  %s\n\n", paths.GetDatabasePath())

	fmt.Println("Updater Log Path:")
	fmt.Printf("  %s\n\n", paths.GetUpdaterLogPath())

	fmt.Println("Agent Log Path:")
	fmt.Printf("  %s\n\n", paths.GetAgentLogPath())

	fmt.Println("Binary Directory:")
	fmt.Printf("  %s\n\n", paths.GetBinaryDirectory())

	fmt.Println("Main Agent Binary Path:")
	fmt.Printf("  %s\n", paths.GetMainAgentBinaryPath())
}
