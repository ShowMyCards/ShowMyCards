package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/models"

	"github.com/BlueMonday/go-scryfall"
	"github.com/gofiber/fiber/v3"
)

// mockScryfallClient implements the methods needed for testing
type mockScryfallClient struct {
	searchFunc func(ctx context.Context, query string, page int) (mockSearchResult, error)
}

type mockSearchResult struct {
	Cards      []scryfall.Card
	TotalCards int
	HasMore    bool
	Page       int
}

func (m *mockScryfallClient) Search(ctx context.Context, query string, page int) (mockSearchResult, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, query, page)
	}
	return mockSearchResult{}, nil
}

// testSearchHandler wraps the handler for testing with mock
type testSearchHandler struct {
	mock *mockScryfallClient
}

func (h *testSearchHandler) Search(c fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "query parameter 'q' is required",
		})
	}

	page := fiber.Query[int](c, "page", 1)
	if page < 1 {
		page = 1
	}

	result, err := h.mock.Search(c.RequestCtx(), query, page)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": "failed to search cards",
		})
	}

	cards := make([]EnhancedCardResult, len(result.Cards))
	for i, card := range result.Cards {
		cardResult := BuildCardResult(card)
		if card.ImageURIs != nil && card.ImageURIs.PNG != "" {
			cardResult.ImageURI = &card.ImageURIs.PNG
		}
		cards[i] = EnhancedCardResult{
			CardResult: cardResult,
			Inventory: CardInventoryData{
				ThisPrinting:   []models.Inventory{},
				OtherPrintings: []models.Inventory{},
				TotalQuantity:  0,
			},
		}
	}

	return c.JSON(SearchResponse{
		Data:       cards,
		Page:       result.Page,
		TotalCards: result.TotalCards,
		HasMore:    result.HasMore,
	})
}

func setupSearchTestApp(mock *mockScryfallClient) *fiber.App {
	app := fiber.New()
	handler := &testSearchHandler{mock: mock}
	app.Get("/search", handler.Search)
	return app
}

func TestSearch_Success(t *testing.T) {
	mock := &mockScryfallClient{
		searchFunc: func(ctx context.Context, query string, page int) (mockSearchResult, error) {
			if query != "lightning bolt" {
				t.Errorf("expected query 'lightning bolt', got '%s'", query)
			}
			return mockSearchResult{
				Cards: []scryfall.Card{
					{ID: "card-1", OracleID: "oracle-123", Name: "Lightning Bolt", ManaCost: "{R}", TypeLine: "Instant"},
					{ID: "card-2", OracleID: "oracle-123", Name: "Lightning Bolt", ManaCost: "{R}", TypeLine: "Instant", SetName: "Alpha"},
				},
				TotalCards: 50,
				HasMore:    true,
				Page:       1,
			}, nil
		},
	}

	app := setupSearchTestApp(mock)

	req := httptest.NewRequest(http.MethodGet, "/search?q=lightning+bolt", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result.Data) != 2 {
		t.Errorf("expected 2 cards, got %d", len(result.Data))
	}

	// Check that oracle_id is included
	if result.Data[0].OracleID != "oracle-123" {
		t.Errorf("expected oracle_id 'oracle-123', got '%s'", result.Data[0].OracleID)
	}

	if result.TotalCards != 50 {
		t.Errorf("expected TotalCards 50, got %d", result.TotalCards)
	}

	if !result.HasMore {
		t.Error("expected HasMore to be true")
	}

	if result.Page != 1 {
		t.Errorf("expected Page 1, got %d", result.Page)
	}
}

func TestSearch_MissingQuery(t *testing.T) {
	mock := &mockScryfallClient{}
	app := setupSearchTestApp(mock)

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result["error"] != "query parameter 'q' is required" {
		t.Errorf("unexpected error message: %s", result["error"])
	}
}

