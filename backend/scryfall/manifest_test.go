package scryfall

import (
	"backend/version"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetManifestPage_DecodesAndSendsHeaders(t *testing.T) {
	var gotUA, gotAccept string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"object": "list",
			"total_cards": 120276,
			"has_more": true,
			"next_page": "https://api.scryfall.com/cards/manifest?lang=en&page=2",
			"data": [
				{
					"id": "card-1",
					"oracle_id": "oracle-1",
					"name": "Lightning Bolt",
					"set_code": "lea",
					"collector_number": "161",
					"lang": "en",
					"created_at": null,
					"data_updated_at": null,
					"image_updated_at": "2026-10-10T13:01:28Z"
				}
			]
		}`))
	}))
	defer server.Close()

	client := newClientWithAPI(&mockAPI{})
	client.manifestInterval = 0
	defer client.Close()

	page, err := client.GetManifestPage(context.Background(), server.URL+"/cards/manifest?lang=en&page=1")
	if err != nil {
		t.Fatalf("GetManifestPage failed: %v", err)
	}

	if gotUA != version.UserAgent() || !strings.HasPrefix(gotUA, "ShowMyCards/") {
		t.Errorf("User-Agent = %q, want %q", gotUA, version.UserAgent())
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json", gotAccept)
	}
	if page.TotalCards != 120276 || !page.HasMore || page.NextPage == "" {
		t.Errorf("unexpected page metadata: %+v", page)
	}
	if len(page.Data) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(page.Data))
	}
	entry := page.Data[0]
	if entry.ID != "card-1" || entry.CreatedAt != nil || entry.DataUpdatedAt != nil {
		t.Errorf("unexpected entry: %+v", entry)
	}
	if entry.ImageUpdatedAt == nil || *entry.ImageUpdatedAt != "2026-10-10T13:01:28Z" {
		t.Errorf("unexpected ImageUpdatedAt: %v", entry.ImageUpdatedAt)
	}
}

func TestGetManifestPage_PacingAndContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","has_more":false,"data":[]}`))
	}))
	defer server.Close()

	client := newClientWithAPI(&mockAPI{})
	client.manifestInterval = 60 * time.Millisecond
	defer client.Close()

	start := time.Now()
	if _, err := client.GetManifestPage(context.Background(), server.URL); err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if _, err := client.GetManifestPage(context.Background(), server.URL); err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 50*time.Millisecond {
		t.Errorf("expected manifest pacing to wait ~60ms, elapsed=%v", elapsed)
	}

	// Cancel context while waiting for the next slot.
	client.manifestInterval = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := client.GetManifestPage(ctx, server.URL)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestGetManifestPage_RateLimitRetry(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"code":"too_many_requests"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","total_cards":1,"has_more":false,"data":[{"id":"ok"}]}`))
	}))
	defer server.Close()

	client := newClientWithAPI(&mockAPI{})
	client.manifestInterval = 0
	client.retryDelay = time.Millisecond
	defer client.Close()

	page, err := client.GetManifestPage(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("expected retry to succeed, got: %v", err)
	}
	if attempts.Load() != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts.Load())
	}
	if len(page.Data) != 1 || page.Data[0].ID != "ok" {
		t.Errorf("unexpected data: %+v", page.Data)
	}
}

func TestGetManifestPage_RateLimitExhausted(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := newClientWithAPI(&mockAPI{})
	client.manifestInterval = 0
	client.retryDelay = time.Millisecond
	defer client.Close()

	_, err := client.GetManifestPage(context.Background(), server.URL)
	if err == nil {
		t.Fatal("expected error after exhausting 429 retries")
	}
	if attempts.Load() != maxRateLimitRetries+1 {
		t.Errorf("expected %d attempts, got %d", maxRateLimitRetries+1, attempts.Load())
	}
}

func TestGetManifestPage_ErrorResponses(t *testing.T) {
	t.Run("non-200 status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("bad gateway"))
		}))
		defer server.Close()

		client := newClientWithAPI(&mockAPI{})
		client.manifestInterval = 0
		defer client.Close()

		_, err := client.GetManifestPage(context.Background(), server.URL)
		if err == nil || !strings.Contains(err.Error(), "502") {
			t.Errorf("expected 502 error, got %v", err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("{not-json"))
		}))
		defer server.Close()

		client := newClientWithAPI(&mockAPI{})
		client.manifestInterval = 0
		defer client.Close()

		_, err := client.GetManifestPage(context.Background(), server.URL)
		if err == nil {
			t.Fatal("expected decode error")
		}
	})
}
