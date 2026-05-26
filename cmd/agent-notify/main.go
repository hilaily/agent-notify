package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/longbin/agent-notify/internal/config"
	"github.com/longbin/agent-notify/internal/context"
	"github.com/longbin/agent-notify/internal/hook"
	"github.com/longbin/agent-notify/internal/install"
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
		return cmdTest()
	case "doctor":
		return cmdDoctor()
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
		return fmt.Errorf("unknown agent %q", agent)
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

func cmdTest() error {
	return notify.TestNotification("agent-notify", "测试通知 — 如果你看到这条，说明配置正确")
}

func cmdDoctor() error {
	fmt.Println("agent-notify doctor")
	if tmux.InTmux() {
		fmt.Println("✓ running inside tmux")
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
	return nil
}

func printUsage() {
	fmt.Fprint(os.Stderr, `Usage: agent-notify <command>
Commands:
  send [--title T] [--body B] [--event stop|idle|tool]
  hook cursor stop|tool
  hook claude stop|idle
  install [--all] [--force]
  test
  doctor
`)
}