func TestSearch_Pagination(t *testing.T) {
	mock := &mockScryfallClient{
		searchFunc: func(ctx context.Context, query string, page int) (mockSearchResult, error) {
			if page != 3 {
				t.Errorf("expected page 3, got %d", page)
			}
			return mockSearchResult{
				Cards:      []scryfall.Card{{ID: "card-1", OracleID: "oracle-1", Name: "Test Card"}},
				TotalCards: 100,
				HasMore:    true,
				Page:       3,
			}, nil
		},
	}

	app := setupSearchTestApp(mock)

	req := httptest.NewRequest(http.MethodGet, "/search?q=test&page=3", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Page != 3 {
		t.Errorf("expected Page 3, got %d", result.Page)
	}
}

func TestSearch_InvalidPage(t *testing.T) {
	mock := &mockScryfallClient{
		searchFunc: func(ctx context.Context, query string, page int) (mockSearchResult, error) {
			if page != 1 {
				t.Errorf("expected page to be normalized to 1, got %d", page)
			}
			return mockSearchResult{Page: 1}, nil
		},
	}

	app := setupSearchTestApp(mock)

	req := httptest.NewRequest(http.MethodGet, "/search?q=test&page=-5", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestSearch_APIError(t *testing.T) {
	mock := &mockScryfallClient{
		searchFunc: func(ctx context.Context, query string, page int) (mockSearchResult, error) {
			return mockSearchResult{}, errors.New("scryfall api error")
		},
	}

	app := setupSearchTestApp(mock)

	req := httptest.NewRequest(http.MethodGet, "/search?q=test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected status %d, got %d", http.StatusBadGateway, resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result["error"] != "failed to search cards" {
		t.Errorf("unexpected error message: %s", result["error"])
	}
}

func TestSearch_CardWithImage(t *testing.T) {
	imageURL := "https://example.com/card.png"
	mock := &mockScryfallClient{
		searchFunc: func(ctx context.Context, query string, page int) (mockSearchResult, error) {
			return mockSearchResult{
				Cards: []scryfall.Card{
					{
						ID:       "card-1",
						OracleID: "oracle-1",
						Name:     "Test Card",
						ImageURIs: &scryfall.ImageURIs{
							PNG: imageURL,
						},
					},
				},
				TotalCards: 1,
				Page:       1,
			}, nil
		},
	}

	app := setupSearchTestApp(mock)

	req := httptest.NewRequest(http.MethodGet, "/search?q=test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result.Data) != 1 {
		t.Fatalf("expected 1 card, got %d", len(result.Data))
	}

	if result.Data[0].ImageURI == nil {
		t.Fatal("expected ImageURI to be set")
	}

	if *result.Data[0].ImageURI != imageURL {
		t.Errorf("expected ImageURI '%s', got '%s'", imageURL, *result.Data[0].ImageURI)
	}
}

func TestSearch_CardWithoutImage(t *testing.T) {
	mock := &mockScryfallClient{
		searchFunc: func(ctx context.Context, query string, page int) (mockSearchResult, error) {
			return mockSearchResult{
				Cards: []scryfall.Card{
					{ID: "card-1", OracleID: "oracle-1", Name: "Test Card"},
				},
				TotalCards: 1,
				Page:       1,
			}, nil
		},
	}

	app := setupSearchTestApp(mock)

	req := httptest.NewRequest(http.MethodGet, "/search?q=test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Data[0].ImageURI != nil {
		t.Error("expected ImageURI to be nil for card without image")
	}
}

func TestHasLanguageClause(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  bool
	}{
		{"short form", "l:de", true},
		{"long form", "lang:german", true},
		{"uppercase", "L:EN", true},
		{"mixed case lang", "LaNg:fr", true},
		{"negated short", "-l:en", true},
		{"negated long", "-lang:en", true},
		{"bang negated", "!l:en", true},
		{"with surrounding terms", "name:foo l:de set:lea", true},
		{"trailing", "lightning bolt l:de", true},
		{"leading whitespace", "  l:de", true},

		{"empty", "", false},
		{"plain text", "lightning bolt", false},
		{"legal: not a language", "legal:standard", false},
		{"loyalty: not a language", "loyalty:3", false},
		{"layout: not a language", "layout:normal", false},
		{"name with l in it", "name:lightning", false},
		{"colon in text", "foo:bar", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasLanguageClause(tc.query); got != tc.want {
				t.Errorf("hasLanguageClause(%q) = %v, want %v", tc.query, got, tc.want)
			}
		})
	}
}

func TestApplyLanguage(t *testing.T) {
	cases := []struct {
		name         string
		query        string
		explicit     string
		savedDefault string
		want         string
	}{
		// Saved-default fallback (no explicit override)
		{"no explicit, empty default is no-op", "lightning bolt", "", "", "lightning bolt"},
		{"no explicit, english default is no-op", "lightning bolt", "", "en", "lightning bolt"},
		{"no explicit, german default appended", "lightning bolt", "", "de", "lightning bolt l:de"},
		{"no explicit, french default appended", "lightning bolt", "", "fr", "lightning bolt l:fr"},

		// Explicit override (per-request dropdown choice)
		{"explicit english is appended (no longer a no-op)", "lightning bolt", "en", "de", "lightning bolt l:en"},
		{"explicit german wins over english default", "lightning bolt", "de", "en", "lightning bolt l:de"},
		{"explicit french wins over german default", "lightning bolt", "fr", "de", "lightning bolt l:fr"},
		{"explicit english appended even with empty default", "lightning bolt", "en", "", "lightning bolt l:en"},

		// Typed clause in query always wins
		{"existing l: clause beats explicit", "lightning bolt l:en", "de", "fr", "lightning bolt l:en"},
		{"existing lang: clause beats explicit", "lightning bolt lang:german", "fr", "en", "lightning bolt lang:german"},
		{"negated language clause beats explicit", "lightning bolt -l:en", "de", "fr", "lightning bolt -l:en"},

		// Non-language clauses don't false-match
		{"loyalty does not look like a language clause", "loyalty:3", "", "de", "loyalty:3 l:de"},
		{"legal does not look like a language clause", "legal:standard", "fr", "", "legal:standard l:fr"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := applyLanguage(tc.query, tc.explicit, tc.savedDefault); got != tc.want {
				t.Errorf("applyLanguage(%q, %q, %q) = %q, want %q",
					tc.query, tc.explicit, tc.savedDefault, got, tc.want)
			}
		})
	}
}

// fiber:context-methods migrated
