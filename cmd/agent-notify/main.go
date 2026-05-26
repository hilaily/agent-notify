package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/longbin/agent-notify/internal/config"
	"github.com/longbin/agent-notify/internal/context"
	"github.com/longbin/agent-notify/internal/hook"
	"github.com/longbin/agent-notify/internal/install"
	"github.com/longbin/agent-notify/internal/logx"
	"github.com/longbin/agent-notify/internal/notify"
	"github.com/longbin/agent-notify/internal/tmux"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	if err := run(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "agent-notify:", err)
		os.Exit(1)
	}
}

func run(cmd string, args []string) error {
	switch cmd {
	case "send":
		return cmdSend(args)
	case "hook":
		return cmdHook(args)
	case "install":
		return cmdInstall(args)
	case "test":
		return cmdTest(args)
	case "doctor":
		return cmdDoctor()
	case "logs":
		return cmdLogs(args)
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		printUsage()
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func cmdSend(args []string) error {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	title := fs.String("title", "", "notification title")
	body := fs.String("body", "", "notification body")
	event := fs.String("event", "stop", "event type")
	_ = fs.Parse(args)
	cfg, err := config.LoadDefault()
	if err != nil {
		return err
	}
	if *title == "" {
		meta := context.MetaFromEnv("Agent", *event)
		*title = context.Render(cfg.Notify.TitleTemplate, meta)
	}
	if *body == "" {
		*body = cfg.BodyForEvent(*event)
	}
	return notify.SendAuto(cfg.Notify.Protocol, *title, *body)
}

func cmdHook(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: agent-notify hook <cursor|claude> <event>")
	}
	agent, event := args[0], args[1]
	cfg, err := config.LoadDefault()
	if err != nil {
		return err
	}
	switch agent {
	case "cursor":
		return hook.RunCursor(os.Stdin, cfg, event, os.Stdout)
	case "claude":
		return hook.RunClaude(os.Stdin, cfg, event, os.Stdout)
	default:
		return fmt.Errorf("unknown agent %q (use cursor stop|response|tool or claude stop|idle)", agent)
	}
}

func cmdInstall(args []string) error {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	all := fs.Bool("all", false, "install all")
	force := fs.Bool("force", false, "overwrite existing hooks")
	_ = fs.Parse(args)
	if *all || len(args) == 0 {
		return install.InstallAll(*force)
	}
	return fmt.Errorf("use --all")
}

func cmdTest(args []string) error {
	mode := "cursor"
	apply := false
	tryAll := false

	for _, arg := range args {
		switch arg {
		case "-v", "--verbose":
			// accepted for compatibility; status always prints to stderr now
		case "--apply":
			apply = true
		case "--try-all":
			tryAll = true
		case "cursor", "claude":
			mode = arg
		case "help", "-h", "--help":
			fmt.Fprint(os.Stderr, `Usage: agent-notify test [cursor|claude] [flags]

Flags:
  --apply     Claude only: emit terminalSequence to terminal
  --try-all   Cursor only: try every delivery method (debug)

Examples:
  agent-notify test cursor
  agent-notify test cursor -v
  agent-notify test cursor --try-all
  agent-notify test claude --apply
`)
			return nil
		default:
			return fmt.Errorf("unknown test argument %q (try: agent-notify test help)", arg)
		}
	}

	switch mode {
	case "cursor":
		_, err := notify.TestCursor("", "", tryAll)
		return err
	case "claude":
		_, err := notify.TestClaude("", "", apply, os.Stdout)
		return err
	default:
		return fmt.Errorf("usage: agent-notify test [cursor|claude] [--apply] [--try-all]")
	}
}

func cmdLogs(args []string) error {
	fs := flag.NewFlagSet("logs", flag.ExitOnError)
	tail := fs.Int("tail", 30, "number of recent lines")
	_ = fs.Parse(args)

	lines, err := logx.Tail(*tail)
	if err != nil {
		return err
	}
	if len(lines) == 0 {
		fmt.Printf("no log entries yet (log file: %s)\n", logx.Path())
		return nil
	}
	fmt.Printf("# %s\n", logx.Path())
	for _, line := range lines {
		fmt.Println(line)
	}
	return nil
}

func cmdDoctor() error {
	fmt.Println("agent-notify doctor")
	if tmux.InTmux() {
		fmt.Println("✓ running inside tmux")
		if tmux.IsSSHSession() {
			fmt.Println("✓ SSH session detected (prefers passthrough delivery)")
		}
		ok, val, err := tmux.AllowPassthroughEnabled()
		if err != nil {
			fmt.Printf("✗ allow-passthrough check failed: %v\n", err)
		} else if ok {
			fmt.Printf("✓ allow-passthrough=%s\n", val)
		} else {
			fmt.Printf("✗ allow-passthrough=%q — add to ~/.tmux.conf: set -g allow-passthrough on\n", val)
		}
		tty, _ := tmux.ClientTTY()
		fmt.Printf("  client_tty=%s\n", tty)
	} else {
		fmt.Println("  not in tmux (direct Ghostty mode)")
	}
	cfgPath := config.DefaultPath()
	fmt.Printf("  config=%s\n", cfgPath)
	fmt.Printf("  hook_log=%s\n", logx.Path())
	return nil
}

func printUsage() {
	fmt.Fprint(os.Stderr, `Usage: agent-notify <command>
Commands:
  send [--title T] [--body B] [--event stop|idle|tool]
  hook cursor stop|response|tool
  hook claude stop|idle
  install [--all] [--force]
  test cursor [--try-all]
  test claude [--apply]
  logs [--tail 30]
  doctor
`)
}
