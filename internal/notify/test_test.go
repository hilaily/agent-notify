package notify

import (
	"bytes"
	"strings"
	"testing"

	"github.com/longbin/agent-notify/internal/tmux"
)

func TestAutoMethodsSSHPrefersPassthrough(t *testing.T) {
	methods := autoMethods(true, true)
	if methods[0] != MethodPassthroughStdout {
		t.Fatalf("expected passthrough first over ssh, got %v", methods)
	}
}

func TestAutoMethodsLocalPrefersClientTTY(t *testing.T) {
	methods := autoMethods(true, false)
	if methods[0] != MethodClientTTYRaw {
		t.Fatalf("expected client tty first locally, got %v", methods)
	}
}

func TestTestClaudeJSON(t *testing.T) {
	var out bytes.Buffer
	_, err := TestClaude("t", "b", false, &out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "terminalSequence") {
		t.Fatalf("expected json output, got %q", out.String())
	}
}

func TestEmitSequenceDirect(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("SSH_CONNECTION", "")
	_ = tmux.InTmux()
	seq := BuildSequence("osc777", "t", "b")
	var buf bytes.Buffer
	opts := SendOptions{Writer: &buf, InTmux: false}
	if err := deliver(seq, opts, MethodDirectStdout); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "777;notify") {
		t.Fatalf("unexpected output %q", buf.String())
	}
}
