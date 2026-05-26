package hook

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/longbin/agent-notify/internal/config"
	"github.com/longbin/agent-notify/internal/inbox"
)

func TestCursorStopHookDisabled(t *testing.T) {
	stubCursorSend(t)
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

func TestCursorHookUploadsInboxRecord(t *testing.T) {
	stubCursorSend(t)
	var uploaded inbox.Record
	stubInbox(t,
		func(rec inbox.Record, cfg config.Config) error {
			uploaded = rec
			return nil
		},
		func(rec inbox.Record) error {
			t.Fatalf("unexpected fallback append: %+v", rec)
			return nil
		},
	)

	cfg := config.Default()
	err := RunCursor(bytes.NewReader([]byte(`{"workspace_roots":["/tmp/proj"]}`)), cfg, "stop", &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if uploaded.Agent != "Cursor" || uploaded.Event != "stop" || uploaded.CWD != "/tmp/proj" {
		t.Fatalf("unexpected upload record: %+v", uploaded)
	}
}

func TestCursorHookFallbacksWhenInboxUploadFails(t *testing.T) {
	stubCursorSend(t)
	var fallback inbox.Record
	stubInbox(t,
		func(rec inbox.Record, cfg config.Config) error {
			return errors.New("offline")
		},
		func(rec inbox.Record) error {
			fallback = rec
			return nil
		},
	)

	cfg := config.Default()
	err := RunCursor(bytes.NewReader([]byte(`{"workspace_roots":["/tmp/proj"]}`)), cfg, "stop", &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if fallback.Source != inbox.SourceFallback || fallback.Title == "" {
		t.Fatalf("unexpected fallback record: %+v", fallback)
	}
}

func TestDisabledHookDoesNotRecordInbox(t *testing.T) {
	stubCursorSend(t)
	stubInbox(t,
		func(rec inbox.Record, cfg config.Config) error {
			t.Fatalf("unexpected upload: %+v", rec)
			return nil
		},
		func(rec inbox.Record) error {
			t.Fatalf("unexpected fallback: %+v", rec)
			return nil
		},
	)

	cfg := config.Default()
	cfg.Events.Stop = false
	if err := RunCursor(bytes.NewReader([]byte(`{}`)), cfg, "stop", &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
}

func TestClaudeHookUploadsInboxRecord(t *testing.T) {
	var uploaded inbox.Record
	stubInbox(t,
		func(rec inbox.Record, cfg config.Config) error {
			uploaded = rec
			return nil
		},
		func(rec inbox.Record) error {
			t.Fatalf("unexpected fallback append: %+v", rec)
			return nil
		},
	)

	cfg := config.Default()
	var out bytes.Buffer
	if err := RunClaude(strings.NewReader(`{"stop_hook_active":false}`), cfg, "stop", &out); err != nil {
		t.Fatal(err)
	}
	if uploaded.Agent != "Claude" || uploaded.Event != "stop" || uploaded.Title == "" {
		t.Fatalf("unexpected upload record: %+v", uploaded)
	}
}

func stubInbox(t *testing.T, upload func(inbox.Record, config.Config) error, appendLocal func(inbox.Record) error) {
	t.Helper()
	prevUpload := uploadInboxRecord
	prevAppend := appendInboxRecord
	uploadInboxRecord = upload
	appendInboxRecord = appendLocal
	t.Cleanup(func() {
		uploadInboxRecord = prevUpload
		appendInboxRecord = prevAppend
	})
}
