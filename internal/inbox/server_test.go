package inbox

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandlerAppendsInboxRecord(t *testing.T) {
	store := NewStore(t.TempDir() + "/inbox.jsonl")
	handler := NewHandler(store)
	body := bytes.NewBufferString(`{"host":"remote-a","agent":"cursor","event":"stop","title":"done"}`)

	req := httptest.NewRequest(http.MethodPost, "/inbox", body)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	recs, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 record, got %d", len(recs))
	}
	if recs[0].ID == "" || recs[0].Time.IsZero() || recs[0].Status != StatusPending {
		t.Fatalf("record not completed: %+v", recs[0])
	}
}

func TestHandlerRejectsInvalidRequests(t *testing.T) {
	handler := NewHandler(NewStore(t.TempDir() + "/inbox.jsonl"))

	for _, tc := range []struct {
		name   string
		method string
		body   string
		want   int
	}{
		{name: "method", method: http.MethodGet, body: `{}`, want: http.StatusMethodNotAllowed},
		{name: "json", method: http.MethodPost, body: `{bad`, want: http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/inbox", bytes.NewBufferString(tc.body))
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != tc.want {
				t.Fatalf("status=%d want=%d", rr.Code, tc.want)
			}
		})
	}
}

func TestClientPostsRecord(t *testing.T) {
	var got Record
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/inbox" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{URL: server.URL, Timeout: time.Second})
	if err := client.Upload(context.Background(), Record{ID: "id-1", Title: "ready"}); err != nil {
		t.Fatal(err)
	}
	if got.ID != "id-1" || got.Title != "ready" {
		t.Fatalf("unexpected upload body: %+v", got)
	}
}

func TestClientReturnsErrorForServerFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{URL: server.URL, Timeout: time.Second})
	if err := client.Upload(context.Background(), Record{}); err == nil {
		t.Fatal("expected upload error")
	}
}
