package notify

import "testing"

func TestBuildOSC777(t *testing.T) {
	seq := BuildSequence("osc777", "Claude — proj", "等待输入")
	want := "\033]777;notify;Claude — proj;等待输入\007"
	if seq != want {
		t.Fatalf("got %q want %q", seq, want)
	}
}

func TestBuildOSC9(t *testing.T) {
	seq := BuildSequence("osc9", "ignored", "等待输入")
	want := "\033]9;等待输入\007"
	if seq != want {
		t.Fatalf("got %q want %q", seq, want)
	}
}

func TestSemicolonInTitleEscaped(t *testing.T) {
	seq := BuildSequence("osc777", "a;b", "body")
	if seq != "\033]777;notify;a\\;b;body\007" {
		t.Fatalf("unexpected escape: %q", seq)
	}
}
