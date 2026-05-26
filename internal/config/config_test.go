package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	t.Setenv("USER", "example")
	cfg := Default()
	if !cfg.Events.Stop {
		t.Fatal("expected stop=true by default")
	}
	if cfg.Events.Idle {
		t.Fatal("expected idle=false by default")
	}
	if cfg.Notify.Protocol != "osc777" {
		t.Fatalf("expected osc777, got %q", cfg.Notify.Protocol)
	}
	if !cfg.Inbox.Enabled {
		t.Fatal("expected inbox enabled by default")
	}
	if cfg.Inbox.Addr != "127.0.0.1:17777" {
		t.Fatalf("expected default inbox addr, got %q", cfg.Inbox.Addr)
	}
	if cfg.Inbox.Socket == "" {
		t.Fatal("expected default inbox socket")
	}
	if cfg.Inbox.RemoteSocket != "/tmp/agent-notify-example.sock" {
		t.Fatalf("expected default remote socket, got %q", cfg.Inbox.RemoteSocket)
	}
	if !cfg.Inbox.FallbackLocal {
		t.Fatal("expected local fallback enabled by default")
	}
	if cfg.Inbox.TimeoutMS != 500 {
		t.Fatalf("expected 500ms timeout, got %d", cfg.Inbox.TimeoutMS)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[events]
stop = false
tool = true

[notify]
body_stop = "custom stop"

[inbox]
enabled = false
socket = "/tmp/custom-agent-notify.sock"
remote_socket = "/tmp/custom-remote-agent-notify.sock"
addr = "127.0.0.1:18888"
fallback_local = false
timeout_ms = 250
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Events.Stop {
		t.Fatal("expected stop=false")
	}
	if !cfg.Events.Tool {
		t.Fatal("expected tool=true")
	}
	if cfg.Notify.BodyStop != "custom stop" {
		t.Fatalf("got %q", cfg.Notify.BodyStop)
	}
	if cfg.Inbox.Enabled {
		t.Fatal("expected inbox disabled")
	}
	if cfg.Inbox.Socket != "/tmp/custom-agent-notify.sock" {
		t.Fatalf("got inbox socket %q", cfg.Inbox.Socket)
	}
	if cfg.Inbox.RemoteSocket != "/tmp/custom-remote-agent-notify.sock" {
		t.Fatalf("got inbox remote socket %q", cfg.Inbox.RemoteSocket)
	}
	if cfg.Inbox.Addr != "127.0.0.1:18888" {
		t.Fatalf("got inbox addr %q", cfg.Inbox.Addr)
	}
	if cfg.Inbox.FallbackLocal {
		t.Fatal("expected fallback_local=false")
	}
	if cfg.Inbox.TimeoutMS != 250 {
		t.Fatalf("got timeout %d", cfg.Inbox.TimeoutMS)
	}
}

func TestDefaultInboxSocketPrefersXDGRuntimeDir(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/1234")
	t.Setenv("HOME", "/home/example")

	got := DefaultInboxSocket()
	want := "/run/user/1234/agent-notify.sock"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEventEnabled(t *testing.T) {
	cfg := Default()
	if !cfg.EventEnabled("stop") {
		t.Fatal("expected stop enabled by default")
	}
	if !cfg.EventEnabled("STOP") {
		t.Fatal("expected case-insensitive match")
	}
	cfg.Events.Stop = false
	if cfg.EventEnabled("stop") {
		t.Fatal("expected stop disabled")
	}
}
