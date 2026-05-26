package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
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
