package notify

import "testing"

func TestHookMethodsSkipStdoutWhenNotTTY(t *testing.T) {
	methods := hookMethods(true, true, "/dev/pts/1")
	if len(methods) < 2 {
		t.Fatalf("expected hook methods, got %v", methods)
	}
	if methods[0] != MethodControllingTTYRaw {
		t.Fatalf("expected controlling tty first, got %v", methods)
	}
	for _, m := range methods {
		if m == MethodPassthroughStdout || m == MethodDirectStdout {
			t.Fatalf("stdout methods should not appear without terminal stdout, got %v", methods)
		}
	}
}

func TestHookMethodsIncludeStdoutWhenTTY(t *testing.T) {
	if !stdoutIsTerminal() {
		t.Skip("stdout is not a terminal in test runner")
	}
	methods := hookMethods(true, false, "/dev/pts/1")
	found := false
	for _, m := range methods {
		if m == MethodPassthroughStdout {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected passthrough stdout in manual mode, got %v", methods)
	}
}
