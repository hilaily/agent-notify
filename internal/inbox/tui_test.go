package inbox

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTUIViewShowsRecords(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "inbox.jsonl"))
	mustAppend(t, store, Record{ID: "id-1", Status: StatusPending, Host: "host", Agent: "Cursor", Event: "stop", Title: "ready"})

	model := newTUIModel(store)
	view := model.View()
	if !strings.Contains(view, "ready") || !strings.Contains(view, "id-1") {
		t.Fatalf("unexpected view: %s", view)
	}
}

func TestTUIDoneMarksSelectedRecordDone(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "inbox.jsonl"))
	mustAppend(t, store, Record{ID: "id-1", Status: StatusPending, Title: "ready"})

	model := newTUIModel(store)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = updated.(tuiModel)

	recs, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if recs[0].Status != StatusDone {
		t.Fatalf("expected done, got %+v", recs[0])
	}
}
