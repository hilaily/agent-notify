package hook

import (
	"bytes"
	"testing"

	"github.com/longbin/agent-notify/internal/config"
)

func TestRunCursorEmptyStdinDoesNotBlock(t *testing.T) {
	cfg := config.Default()
	err := RunCursor(bytes.NewReader(nil), cfg, "stop", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunCursorWithPayload(t *testing.T) {
	cfg := config.Default()
	err := RunCursor(bytes.NewReader([]byte(`{"workspace_roots":["/tmp/proj"]}`)), cfg, "stop", nil)
	if err != nil {
		t.Fatal(err)
	}
}
