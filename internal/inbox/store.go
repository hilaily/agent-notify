package inbox

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Store struct {
	path string
}

func DefaultStorePath() string {
	return filepath.Join(os.Getenv("HOME"), ".local", "state", "agent-notify", "inbox.jsonl")
}

func NewStore(path string) Store {
	if path == "" {
		path = DefaultStorePath()
	}
	return Store{path: path}
}

func (s Store) Path() string {
	return s.path
}

func (s Store) Append(rec Record) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(rec); err != nil {
		return err
	}
	return nil
}

func (s Store) List() ([]Record, error) {
	f, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var records []Record
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var rec Record
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}
		records = append(records, rec)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func (s Store) Pending() ([]Record, error) {
	records, err := s.List()
	if err != nil {
		return nil, err
	}
	var pending []Record
	for _, rec := range records {
		if rec.Status == StatusPending {
			pending = append(pending, rec)
		}
	}
	return pending, nil
}

func (s Store) MarkDone(ids []string) (int, error) {
	idSet := makeSet(ids)
	return s.rewrite(func(rec Record) (Record, bool, bool) {
		if _, ok := idSet[rec.ID]; !ok {
			return rec, true, false
		}
		if rec.Status == StatusDone {
			return rec, true, false
		}
		rec.Status = StatusDone
		return rec, true, true
	})
}

func (s Store) Remove(ids []string) (int, error) {
	idSet := makeSet(ids)
	return s.rewrite(func(rec Record) (Record, bool, bool) {
		if _, ok := idSet[rec.ID]; ok {
			return rec, false, true
		}
		return rec, true, false
	})
}

func (s Store) ClearDone() (int, error) {
	return s.rewrite(func(rec Record) (Record, bool, bool) {
		if rec.Status == StatusDone {
			return rec, false, true
		}
		return rec, true, false
	})
}

func (s Store) ClearAll() (int, error) {
	records, err := s.List()
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return 0, err
	}
	if err := os.WriteFile(s.path, nil, 0644); err != nil {
		return 0, err
	}
	return len(records), nil
}

func (s Store) rewrite(fn func(Record) (Record, bool, bool)) (int, error) {
	records, err := s.List()
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return 0, err
	}

	tmp, err := os.CreateTemp(filepath.Dir(s.path), "inbox-*.jsonl")
	if err != nil {
		return 0, err
	}
	tmpPath := tmp.Name()
	changed := 0
	enc := json.NewEncoder(tmp)
	for _, rec := range records {
		next, keep, didChange := fn(rec)
		if didChange {
			changed++
		}
		if !keep {
			continue
		}
		if err := enc.Encode(next); err != nil {
			tmp.Close()
			os.Remove(tmpPath)
			return 0, err
		}
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return 0, err
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		os.Remove(tmpPath)
		return 0, err
	}
	return changed, nil
}

func makeSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}
