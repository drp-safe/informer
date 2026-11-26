// Package report envoie le listing des images à l'API destinataire.
package report

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/drp-safe/informer/internal/inventory"
)

// Payload est le corps JSON envoyé.
type Payload struct {
	Namespace   string            `json:"namespace"`
	CollectedAt time.Time         `json:"collected_at"`
	Images      []inventory.Image `json:"images"`
}

// Sender poste un Payload avec un jeton Bearer.
type Sender struct {
	URL      string
	Token    string
	Client   *http.Client
	Attempts int
	// Backoff renvoie l'attente avant la tentative n (n ≥ 1).
	Backoff func(n int) time.Duration
}

// NewSender construit un Sender : 3 tentatives, attente de 2 s puis 4 s.
func NewSender(url, token string, timeout time.Duration) *Sender {
	return &Sender{
		URL:      url,
		Token:    token,
		Client:   &http.Client{Timeout: timeout},
		Attempts: 3,
		Backoff:  func(n int) time.Duration { return time.Duration(1<<n) * time.Second },
	}
}

// permanentError signale un échec qu'une nouvelle tentative ne corrigera pas.
type permanentError struct{ error }

// Send poste le payload. Une erreur réseau ou une réponse 5xx est retentée ;
// une réponse 4xx (jeton refusé, corps rejeté) échoue aussitôt.
func (s *Sender) Send(ctx context.Context, p Payload) (int, error) {
	body, err := json.Marshal(p)
	if err != nil {
		return 0, fmt.Errorf("encodage JSON : %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= s.Attempts; attempt++ {
		if attempt > 1 {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(s.Backoff(attempt - 1)):
			}
		}
		status, err := s.post(ctx, body)
		if err == nil {
			return status, nil
		}
		if pe, ok := err.(permanentError); ok {
			return status, pe.error
		}
		lastErr = err
	}
	return 0, fmt.Errorf("échec après %d tentatives : %w", s.Attempts, lastErr)
}

func (s *Sender) post(ctx context.Context, body []byte) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.URL, bytes.NewReader(body))
	if err != nil {
		return 0, permanentError{fmt.Errorf("requête : %w", err)}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.Token)

	resp, err := s.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	// Extrait borné de la réponse, pour le journal en cas d'échec.
	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return resp.StatusCode, nil
	case resp.StatusCode >= 500:
		return resp.StatusCode, fmt.Errorf("réponse %d : %s", resp.StatusCode, bytes.TrimSpace(snippet))
	default:
		return resp.StatusCode, permanentError{fmt.Errorf("réponse %d : %s", resp.StatusCode, bytes.TrimSpace(snippet))}
	}
}
