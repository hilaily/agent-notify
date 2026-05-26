package inbox

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func NewHandler(store Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/inbox", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()
		var rec Record
		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		CompleteRecord(&rec)
		if err := store.Append(rec); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": rec.ID})
	})
	return mux
}

func CompleteRecord(rec *Record) {
	if rec.ID == "" {
		rec.ID = NewID(time.Now())
	}
	if rec.Time.IsZero() {
		rec.Time = time.Now()
	}
	if rec.Status == "" {
		rec.Status = StatusPending
	}
}

func NewID(t time.Time) string {
	var b [3]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-000000", t.Format("20060102-150405"))
	}
	return fmt.Sprintf("%s-%s", t.Format("20060102-150405"), hex.EncodeToString(b[:]))
}
