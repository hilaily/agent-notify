package hook

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/longbin/agent-notify/internal/config"
)

func TestCursorStopHookDisabled(t *testing.T) {
	cfg := config.Default()
	cfg.Events.Stop = false
	err := RunCursor(bytes.NewReader([]byte(`{"workspace_roots":["/tmp/proj"]}`)), cfg, "stop", &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestClaudeStopOutputsTerminalSequence(t *testing.T) {
	cfg := config.Default()
	var out bytes.Buffer
	err := RunClaude(strings.NewReader(`{"stop_hook_active":false}`), cfg, "stop", &out)
	if err != nil {
		t.Fatal(err)
	}
	var resp map[string]string
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp["terminalSequence"], "777;notify") {
		t.Fatalf("bad sequence: %v", resp)
	}
}

func TestClaudeStopHookActiveSkips(t *testing.T) {
	cfg := config.Default()
	var out bytes.Buffer
	err := RunClaude(strings.NewReader(`{"stop_hook_active":true}`), cfg, "stop", &out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "{}" {
		t.Fatalf("expected {}, got %q", out.String())
	}
}
