package scryfall

import (
	"backend/version"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
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

	// ManifestAPITimeout is the timeout for Scryfall /cards/manifest page requests,
	// which return up to 15,000 entries (~2.5 MB) per page.
	ManifestAPITimeout = 60 * time.Second

	// ScryfallRPS bounds the Scryfall request rate.
	// The search endpoint is rate limited to 2 req/s
	// (even if the error message when hitting the rate limiter
	// claims a limit of 10 req/s).
	// See https://scryfall.com/docs/api/rate-limits for actual
	// rate limits.
	// Setting the limit to 2 req/s still results in 429 on bulk imports
	// so we are very conservative here.
	ScryfallRPS = 1

	// MaxCollectionBatchSize is the maximum number of identifiers accepted by POST /cards/collection.
	MaxCollectionBatchSize = 75

	// ManifestRequestInterval spaces GET /cards/manifest requests to respect
	// Scryfall's 10 requests/minute (6,000ms) rate limit with a small safety margin.
	ManifestRequestInterval = 6100 * time.Millisecond

	// DefaultRateLimitRetryAfter is how long we wait after a 429 response
	// (Scryfall enforces a 30-second cooldown).
	DefaultRateLimitRetryAfter = 35 * time.Second

	maxRateLimitRetries = 2
)

// SupportedLanguages lists the 11 card languages supported by the application
// (matching frontend SCRYFALL_LANGUAGES).
var SupportedLanguages = []string{"en", "de", "es", "fr", "it", "ja", "ko", "pt", "ru", "zhs", "zht"}

// ScryfallAPI defines the interface for Scryfall API operations
type ScryfallAPI interface {
	SearchCards(ctx context.Context, query string, opts scryfall.SearchCardsOptions) (scryfall.CardListResponse, error)
	GetCard(ctx context.Context, id string) (scryfall.Card, error)
	GetCardsByIdentifiers(ctx context.Context, identifiers []scryfall.CardIdentifier) (scryfall.GetCardsByIdentifiersResponse, error)
	ListSets(ctx context.Context) ([]scryfall.Set, error)
	ListCardSymbols(ctx context.Context) ([]scryfall.CardSymbol, error)
	AutocompleteCard(ctx context.Context, s string) ([]string, error)
}

// Client wraps the Scryfall API client with caching
type Client struct {
	api                ScryfallAPI
	cache              *gocache.Cache
	manifestHTTPClient *http.Client
	manifestInterval   time.Duration
	retryDelay         time.Duration
	manifestMu         sync.Mutex
	lastManifestReq    time.Time
}

// NewClient creates a new Scryfall client with caching
func NewClient() (*Client, error) {
	api, err := newAPI()
	if err != nil {
		return nil, err
	}

	return newClientWithAPI(api), nil
}

// newAPI builds the underlying go-scryfall client with our rate limiter and
// User-Agent. Scryfall rejects generic library User-Agents (the go-scryfall
// default is "go-scryfall/<version>"), so we must identify ShowMyCards.
// Extra options are appended last (tests use this to set a stub base URL).
func newAPI(extra ...scryfall.ClientOption) (*scryfall.Client, error) {
	limiter := ratelimit.New(ScryfallRPS, ratelimit.WithoutSlack)
	opts := append([]scryfall.ClientOption{
		scryfall.WithLimiter(limiter),
		scryfall.WithUserAgent(version.UserAgent()),
	}, extra...)
	return scryfall.NewClient(opts...)
}

// newClientWithAPI creates a client with a specific API implementation (for testing)
func newClientWithAPI(api ScryfallAPI) *Client {
	cache := gocache.NewCache().WithMaxSize(10000)
	cache.StartJanitor()

	return &Client{
		api:                api,
		cache:              cache,
		manifestHTTPClient: &http.Client{Timeout: ManifestAPITimeout},
		manifestInterval:   ManifestRequestInterval,
		retryDelay:         DefaultRateLimitRetryAfter,
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

// GetCollectionByIDs fetches up to MaxCollectionBatchSize cards by Scryfall ID
// via POST /cards/collection. Returns the fetched cards and any IDs Scryfall
// reported in not_found. Does not populate the interactive search cache so
// background syncs don't evict hot search entries.
func (c *Client) GetCollectionByIDs(ctx context.Context, ids []string) ([]scryfall.Card, []string, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}
	if len(ids) > MaxCollectionBatchSize {
		return nil, nil, fmt.Errorf("collection batch size %d exceeds maximum %d", len(ids), MaxCollectionBatchSize)
	}

	identifiers := make([]scryfall.CardIdentifier, len(ids))
	for i, id := range ids {
		identifiers[i] = scryfall.CardIdentifier{ID: id}
	}

	for attempt := 0; attempt <= maxRateLimitRetries; attempt++ {
		reqCtx, cancel := context.WithTimeout(ctx, DefaultAPITimeout)
		resp, err := c.api.GetCardsByIdentifiers(reqCtx, identifiers)
		cancel()

		if err == nil {
			var notFound []string
			for _, nf := range resp.NotFound {
				if nf.ID != "" {
					notFound = append(notFound, nf.ID)
				}
			}
			return resp.Data, notFound, nil
		}

		var scryfallErr *scryfall.Error
		if errors.As(err, &scryfallErr) && scryfallErr.Status == http.StatusTooManyRequests && attempt < maxRateLimitRetries {
			slog.Warn("scryfall rate limit hit on collection fetch, backing off",
				"component", "scryfall", "attempt", attempt+1, "retry_after", c.retryDelay)
			if sleepErr := sleepWithContext(ctx, c.retryDelay); sleepErr != nil {
				return nil, nil, sleepErr
			}
			continue
		}

		slog.Error("GetCollectionByIDs failed", "component", "scryfall", "batch_size", len(ids), "error", err)
		return nil, nil, fmt.Errorf("fetching card collection batch: %w", err)
	}

	return nil, nil, fmt.Errorf("fetching card collection batch: exhausted rate limit retries")
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
