package inbox

import "testing"

func TestBuildRecordPopulatesMetadata(t *testing.T) {
	t.Setenv("TMUX_PANE", "%12")

	rec := BuildRecord(BuildInput{
		Agent:  "Cursor",
		Event:  "stop",
		CWD:    "/work/proj",
		Title:  "Cursor - proj",
		Body:   "等待输入",
		Source: SourceLocal,
	})

	if rec.ID == "" {
		t.Fatal("expected id")
	}
	if rec.Time.IsZero() {
		t.Fatal("expected time")
	}
	if rec.Host == "" {
		t.Fatal("expected host")
	}
	if rec.Agent != "Cursor" || rec.Event != "stop" || rec.CWD != "/work/proj" {
		t.Fatalf("unexpected record: %+v", rec)
	}
	if rec.Status != StatusPending || rec.Source != SourceLocal {
		t.Fatalf("unexpected status/source: %+v", rec)
	}
	if rec.Tmux.Pane != "%12" {
		t.Fatalf("unexpected tmux pane: %+v", rec.Tmux)
	}
}
