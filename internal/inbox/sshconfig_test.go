package inbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallSSHConfigCreatesManagedBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")

	block, err := InstallSSHConfig(path, "/tmp/remote-agent-notify.sock", "/run/user/1000/agent-notify.sock")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "# BEGIN agent-notify inbox") {
		t.Fatalf("missing managed block: %s", text)
	}
	if !strings.Contains(text, "RemoteForward /tmp/remote-agent-notify.sock /run/user/1000/agent-notify.sock") {
		t.Fatalf("missing RemoteForward: %s", text)
	}
	if !strings.Contains(block, "Host *") || !strings.Contains(block, "RemoteForward") {
		t.Fatalf("unexpected returned block: %s", block)
	}
}

func TestInstallSSHConfigReplacesManagedBlockAndPreservesUserConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	existing := `Host prod
    HostName prod.example

# BEGIN agent-notify inbox
Host *
    RemoteForward /old.sock /old.sock
# END agent-notify inbox
`
	if err := os.WriteFile(path, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := InstallSSHConfig(path, "/remote-new.sock", "/local-new.sock")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Count(text, "# BEGIN agent-notify inbox") != 1 {
		t.Fatalf("expected one managed block: %s", text)
	}
	if !strings.Contains(text, "Host prod") {
		t.Fatalf("user config not preserved: %s", text)
	}
	if strings.Contains(text, "/old.sock") || !strings.Contains(text, "/remote-new.sock") || !strings.Contains(text, "/local-new.sock") {
		t.Fatalf("block not replaced: %s", text)
	}
}
