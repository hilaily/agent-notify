package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/longbin/agent-notify/internal/config"
	"github.com/longbin/agent-notify/internal/inbox"
)

func cmdInbox(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: agent-notify inbox <list|show|done|rm|clear|serve|ssh-config>")
	}
	switch args[0] {
	case "list":
		return inboxList(args[1:], stdout)
	case "show":
		return inboxShow(args[1:], stdout)
	case "done":
		return inboxDone(args[1:], stdout)
	case "rm":
		return inboxRemove(args[1:], stdout)
	case "clear":
		return inboxClear(args[1:], stdout)
	case "serve":
		return inboxServe(args[1:], stderr)
	case "ssh-config":
		return inboxSSHConfig(args[1:], stdout)
	case "tui":
		return inbox.RunTUI(inbox.NewStore(""))
	default:
		return fmt.Errorf("unknown inbox command %q", args[0])
	}
}

func inboxList(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("inbox list", flag.ExitOnError)
	all := fs.Bool("all", false, "include all records")
	done := fs.Bool("done", false, "show done records")
	_ = fs.Parse(args)
	records, err := inbox.NewStore("").List()
	if err != nil {
		return err
	}
	for _, rec := range records {
		if !*all && !*done && rec.Status != inbox.StatusPending {
			continue
		}
		if *done && rec.Status != inbox.StatusDone {
			continue
		}
		fmt.Fprintf(stdout, "%s\t%s\t%s\t%s/%s\t%s\t%s\n",
			rec.ID, rec.Status, rec.Host, rec.Agent, rec.Event, rec.CWD, rec.Title)
	}
	return nil
}

func inboxShow(args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: agent-notify inbox show <id>")
	}
	rec, ok, err := findRecord(args[0])
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("inbox record %q not found", args[0])
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(rec)
}

func inboxDone(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: agent-notify inbox done <id...>")
	}
	n, err := inbox.NewStore("").MarkDone(args)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "marked %d done\n", n)
	return nil
}

func inboxRemove(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: agent-notify inbox rm <id...>")
	}
	n, err := inbox.NewStore("").Remove(args)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "removed %d\n", n)
	return nil
}

func inboxClear(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("inbox clear", flag.ExitOnError)
	done := fs.Bool("done", false, "clear done records only")
	_ = fs.Parse(args)
	var (
		n   int
		err error
	)
	if *done {
		n, err = inbox.NewStore("").ClearDone()
	} else {
		n, err = inbox.NewStore("").ClearAll()
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "cleared %d\n", n)
	return nil
}

func inboxServe(args []string, stderr io.Writer) error {
	cfg, err := config.LoadDefault()
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("inbox serve", flag.ExitOnError)
	socket := fs.String("socket", cfg.Inbox.Socket, "Unix socket path")
	addr := fs.String("addr", "", "TCP address")
	_ = fs.Parse(args)

	handler := inbox.NewHandler(inbox.NewStore(""))
	server := &http.Server{Handler: handler}
	if *addr != "" {
		ln, err := net.Listen("tcp", *addr)
		if err != nil {
			return err
		}
		fmt.Fprintf(stderr, "agent-notify inbox listening on tcp %s\n", *addr)
		return server.Serve(ln)
	}
	if err := os.RemoveAll(*socket); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*socket), 0700); err != nil {
		return err
	}
	ln, err := net.Listen("unix", *socket)
	if err != nil {
		return err
	}
	if err := os.Chmod(*socket, 0600); err != nil {
		ln.Close()
		return err
	}
	fmt.Fprintf(stderr, "agent-notify inbox listening on unix %s\n", *socket)
	return server.Serve(ln)
}

func inboxSSHConfig(args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "install" {
		return fmt.Errorf("usage: agent-notify inbox ssh-config install [--path PATH] [--socket PATH]")
	}
	cfg, err := config.LoadDefault()
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("inbox ssh-config install", flag.ExitOnError)
	path := fs.String("path", inbox.DefaultSSHConfigPath(), "SSH config path")
	socket := fs.String("socket", cfg.Inbox.Socket, "local agent-notify Unix socket path")
	remoteSocket := fs.String("remote-socket", cfg.Inbox.RemoteSocket, "remote agent-notify Unix socket path")
	_ = fs.Parse(args[1:])

	block, err := inbox.InstallSSHConfig(*path, *remoteSocket, *socket)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "wrote SSH config: %s\n\n%s\nReconnect existing SSH sessions for RemoteForward to take effect.\n", *path, block)
	return nil
}

func findRecord(id string) (inbox.Record, bool, error) {
	records, err := inbox.NewStore("").List()
	if err != nil {
		return inbox.Record{}, false, err
	}
	for _, rec := range records {
		if rec.ID == id {
			return rec, true, nil
		}
	}
	return inbox.Record{}, false, nil
}

func uploadTestRecord(cfg config.Config) error {
	rec := inbox.BuildRecord(inbox.BuildInput{
		Agent:  "agent-notify",
		Event:  "test",
		Title:  "agent-notify test",
		Body:   "inbox test",
		Source: inbox.SourceLocal,
	})
	timeout := time.Duration(cfg.Inbox.TimeoutMS) * time.Millisecond
	client := inbox.NewClient(inbox.ClientConfig{Socket: cfg.Inbox.Socket, Addr: cfg.Inbox.Addr, Timeout: timeout})
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return client.Upload(ctx, rec)
}
