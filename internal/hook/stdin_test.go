package hook

import (
	"bytes"
	"testing"

	"github.com/longbin/agent-notify/internal/config"
	"github.com/longbin/agent-notify/internal/notify"
)

func stubCursorSend(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	prevHook := sendForHook
	prevAuto := sendAuto
	sendForHook = func(string, string, string) (notify.SendResult, error) {
		return notify.SendResult{Method: "test"}, nil
	}
	sendAuto = sendForHook
	t.Cleanup(func() {
		sendForHook = prevHook
		sendAuto = prevAuto
	})
}

func TestRunCursorEmptyStdinDoesNotBlock(t *testing.T) {
	stubCursorSend(t)
	cfg := config.Default()
	err := RunCursor(bytes.NewReader(nil), cfg, "stop", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunCursorWithPayload(t *testing.T) {
	stubCursorSend(t)
	cfg := config.Default()
	err := RunCursor(bytes.NewReader([]byte(`{"workspace_roots":["/tmp/proj"]}`)), cfg, "stop", nil)
	if err != nil {
		t.Fatal(err)
	}
}
