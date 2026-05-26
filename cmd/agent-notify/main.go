package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "send", "hook", "install", "test", "doctor", "help", "-h", "--help":
		fmt.Fprintf(os.Stderr, "agent-notify: %s not implemented yet\n", os.Args[1])
		os.Exit(1)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprint(os.Stderr, `Usage: agent-notify <command>
Commands:
  send     Send a notification
  hook     Agent hook entrypoint
  install  Install hook configs
  test     Send test notification
  doctor   Check environment
`)
}
