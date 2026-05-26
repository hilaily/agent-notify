package main

import "fmt"

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func cmdVersion() error {
	fmt.Printf("agent-notify %s\n", version)
	fmt.Printf("commit: %s\n", commit)
	fmt.Printf("built: %s\n", buildDate)
	return nil
}
