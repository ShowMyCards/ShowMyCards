package scryfall

import (
	"context"
	"errors"
	"testing"

	"github.com/BlueMonday/go-scryfall"
)

// mockAPI implements ScryfallAPI for testing
type mockAPI struct {
	searchFunc  func(ctx context.Context, query string, opts scryfall.SearchCardsOptions) (scryfall.CardListResponse, error)
	getCardFunc func(ctx context.Context, id string) (scryfall.Card, error)
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

func (m *mockAPI) ListSets(ctx context.Context) ([]scryfall.Set, error) {
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
