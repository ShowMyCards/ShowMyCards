package scryfall

import (
	"backend/version"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ManifestEntry represents a single card entry returned by Scryfall's /cards/manifest endpoint.
type ManifestEntry struct {
	ID              string  `json:"id"`
	OracleID        string  `json:"oracle_id"`
	Name            string  `json:"name"`
	SetCode         string  `json:"set_code"`
	CollectorNumber string  `json:"collector_number"`
	Lang            string  `json:"lang"`
	CreatedAt       *string `json:"created_at"`
	DataUpdatedAt   *string `json:"data_updated_at"`
	ImageUpdatedAt  *string `json:"image_updated_at"`
}

// ManifestPage represents one paginated response from GET /cards/manifest.
type ManifestPage struct {
	Object     string          `json:"object"`
	TotalCards int             `json:"total_cards"`
	HasMore    bool            `json:"has_more"`
	NextPage   string          `json:"next_page"`
	Data       []ManifestEntry `json:"data"`
}

type rateLimitError struct {
	wait   time.Duration
	status int
	body   string
}

func (e *rateLimitError) Error() string {
	return fmt.Sprintf("manifest endpoint returned status %d: %s", e.status, e.body)
}

// GetManifestPage fetches a single page from Scryfall's /cards/manifest endpoint,
// pacing requests to respect the 10/minute limit and retrying on HTTP 429.
func (c *Client) GetManifestPage(ctx context.Context, pageURL string) (ManifestPage, error) {
	for attempt := 0; attempt <= maxRateLimitRetries; attempt++ {
		page, err := c.fetchManifestPageOnce(ctx, pageURL)
		if err == nil {
			return page, nil
		}

		var rlErr *rateLimitError
		if errors.As(err, &rlErr) && attempt < maxRateLimitRetries {
			slog.Warn("scryfall manifest rate limit hit, backing off",
				"component", "scryfall", "attempt", attempt+1, "retry_after", rlErr.wait)
			if sleepErr := sleepWithContext(ctx, rlErr.wait); sleepErr != nil {
				return ManifestPage{}, sleepErr
			}
			continue
		}

		return ManifestPage{}, err
	}

	return ManifestPage{}, fmt.Errorf("fetching manifest page %s: exhausted rate limit retries", pageURL)
}

func (c *Client) fetchManifestPageOnce(ctx context.Context, pageURL string) (ManifestPage, error) {
	if err := c.waitManifestSlot(ctx); err != nil {
		return ManifestPage{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return ManifestPage{}, fmt.Errorf("creating manifest request: %w", err)
	}
	req.Header.Set("User-Agent", version.UserAgent())
	req.Header.Set("Accept", "application/json")

	resp, err := c.manifestHTTPClient.Do(req)
	if err != nil {
		return ManifestPage{}, fmt.Errorf("fetching manifest page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		body := readErrorBody(resp.Body)
		wait := c.retryDelay
		if rawRetry := strings.TrimSpace(resp.Header.Get("Retry-After")); rawRetry != "" {
			if secs, convErr := strconv.Atoi(rawRetry); convErr == nil && secs > 0 {
				if parsed := time.Duration(secs) * time.Second; parsed > wait {
					wait = parsed
				}
			}
		}
		return ManifestPage{}, &rateLimitError{
			wait:   wait,
			status: resp.StatusCode,
			body:   body,
		}
	}

	if resp.StatusCode != http.StatusOK {
		body := readErrorBody(resp.Body)
		return ManifestPage{}, fmt.Errorf("manifest request returned status %d: %s", resp.StatusCode, body)
	}

	var page ManifestPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return ManifestPage{}, fmt.Errorf("decoding manifest page: %w", err)
	}

	return page, nil
}

func (c *Client) waitManifestSlot(ctx context.Context) error {
	c.manifestMu.Lock()
	defer c.manifestMu.Unlock()

	if c.manifestInterval > 0 && !c.lastManifestReq.IsZero() {
		if wait := c.manifestInterval - time.Since(c.lastManifestReq); wait > 0 {
			if err := sleepWithContext(ctx, wait); err != nil {
				return err
			}
		}
	}

	c.lastManifestReq = time.Now()
	return nil
}

func readErrorBody(r io.Reader) string {
	body, err := io.ReadAll(io.LimitReader(r, 1024))
	if err != nil {
		slog.Warn("failed to read scryfall error response body", "component", "scryfall", "error", err)
		return ""
	}
	return string(body)
}
