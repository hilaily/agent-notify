package notify

import (
	"bytes"
	"testing"
)

func TestSendDirect(t *testing.T) {
	var buf bytes.Buffer
	err := Send(SendOptions{
		Protocol: "osc777",
		Title:    "Cursor — app",
		Body:     "等待输入",
		Writer:   &buf,
		InTmux:   false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("777;notify")) {
		t.Fatalf("missing osc777: %q", buf.String())
	}
}

func TestSendTmuxUsesPassthroughWhenNoTTY(t *testing.T) {
	var buf bytes.Buffer
	err := Send(SendOptions{
		Protocol:  "osc777",
		Title:     "t",
		Body:      "b",
		Writer:    &buf,
		InTmux:    true,
		Layers:    1,
		ClientTTY: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("\033Ptmux;")) {
		t.Fatalf("expected passthrough prefix, got %q", buf.String())
	}
}
