package scryfall

import (
	"backend/version"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/BlueMonday/go-scryfall"
)

// roundTripFunc lets a plain function act as an http.RoundTripper, so tests
// can inspect outbound requests without opening a network listener.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestNewAPI_SendsShowMyCardsUserAgent guards against regressing to the
// go-scryfall default User-Agent, which Scryfall rejects as generic.
func TestNewAPI_SendsShowMyCardsUserAgent(t *testing.T) {
	var gotUA string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotUA = r.Header.Get("User-Agent")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"object":"card","id":"test-id"}`)),
			Request:    r,
		}, nil
	})

	api, err := newAPI(scryfall.WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("newAPI: %v", err)
	}

	if _, err := api.GetCard(context.Background(), "test-id"); err != nil {
		t.Fatalf("GetCard: %v", err)
	}

	if gotUA != version.UserAgent() {
		t.Errorf("User-Agent = %q, want %q", gotUA, version.UserAgent())
	}
	if !strings.HasPrefix(gotUA, "ShowMyCards/") {
		t.Errorf("User-Agent = %q, want prefix %q", gotUA, "ShowMyCards/")
	}
}

// mockAPI implements ScryfallAPI for testing
type mockAPI struct {
	searchFunc                func(ctx context.Context, query string, opts scryfall.SearchCardsOptions) (scryfall.CardListResponse, error)
	getCardFunc               func(ctx context.Context, id string) (scryfall.Card, error)
	getCardsByIdentifiersFunc func(ctx context.Context, identifiers []scryfall.CardIdentifier) (scryfall.GetCardsByIdentifiersResponse, error)
}

func (m *mockAPI) SearchCards(ctx context.Context, query string, opts scryfall.SearchCardsOptions) (scryfall.CardListResponse, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, query, opts)
	}
	return scryfall.CardListResponse{}, nil
}

func (m *mockAPI) GetCard(ctx context.Context, id string) (scryfall.Card, error) {
	if m.getCardFunc != nil {
		return m.getCardFunc(ctx, id)
	}
	return scryfall.Card{}, nil
}

func (m *mockAPI) GetCardsByIdentifiers(ctx context.Context, identifiers []scryfall.CardIdentifier) (scryfall.GetCardsByIdentifiersResponse, error) {
	if m.getCardsByIdentifiersFunc != nil {
		return m.getCardsByIdentifiersFunc(ctx, identifiers)
	}
	return scryfall.GetCardsByIdentifiersResponse{}, nil
}

func (m *mockAPI) ListSets(ctx context.Context) ([]scryfall.Set, error) {
	return nil, nil
}

func (m *mockAPI) ListCardSymbols(ctx context.Context) ([]scryfall.CardSymbol, error) {
	return nil, nil
}

func (m *mockAPI) AutocompleteCard(ctx context.Context, s string) ([]string, error) {
	return nil, nil
}

func TestSearch_Success(t *testing.T) {
	expectedCards := []scryfall.Card{
		{ID: "card-1", Name: "Lightning Bolt"},
		{ID: "card-2", Name: "Lightning Helix"},
	}

	mock := &mockAPI{
		searchFunc: func(ctx context.Context, query string, opts scryfall.SearchCardsOptions) (scryfall.CardListResponse, error) {
			if query != "lightning" {
				t.Errorf("expected query 'lightning', got '%s'", query)
			}
			return scryfall.CardListResponse{
				Cards:      expectedCards,
				TotalCards: 2,
				HasMore:    false,
			}, nil
		},
	}

	client := newClientWithAPI(mock)
	defer client.Close()

	result, err := client.Search(context.Background(), "lightning", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Cards) != 2 {
		t.Errorf("expected 2 cards, got %d", len(result.Cards))
	}

	if result.Cards[0].Name != "Lightning Bolt" {
		t.Errorf("expected first card 'Lightning Bolt', got '%s'", result.Cards[0].Name)
	}

	if result.TotalCards != 2 {
		t.Errorf("expected TotalCards 2, got %d", result.TotalCards)
	}
}

func TestSearch_Error(t *testing.T) {
	expectedErr := errors.New("api error")

	mock := &mockAPI{
		searchFunc: func(ctx context.Context, query string, opts scryfall.SearchCardsOptions) (scryfall.CardListResponse, error) {
			return scryfall.CardListResponse{}, expectedErr
		},
	}

	client := newClientWithAPI(mock)
	defer client.Close()

	_, err := client.Search(context.Background(), "test", 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("expected error '%v', got '%v'", expectedErr, err)
	}
}

func TestSearch_CachesResults(t *testing.T) {
	card := scryfall.Card{ID: "cached-card", Name: "Cached Card"}

	mock := &mockAPI{
		searchFunc: func(ctx context.Context, query string, opts scryfall.SearchCardsOptions) (scryfall.CardListResponse, error) {
			return scryfall.CardListResponse{Cards: []scryfall.Card{card}}, nil
		},
		getCardFunc: func(ctx context.Context, id string) (scryfall.Card, error) {
			t.Error("GetCard should not be called when card is cached")
			return scryfall.Card{}, nil
		},
	}

	client := newClientWithAPI(mock)
	defer client.Close()

	// Search to populate cache
	_, err := client.Search(context.Background(), "test", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// GetByID should return cached card without calling API
	result, err := client.GetByID(context.Background(), "cached-card")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != "Cached Card" {
		t.Errorf("expected 'Cached Card', got '%s'", result.Name)
	}
}

func TestSearch_Pagination(t *testing.T) {
	mock := &mockAPI{
		searchFunc: func(ctx context.Context, query string, opts scryfall.SearchCardsOptions) (scryfall.CardListResponse, error) {
			if opts.Page != 2 {
				t.Errorf("expected page 2, got %d", opts.Page)
			}
			return scryfall.CardListResponse{
				Cards:      []scryfall.Card{{ID: "page-2-card", Name: "Page 2 Card"}},
				TotalCards: 50,
				HasMore:    true,
			}, nil
		},
	}

	client := newClientWithAPI(mock)
	defer client.Close()

	result, err := client.Search(context.Background(), "test", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Page != 2 {
		t.Errorf("expected page 2, got %d", result.Page)
	}

	if !result.HasMore {
		t.Error("expected HasMore to be true")
	}

	if result.TotalCards != 50 {
		t.Errorf("expected TotalCards 50, got %d", result.TotalCards)
	}
}

func TestSearch_InvalidPage(t *testing.T) {
	mock := &mockAPI{
		searchFunc: func(ctx context.Context, query string, opts scryfall.SearchCardsOptions) (scryfall.CardListResponse, error) {
			if opts.Page != 1 {
				t.Errorf("expected page to be normalized to 1, got %d", opts.Page)
			}
			return scryfall.CardListResponse{}, nil
		},
	}

	client := newClientWithAPI(mock)
	defer client.Close()

	// Page 0 should be normalized to 1
	result, err := client.Search(context.Background(), "test", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Page != 1 {
		t.Errorf("expected page 1, got %d", result.Page)
	}
}

func TestGetByID_CacheHit(t *testing.T) {
	apiCalled := false

	mock := &mockAPI{
		getCardFunc: func(ctx context.Context, id string) (scryfall.Card, error) {
			apiCalled = true
			return scryfall.Card{ID: id, Name: "From API"}, nil
		},
	}

	client := newClientWithAPI(mock)
	defer client.Close()

	ctx := context.Background()

	// First call should hit API
	card1, err := client.GetByID(ctx, "test-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !apiCalled {
		t.Error("expected API to be called on first request")
	}
	if card1.Name != "From API" {
		t.Errorf("expected 'From API', got '%s'", card1.Name)
	}

	// Reset flag
	apiCalled = false

	// Second call should hit cache
	card2, err := client.GetByID(ctx, "test-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiCalled {
		t.Error("expected cache hit, but API was called")
	}
	if card2.Name != "From API" {
		t.Errorf("expected 'From API', got '%s'", card2.Name)
	}
}

func TestGetByID_CacheMiss(t *testing.T) {
	expectedCard := scryfall.Card{ID: "new-id", Name: "New Card"}

	mock := &mockAPI{
		getCardFunc: func(ctx context.Context, id string) (scryfall.Card, error) {
			if id != "new-id" {
				t.Errorf("expected id 'new-id', got '%s'", id)
			}
			return expectedCard, nil
		},
	}

	client := newClientWithAPI(mock)
	defer client.Close()

	card, err := client.GetByID(context.Background(), "new-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if card.Name != "New Card" {
		t.Errorf("expected 'New Card', got '%s'", card.Name)
	}
}

func TestGetByID_Error(t *testing.T) {
	expectedErr := errors.New("card not found")

	mock := &mockAPI{
		getCardFunc: func(ctx context.Context, id string) (scryfall.Card, error) {
			return scryfall.Card{}, expectedErr
		},
	}

	client := newClientWithAPI(mock)
	defer client.Close()

	_, err := client.GetByID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("expected error '%v', got '%v'", expectedErr, err)
	}
}

func TestGetByID_CachesResult(t *testing.T) {
	callCount := 0

	mock := &mockAPI{
		getCardFunc: func(ctx context.Context, id string) (scryfall.Card, error) {
			callCount++
			return scryfall.Card{ID: id, Name: "Fetched Card"}, nil
		},
	}

	client := newClientWithAPI(mock)
	defer client.Close()

	ctx := context.Background()

	// Call multiple times
	for i := 0; i < 5; i++ {
		_, err := client.GetByID(ctx, "same-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if callCount != 1 {
		t.Errorf("expected API to be called once, was called %d times", callCount)
	}
}

func TestGetCollectionByIDs_Empty(t *testing.T) {
	called := false
	mock := &mockAPI{
		getCardsByIdentifiersFunc: func(ctx context.Context, identifiers []scryfall.CardIdentifier) (scryfall.GetCardsByIdentifiersResponse, error) {
			called = true
			return scryfall.GetCardsByIdentifiersResponse{}, nil
		},
	}

	client := newClientWithAPI(mock)
	defer client.Close()

	cards, notFound, err := client.GetCollectionByIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("expected API not to be called for empty ID slice")
	}
	if cards != nil || notFound != nil {
		t.Errorf("expected nil slices, got cards=%v notFound=%v", cards, notFound)
	}
}

func TestGetCollectionByIDs_ExceedsMaxBatchSize(t *testing.T) {
	called := false
	mock := &mockAPI{
		getCardsByIdentifiersFunc: func(ctx context.Context, identifiers []scryfall.CardIdentifier) (scryfall.GetCardsByIdentifiersResponse, error) {
			called = true
			return scryfall.GetCardsByIdentifiersResponse{}, nil
		},
	}

	client := newClientWithAPI(mock)
	defer client.Close()

	ids := make([]string, MaxCollectionBatchSize+1)
	for i := range ids {
		ids[i] = "id"
	}

	_, _, err := client.GetCollectionByIDs(context.Background(), ids)
	if err == nil {
		t.Fatal("expected error for oversized batch")
	}
	if called {
		t.Error("expected API not to be called when batch exceeds max size")
	}
}

func TestGetCollectionByIDs_SuccessWithNotFound(t *testing.T) {
	mock := &mockAPI{
		getCardsByIdentifiersFunc: func(ctx context.Context, identifiers []scryfall.CardIdentifier) (scryfall.GetCardsByIdentifiersResponse, error) {
			if len(identifiers) != 2 || identifiers[0].ID != "id-1" || identifiers[1].ID != "id-missing" {
				t.Errorf("unexpected identifiers: %+v", identifiers)
			}
			return scryfall.GetCardsByIdentifiersResponse{
				Data:     []scryfall.Card{{ID: "id-1", Name: "Card One"}},
				NotFound: []scryfall.CardIdentifier{{ID: "id-missing"}},
			}, nil
		},
	}

	client := newClientWithAPI(mock)
	defer client.Close()

	cards, notFound, err := client.GetCollectionByIDs(context.Background(), []string{"id-1", "id-missing"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cards) != 1 || cards[0].ID != "id-1" {
		t.Errorf("unexpected cards: %+v", cards)
	}
	if len(notFound) != 1 || notFound[0] != "id-missing" {
		t.Errorf("unexpected notFound: %+v", notFound)
	}
}

func TestGetCollectionByIDs_RateLimitRetry(t *testing.T) {
	attempts := 0
	mock := &mockAPI{
		getCardsByIdentifiersFunc: func(ctx context.Context, identifiers []scryfall.CardIdentifier) (scryfall.GetCardsByIdentifiersResponse, error) {
			attempts++
			if attempts == 1 {
				return scryfall.GetCardsByIdentifiersResponse{}, &scryfall.Error{Status: http.StatusTooManyRequests, Details: "rate limited"}
			}
			return scryfall.GetCardsByIdentifiersResponse{
				Data: []scryfall.Card{{ID: "id-1", Name: "Recovered"}},
			}, nil
		},
	}

	client := newClientWithAPI(mock)
	client.retryDelay = time.Millisecond
	defer client.Close()

	cards, _, err := client.GetCollectionByIDs(context.Background(), []string{"id-1"})
	if err != nil {
		t.Fatalf("expected retry to succeed, got: %v", err)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
	if len(cards) != 1 || cards[0].Name != "Recovered" {
		t.Errorf("unexpected cards: %+v", cards)
	}
}

func TestGetCollectionByIDs_RateLimitExhausted(t *testing.T) {
	attempts := 0
	mock := &mockAPI{
		getCardsByIdentifiersFunc: func(ctx context.Context, identifiers []scryfall.CardIdentifier) (scryfall.GetCardsByIdentifiersResponse, error) {
			attempts++
			return scryfall.GetCardsByIdentifiersResponse{}, &scryfall.Error{Status: http.StatusTooManyRequests, Details: "still limited"}
		},
	}

	client := newClientWithAPI(mock)
	client.retryDelay = time.Millisecond
	defer client.Close()

	_, _, err := client.GetCollectionByIDs(context.Background(), []string{"id-1"})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if attempts != maxRateLimitRetries+1 {
		t.Errorf("expected %d attempts, got %d", maxRateLimitRetries+1, attempts)
	}
}

func TestGetCollectionByIDs_ContextCancelledDuringBackoff(t *testing.T) {
	mock := &mockAPI{
		getCardsByIdentifiersFunc: func(ctx context.Context, identifiers []scryfall.CardIdentifier) (scryfall.GetCardsByIdentifiersResponse, error) {
			return scryfall.GetCardsByIdentifiersResponse{}, &scryfall.Error{Status: http.StatusTooManyRequests}
		},
	}

	client := newClientWithAPI(mock)
	client.retryDelay = 5 * time.Second
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, _, err := client.GetCollectionByIDs(ctx, []string{"id-1"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}
