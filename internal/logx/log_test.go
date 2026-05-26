package logx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppendAndTail(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	Append("hello %s", "world")
	lines, err := Tail(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0] == "" {
		t.Fatal("empty line")
	}

	path := filepath.Join(dir, ".local", "state", "agent-notify", "hook.log")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("log file not created: %v", err)
	}
}
