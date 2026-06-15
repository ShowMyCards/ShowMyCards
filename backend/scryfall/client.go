package scryfall

import (
	"context"
	"log/slog"
	"time"

	"github.com/BlueMonday/go-scryfall"
	"github.com/TwiN/gocache/v2"
	"go.uber.org/ratelimit"
)

const (
	// CacheTTL is the time-to-live for cached cards (24 hours for pricing validity)
	CacheTTL = 24 * time.Hour

	// DefaultAPITimeout is the default timeout for Scryfall API calls
	DefaultAPITimeout = 30 * time.Second

	// ScryfallRPS bounds the Scryfall request rate.
	// The search endpoint is rate limited to 2 req/s
	// (even if the error message when hitting the rate limiter
	// claims a limit of 10 req/s).
	// See https://scryfall.com/docs/api/rate-limits for actual
	// rate limits.
	// Setting the limit to 2 req/s still results in 429 on bulk imports
	// so we are very conservative here.
	ScryfallRPS = 1
)

// ScryfallAPI defines the interface for Scryfall API operations
type ScryfallAPI interface {
	SearchCards(ctx context.Context, query string, opts scryfall.SearchCardsOptions) (scryfall.CardListResponse, error)
	GetCard(ctx context.Context, id string) (scryfall.Card, error)
	ListSets(ctx context.Context) ([]scryfall.Set, error)
	ListCardSymbols(ctx context.Context) ([]scryfall.CardSymbol, error)
	AutocompleteCard(ctx context.Context, s string) ([]string, error)
}

// Client wraps the Scryfall API client with caching
type Client struct {
	api   ScryfallAPI
	cache *gocache.Cache
}

// NewClient creates a new Scryfall client with caching
func NewClient() (*Client, error) {
	limiter := ratelimit.New(ScryfallRPS, ratelimit.WithoutSlack)
	api, err := scryfall.NewClient(scryfall.WithLimiter(limiter))
	if err != nil {
		return nil, err
	}

	return newClientWithAPI(api), nil
}

// newClientWithAPI creates a client with a specific API implementation (for testing)
func newClientWithAPI(api ScryfallAPI) *Client {
	cache := gocache.NewCache().WithMaxSize(10000)
	cache.StartJanitor()

	return &Client{
		api:   api,
		cache: cache,
	}
}

// Close stops the cache janitor
func (c *Client) Close() {
	c.cache.StopJanitor()
}

// SearchResult contains paginated search results
type SearchResult struct {
	Cards      []scryfall.Card
	TotalCards int
	HasMore    bool
	Page       int
}

// SearchOptions contains options for searching cards
type SearchOptions struct {
	Page       int
	UniqueMode scryfall.UniqueMode
}

// Search searches Scryfall for cards matching the query string.
// Results are cached by their Scryfall ID.
func (c *Client) Search(ctx context.Context, query string, page int) (SearchResult, error) {
	return c.SearchWithOptions(ctx, query, SearchOptions{Page: page})
}

// SearchWithOptions searches Scryfall for cards with custom options.
// Results are cached by their Scryfall ID.
func (c *Client) SearchWithOptions(ctx context.Context, query string, opts SearchOptions) (SearchResult, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultAPITimeout)
	defer cancel()

	if opts.Page < 1 {
		opts.Page = 1
	}

	searchOpts := scryfall.SearchCardsOptions{
		Page: opts.Page,
	}

	// Only set unique mode if it's not the zero value
	if opts.UniqueMode != "" {
		searchOpts.Unique = opts.UniqueMode
	}

	result, err := c.api.SearchCards(ctx, query, searchOpts)
	if err != nil {
		slog.Error("search failed", "component", "scryfall", "query", query, "page", opts.Page, "error", err)
		return SearchResult{}, err
	}

	// Cache each card by its Scryfall ID
	for _, card := range result.Cards {
		c.cache.SetWithTTL(card.ID, card, CacheTTL)
	}

	return SearchResult{
		Cards:      result.Cards,
		TotalCards: result.TotalCards,
		HasMore:    result.HasMore,
		Page:       opts.Page,
	}, nil
}

// ListSets retrieves all sets from Scryfall.
func (c *Client) ListSets(ctx context.Context) ([]scryfall.Set, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultAPITimeout)
	defer cancel()

	return c.api.ListSets(ctx)
}

// ListSymbols retrieves all card symbols (mana, tap, etc.) from Scryfall.
func (c *Client) ListSymbols(ctx context.Context) ([]scryfall.CardSymbol, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultAPITimeout)
	defer cancel()

	return c.api.ListCardSymbols(ctx)
}

// Autocomplete returns card name suggestions for a partial query.
func (c *Client) Autocomplete(ctx context.Context, query string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultAPITimeout)
	defer cancel()

	return c.api.AutocompleteCard(ctx, query)
}

// GetByID retrieves a card by its Scryfall ID.
// Returns a cached version if available, otherwise fetches from the API and caches it.
func (c *Client) GetByID(ctx context.Context, id string) (scryfall.Card, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultAPITimeout)
	defer cancel()

	// Check cache first
	if cached, ok := c.cache.Get(id); ok {
		if card, ok := cached.(scryfall.Card); ok {
			return card, nil
		}
		slog.Warn("cache type mismatch, refetching", "component", "scryfall", "id", id)
	}

	startTime := time.Now()

	// Fetch from API
	card, err := c.api.GetCard(ctx, id)

	duration := time.Since(startTime)
	if err != nil {
		slog.Error("GetByID failed", "component", "scryfall", "duration", duration, "id", id, "error", err)
		return scryfall.Card{}, err
	}

	// Cache the result
	c.cache.SetWithTTL(card.ID, card, CacheTTL)

	return card, nil
}
