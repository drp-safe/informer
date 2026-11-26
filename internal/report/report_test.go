package report

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/drp-safe/informer/internal/inventory"
)

func testSender(url string) *Sender {
	s := NewSender(url, "secret-token", 5*time.Second)
	s.Backoff = func(int) time.Duration { return time.Millisecond }
	return s
}

var payload = Payload{
	Namespace:   "pra",
	CollectedAt: time.Date(2026, 11, 26, 15, 0, 0, 0, time.UTC),
	Images:      []inventory.Image{{Image: "nginx", Tag: "1.27"}},
}

func TestSendPostsJSONWithBearer(t *testing.T) {
	var got Payload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("méthode %s, POST attendu", r.Method)
		}
		if h := r.Header.Get("Authorization"); h != "Bearer secret-token" {
			t.Errorf("Authorization = %q", h)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q", ct)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("corps illisible : %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	status, err := testSender(srv.URL).Send(context.Background(), payload)
	if err != nil || status != http.StatusAccepted {
		t.Fatalf("Send = %d, %v", status, err)
	}
	if got.Namespace != "pra" || len(got.Images) != 1 || got.Images[0].Tag != "1.27" {
		t.Errorf("payload reçu inattendu : %+v", got)
	}
}

func TestSendRetriesOnServerError(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if _, err := testSender(srv.URL).Send(context.Background(), payload); err != nil {
		t.Fatalf("succès attendu à la 3e tentative : %v", err)
	}
	if n := calls.Load(); n != 3 {
		t.Errorf("%d appels, 3 attendus", n)
	}
}

func TestSendGivesUpAfterAttempts(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	if _, err := testSender(srv.URL).Send(context.Background(), payload); err == nil {
		t.Fatal("échec attendu")
	}
	if n := calls.Load(); n != 3 {
		t.Errorf("%d appels, 3 attendus", n)
	}
}

func TestSendDoesNotRetryClientError(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "jeton refusé", http.StatusUnauthorized)
	}))
	defer srv.Close()

	status, err := testSender(srv.URL).Send(context.Background(), payload)
	if err == nil || status != http.StatusUnauthorized {
		t.Fatalf("Send = %d, %v ; 401 attendu", status, err)
	}
	if !strings.Contains(err.Error(), "jeton refusé") {
		t.Errorf("l'erreur doit reprendre la réponse : %v", err)
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Error("le jeton ne doit jamais apparaître dans une erreur")
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("%d appels, 1 attendu (pas de nouvelle tentative sur 4xx)", n)
	}
}
