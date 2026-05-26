package context

import "testing"

func TestRenderTitle(t *testing.T) {
	meta := Meta{Agent: "Cursor", Context: "myapp"}
	got := Render("{agent} — {context}", meta)
	want := "Cursor — myapp"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestContextFromCWD(t *testing.T) {
	meta := Meta{Agent: "Claude", CWD: "/home/user/code/myapp"}
	if meta.ResolveContext("") != "myapp" {
		t.Fatalf("got %q", meta.ResolveContext(""))
	}
}

func TestContextPrefersWindow(t *testing.T) {
	meta := Meta{Agent: "Claude", CWD: "/home/user/code/myapp"}
	if meta.ResolveContext("tmux-win") != "tmux-win" {
		t.Fatalf("expected window name")
	}
}
