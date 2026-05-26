package context

import "testing"

func TestRenderTitle(t *testing.T) {
	meta := Meta{Agent: "Cursor", CWD: "/home/user/code/myapp"}
	got := Render("{agent} — {context}", meta)
	want := "Cursor — myapp"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestContextFromCWD(t *testing.T) {
	meta := Meta{Agent: "Claude", CWD: "/home/user/code/myapp"}
	if meta.ResolveContext() != "myapp" {
		t.Fatalf("got %q", meta.ResolveContext())
	}
}

func TestContextExplicitOverride(t *testing.T) {
	meta := Meta{Agent: "Claude", CWD: "/home/user/code/myapp", Context: "custom"}
	if meta.ResolveContext() != "custom" {
		t.Fatalf("expected explicit context override")
	}
}
