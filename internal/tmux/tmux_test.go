package tmux

import "testing"

func TestInTmux(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-123,1,0")
	if !InTmux() {
		t.Fatal("expected InTmux true")
	}
}

func TestWrapPassthroughSingle(t *testing.T) {
	inner := "\033]777;notify;t;b\007"
	got := WrapPassthrough(inner)
	want := "\033Ptmux;\033" + inner + "\033\\"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestWrapPassthroughNested(t *testing.T) {
	inner := "\033]777;notify;t;b\007"
	got := WrapPassthroughLayers(inner, 2)
	once := WrapPassthrough(inner)
	twice := WrapPassthrough(once)
	if got != twice {
		t.Fatalf("nested wrap mismatch")
	}
}
