package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/longbin/agent-notify/internal/inbox"
)

func TestInboxListAndDoneCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := inbox.NewStore("")
	if err := store.Append(inbox.Record{ID: "id-1", Status: inbox.StatusPending, Host: "host-a", Agent: "Cursor", Event: "stop", CWD: "/tmp/proj", Title: "ready"}); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := cmdInbox([]string{"list"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "id-1") || !strings.Contains(out.String(), "ready") {
		t.Fatalf("unexpected list output: %s", out.String())
	}

	out.Reset()
	if err := cmdInbox([]string{"done", "id-1"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "marked 1 done") {
		t.Fatalf("unexpected done output: %s", out.String())
	}
	out.Reset()
	if err := cmdInbox([]string{"list"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "id-1") {
		t.Fatalf("done record should be hidden by default: %s", out.String())
	}
}

func TestInboxShowAndRemoveCommands(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	store := inbox.NewStore("")
	if err := store.Append(inbox.Record{ID: "id-1", Status: inbox.StatusPending, Body: "body text", Title: "title"}); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := cmdInbox([]string{"show", "id-1"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "body text") {
		t.Fatalf("unexpected show output: %s", out.String())
	}
	out.Reset()
	if err := cmdInbox([]string{"rm", "id-1"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "removed 1") {
		t.Fatalf("unexpected rm output: %s", out.String())
	}
}

func TestInboxSSHConfigInstallCommandPrintsBlock(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "ssh_config")

	var out bytes.Buffer
	if err := cmdInbox([]string{"ssh-config", "install", "--path", path, "--remote-socket", "/tmp/remote-agent-notify.sock", "--socket", "/run/user/1000/agent-notify.sock"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "RemoteForward /tmp/remote-agent-notify.sock /run/user/1000/agent-notify.sock") {
		t.Fatalf("config not written: %s", string(data))
	}
	if !strings.Contains(out.String(), "wrote SSH config") || !strings.Contains(out.String(), "# BEGIN agent-notify inbox") {
		t.Fatalf("block not printed: %s", out.String())
	}
}
