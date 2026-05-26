package inbox

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendCreatesJSONLFile(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "inbox.jsonl"))

	rec := Record{
		ID:     "id-1",
		Time:   time.Date(2026, 5, 26, 16, 0, 0, 0, time.UTC),
		Host:   "host-a",
		Agent:  "cursor",
		Event:  "stop",
		CWD:    "/work/proj",
		Title:  "Cursor - proj",
		Body:   "等待输入",
		Status: StatusPending,
	}
	if err := store.Append(rec); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "inbox.jsonl")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected inbox file: %v", err)
	}
	recs, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 record, got %d", len(recs))
	}
	if recs[0].ID != "id-1" || recs[0].Status != StatusPending {
		t.Fatalf("unexpected record: %+v", recs[0])
	}
}

func TestPendingFiltersDoneRecords(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "inbox.jsonl"))
	mustAppend(t, store, Record{ID: "pending", Status: StatusPending})
	mustAppend(t, store, Record{ID: "done", Status: StatusDone})

	recs, err := store.Pending()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].ID != "pending" {
		t.Fatalf("unexpected pending records: %+v", recs)
	}
}

func TestMarkDoneRewritesMatchingRecords(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "inbox.jsonl"))
	mustAppend(t, store, Record{ID: "a", Status: StatusPending})
	mustAppend(t, store, Record{ID: "b", Status: StatusPending})

	n, err := store.MarkDone([]string{"b"})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 updated, got %d", n)
	}
	recs, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if recs[0].Status != StatusPending || recs[1].Status != StatusDone {
		t.Fatalf("unexpected statuses: %+v", recs)
	}
}

func TestRemoveAndClearDone(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "inbox.jsonl"))
	mustAppend(t, store, Record{ID: "a", Status: StatusPending})
	mustAppend(t, store, Record{ID: "b", Status: StatusDone})
	mustAppend(t, store, Record{ID: "c", Status: StatusPending})

	removed, err := store.Remove([]string{"a"})
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("expected 1 removed, got %d", removed)
	}
	cleared, err := store.ClearDone()
	if err != nil {
		t.Fatal(err)
	}
	if cleared != 1 {
		t.Fatalf("expected 1 cleared, got %d", cleared)
	}
	recs, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].ID != "c" {
		t.Fatalf("unexpected records: %+v", recs)
	}
}

func TestListSkipsInvalidJSONLLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inbox.jsonl")
	data := []byte("{bad json\n{\"id\":\"ok\",\"status\":\"pending\"}\n")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	store := NewStore(path)

	recs, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].ID != "ok" {
		t.Fatalf("unexpected records: %+v", recs)
	}
}

func mustAppend(t *testing.T, store Store, rec Record) {
	t.Helper()
	if err := store.Append(rec); err != nil {
		t.Fatal(err)
	}
}
