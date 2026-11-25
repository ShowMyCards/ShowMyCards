package services

import (
	"backend/models"
	scryfallclient "backend/scryfall"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"time"

	scryfall "github.com/BlueMonday/go-scryfall"
	"gorm.io/gorm"
)

const (
	// DefaultCardManifestURL is Scryfall's /cards/manifest endpoint.
	DefaultCardManifestURL = "https://api.scryfall.com/cards/manifest"

	// DefaultIncrementalFallbackThreshold triggers a full bulk import when
	// more than this many cards need hydration. 0 disables fallback.
	DefaultIncrementalFallbackThreshold = 5000

	// maxManifestPagesPerLang bounds pagination per language against a broken next_page loop.
	maxManifestPagesPerLang = 500
)

type manifestCardClient interface {
	GetManifestPage(ctx context.Context, pageURL string) (scryfallclient.ManifestPage, error)
	GetCollectionByIDs(ctx context.Context, ids []string) ([]scryfall.Card, []string, error)
}

// IncrementalSyncMetadata is stored as JSON in Job.Metadata for incremental card sync jobs.
type IncrementalSyncMetadata struct {
	Mode                   string   `json:"mode"` // "incremental" or "full_fallback"
	Phase                  string   `json:"phase"`
	FallbackReason         string   `json:"fallback_reason,omitempty"`
	ManifestPagesFetched   int      `json:"manifest_pages_fetched,omitempty"`
	ManifestEntriesScanned int      `json:"manifest_entries_scanned,omitempty"`
	NewCards               int      `json:"new_cards,omitempty"`
	UpdatedCards           int      `json:"updated_cards,omitempty"`
	UnchangedCards         int      `json:"unchanged_cards,omitempty"`
	NotFoundCards          int      `json:"not_found_cards,omitempty"`
	TotalCards             int      `json:"total_cards"`
	ProcessedCards         int      `json:"processed_cards"`
	FailedCards            int      `json:"failed_cards"`
	FailureExamples        []string `json:"failure_examples,omitempty"`
}

// IncrementalSyncService syncs new and updated cards using Scryfall's
// /cards/manifest and /cards/collection endpoints, falling back to full bulk
// import when the database is empty or the diff exceeds a configured threshold.
//
// Like BulkDataService, it uses an additive UPSERT policy and intentionally does
// not prune local cards absent from the manifest so inventory, list, and deck
// references keep their card data. Reconciling Scryfall /migrations deletions
// and merges is out of scope.
//
// Upgrade behaviour: existing rows created before sync timestamps were stored
// default to empty image_updated_at/data_updated_at, which diffing treats as
// changed. Consequently, the first incremental sync after upgrading counts cards
// on the first manifest page as updated, exceeds the fallback threshold, and
// runs a one-time full bulk import to stamp those columns; subsequent runs use
// the incremental path.
type IncrementalSyncService struct {
	db              *gorm.DB
	jobService      *JobService
	settingsService *SettingsService
	scryfallClient  manifestCardClient
	bulkDataService *BulkDataService
	languages       []string
}

// NewIncrementalSyncService creates a new incremental card sync service.
func NewIncrementalSyncService(
	db *gorm.DB,
	jobService *JobService,
	settingsService *SettingsService,
	scryfallClient manifestCardClient,
	bulkDataService *BulkDataService,
) *IncrementalSyncService {
	return &IncrementalSyncService{
		db:              db,
		jobService:      jobService,
		settingsService: settingsService,
		scryfallClient:  scryfallClient,
		bulkDataService: bulkDataService,
		languages:       append([]string(nil), scryfallclient.SupportedLanguages...),
	}
}

// CreateSyncJob creates a new job record for an incremental card sync.
func (s *IncrementalSyncService) CreateSyncJob(ctx context.Context) (*models.Job, error) {
	return s.jobService.Create(ctx, models.JobTypeIncrementalCardSync, "{}")
}

// Sync executes an incremental card sync for the given job ID.
func (s *IncrementalSyncService) Sync(ctx context.Context, jobID uint) error {
	if err := s.jobService.Start(ctx, jobID); err != nil {
		return fmt.Errorf("failed to start job: %w", err)
	}

	if err := s.settingsService.Set(ctx, "incremental_card_sync_last_status", TaskStatusInProgress); err != nil {
		slog.Warn("failed to update status setting", "key", "incremental_card_sync_last_status", "error", err)
	}

	if err := s.syncInternal(ctx, jobID); err != nil {
		// Use WithoutCancel so a cancelled sync context still persists the failed state.
		persistCtx := context.WithoutCancel(ctx)
		if failErr := s.jobService.Fail(persistCtx, jobID, err.Error()); failErr != nil {
			slog.Error("failed to mark job as failed", "job_id", jobID, "error", failErr)
		}
		if setErr := s.settingsService.Set(persistCtx, "incremental_card_sync_last_status", TaskStatusFailed); setErr != nil {
			slog.Warn("failed to update status setting", "key", "incremental_card_sync_last_status", "error", setErr)
		}
		if setErr := s.settingsService.SetTime(persistCtx, "incremental_card_sync_last_run", time.Now()); setErr != nil {
			slog.Warn("failed to update time setting", "key", "incremental_card_sync_last_run", "error", setErr)
		}
		return fmt.Errorf("incremental card sync job %d failed: %w", jobID, err)
	}

	if err := s.jobService.Complete(ctx, jobID); err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	if setErr := s.settingsService.Set(ctx, "incremental_card_sync_last_status", TaskStatusSuccess); setErr != nil {
		slog.Warn("failed to update status setting", "key", "incremental_card_sync_last_status", "error", setErr)
	}
	if setErr := s.settingsService.SetTime(ctx, "incremental_card_sync_last_run", time.Now()); setErr != nil {
		slog.Warn("failed to update time setting", "key", "incremental_card_sync_last_run", "error", setErr)
	}

	return nil
}

type manifestDiffKind int

const (
	diffUnchanged manifestDiffKind = iota
	diffNew
	diffUpdated
)

type manifestDiffItem struct {
	ID             string
	ImageUpdatedAt string
	DataUpdatedAt  string
}

func (s *IncrementalSyncService) syncInternal(ctx context.Context, jobID uint) error {
	hasData, err := s.bulkDataService.HasBulkData()
	if err != nil {
		return fmt.Errorf("checking for existing card data: %w", err)
	}
	if !hasData {
		slog.Info("local card database is empty, running full bulk import fallback", "component", "incremental_sync")
		return s.runFullFallback(ctx, jobID, IncrementalSyncMetadata{
			Mode:           "full_fallback",
			FallbackReason: "empty_database",
		})
	}

	s.updateJobMetadata(ctx, jobID, IncrementalSyncMetadata{
		Mode:  "incremental",
		Phase: "loading_local_state",
	})

	localState, err := models.GetCardSyncState(ctx, s.db)
	if err != nil {
		return fmt.Errorf("loading local card sync state: %w", err)
	}
	if len(localState) == 0 {
		slog.Info("local card database is empty, running full bulk import fallback", "component", "incremental_sync")
		return s.runFullFallback(ctx, jobID, IncrementalSyncMetadata{
			Mode:           "full_fallback",
			FallbackReason: "empty_database",
		})
	}

	manifestURL, err := s.settingsService.Get(ctx, "card_manifest_url")
	if err != nil || manifestURL == "" {
		manifestURL = DefaultCardManifestURL
	}

	fallbackThreshold := s.settingsService.GetInt(ctx, "incremental_card_sync_fallback_threshold", DefaultIncrementalFallbackThreshold)
	if fallbackThreshold < 0 {
		fallbackThreshold = 0
	}

	diff, err := s.scanManifestDiff(ctx, jobID, manifestURL, fallbackThreshold, localState)
	if err != nil {
		return fmt.Errorf("scanning manifest diff: %w", err)
	}

	if diff.fallbackTriggered {
		reason := fmt.Sprintf("diff_exceeded_threshold (%d > %d)", len(diff.toHydrate), fallbackThreshold)
		slog.Info("incremental diff exceeded threshold, falling back to full bulk import",
			"component", "incremental_sync", "diff_count", len(diff.toHydrate), "threshold", fallbackThreshold)
		return s.runFullFallback(ctx, jobID, IncrementalSyncMetadata{
			Mode:                   "full_fallback",
			FallbackReason:         reason,
			ManifestPagesFetched:   diff.pagesFetched,
			ManifestEntriesScanned: diff.entriesScanned,
			NewCards:               diff.newCards,
			UpdatedCards:           diff.updatedCards,
			UnchangedCards:         diff.unchangedCards,
		})
	}

	if len(diff.toHydrate) == 0 {
		slog.Info("incremental sync found no new or updated cards",
			"component", "incremental_sync", "pages", diff.pagesFetched, "scanned", diff.entriesScanned)
		s.updateJobMetadata(ctx, jobID, IncrementalSyncMetadata{
			Mode:                   "incremental",
			Phase:                  "completed",
			ManifestPagesFetched:   diff.pagesFetched,
			ManifestEntriesScanned: diff.entriesScanned,
			UnchangedCards:         diff.unchangedCards,
		})
		return nil
	}

	meta, err := s.hydrateCards(ctx, jobID, diff)
	if err != nil {
		return err
	}

	meta.Phase = "completed"
	s.updateJobMetadata(ctx, jobID, meta)

	if err := checkSyncFailureRate(meta); err != nil {
		return err
	}

	if meta.ProcessedCards > 0 {
		if _, err := models.ReclassifyDeckExtras(ctx, s.db); err != nil {
			slog.Error("failed to reclassify deck extras after incremental sync", "error", err)
		}
	}

	slog.Info("incremental sync completed",
		"component", "incremental_sync",
		"new", meta.NewCards,
		"updated", meta.UpdatedCards,
		"unchanged", meta.UnchangedCards,
		"processed", meta.ProcessedCards,
		"not_found", meta.NotFoundCards,
		"failed", meta.FailedCards,
	)

	return nil
}

type manifestDiffResult struct {
	toHydrate         []manifestDiffItem
	manifestByID      map[string]manifestDiffItem
	pagesFetched      int
	entriesScanned    int
	newCards          int
	updatedCards      int
	unchangedCards    int
	fallbackTriggered bool
}

func (s *IncrementalSyncService) scanManifestDiff(
	ctx context.Context,
	jobID uint,
	manifestURL string,
	fallbackThreshold int,
	localState map[string]models.CardSyncState,
) (manifestDiffResult, error) {
	seenIDs := make(map[string]struct{}, len(localState))
	res := manifestDiffResult{
		manifestByID: make(map[string]manifestDiffItem),
	}

	for _, lang := range s.languages {
		if err := s.scanLanguageManifest(ctx, jobID, manifestURL, lang, fallbackThreshold, localState, seenIDs, &res); err != nil {
			return manifestDiffResult{}, err
		}
		if res.fallbackTriggered {
			return res, nil
		}
	}

	return res, nil
}

func (s *IncrementalSyncService) scanLanguageManifest(
	ctx context.Context,
	jobID uint,
	manifestURL, lang string,
	fallbackThreshold int,
	localState map[string]models.CardSyncState,
	seenIDs map[string]struct{},
	res *manifestDiffResult,
) error {
	pageURL, err := buildManifestURL(manifestURL, lang, 1)
	if err != nil {
		return fmt.Errorf("building manifest URL for lang %q: %w", lang, err)
	}

	pagesForLang := 0
	for pageURL != "" {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("sync cancelled: %w", err)
		}
		pagesForLang++
		if pagesForLang > maxManifestPagesPerLang {
			return fmt.Errorf("manifest pagination exceeded %d pages for language %q", maxManifestPagesPerLang, lang)
		}

		page, err := s.scryfallClient.GetManifestPage(ctx, pageURL)
		if err != nil {
			return fmt.Errorf("fetching manifest page (lang=%s, page=%d): %w", lang, pagesForLang, err)
		}
		res.pagesFetched++

		diffManifestEntries(page.Data, localState, seenIDs, res)

		s.updateJobMetadata(ctx, jobID, IncrementalSyncMetadata{
			Mode:                   "incremental",
			Phase:                  "fetching_manifest",
			ManifestPagesFetched:   res.pagesFetched,
			ManifestEntriesScanned: res.entriesScanned,
			NewCards:               res.newCards,
			UpdatedCards:           res.updatedCards,
			UnchangedCards:         res.unchangedCards,
		})

		if fallbackThreshold > 0 && len(res.toHydrate) > fallbackThreshold {
			res.fallbackTriggered = true
			return nil
		}

		if !page.HasMore || page.NextPage == "" {
			return nil
		}
		if page.NextPage == pageURL {
			return fmt.Errorf("manifest pagination loop detected at %s", pageURL)
		}
		pageURL = page.NextPage
	}

	return nil
}

func diffManifestEntries(
	entries []scryfallclient.ManifestEntry,
	localState map[string]models.CardSyncState,
	seenIDs map[string]struct{},
	res *manifestDiffResult,
) {
	for _, entry := range entries {
		if entry.ID == "" {
			continue
		}
		if _, dup := seenIDs[entry.ID]; dup {
			continue
		}
		seenIDs[entry.ID] = struct{}{}
		res.entriesScanned++

		local, exists := localState[entry.ID]
		switch classifyManifestEntry(entry, local, exists) {
		case diffNew:
			res.newCards++
			item := toDiffItem(entry)
			res.toHydrate = append(res.toHydrate, item)
			res.manifestByID[entry.ID] = item
		case diffUpdated:
			res.updatedCards++
			item := toDiffItem(entry)
			res.toHydrate = append(res.toHydrate, item)
			res.manifestByID[entry.ID] = item
		case diffUnchanged:
			res.unchangedCards++
		}
	}
}

func (s *IncrementalSyncService) hydrateCards(
	ctx context.Context,
	jobID uint,
	diff manifestDiffResult,
) (IncrementalSyncMetadata, error) {
	meta := IncrementalSyncMetadata{
		Mode:                   "incremental",
		Phase:                  "hydrating_cards",
		ManifestPagesFetched:   diff.pagesFetched,
		ManifestEntriesScanned: diff.entriesScanned,
		NewCards:               diff.newCards,
		UpdatedCards:           diff.updatedCards,
		UnchangedCards:         diff.unchangedCards,
		TotalCards:             len(diff.toHydrate),
		FailureExamples:        make([]string, 0, 10),
	}
	s.updateJobMetadata(ctx, jobID, meta)

	for start := 0; start < len(diff.toHydrate); start += scryfallclient.MaxCollectionBatchSize {
		if err := ctx.Err(); err != nil {
			return meta, fmt.Errorf("sync cancelled: %w", err)
		}

		end := start + scryfallclient.MaxCollectionBatchSize
		if end > len(diff.toHydrate) {
			end = len(diff.toHydrate)
		}

		ids := make([]string, end-start)
		for i, item := range diff.toHydrate[start:end] {
			ids[i] = item.ID
		}

		cards, notFoundIDs, err := s.scryfallClient.GetCollectionByIDs(ctx, ids)
		if err != nil {
			return meta, fmt.Errorf("hydrating cards: %w", err)
		}

		if len(notFoundIDs) > 0 {
			meta.NotFoundCards += len(notFoundIDs)
			sample := notFoundIDs
			if len(sample) > 5 {
				sample = sample[:5]
			}
			slog.Warn("cards in manifest were not found via /cards/collection",
				"component", "incremental_sync", "count", len(notFoundIDs), "sample_ids", sample)
		}

		dbCards := convertHydratedCards(cards, diff.manifestByID, &meta)
		if len(dbCards) > 0 {
			if err := upsertCardsBatch(ctx, s.db, dbCards); err != nil {
				return meta, fmt.Errorf("upserting hydrated cards: %w", err)
			}
			meta.ProcessedCards += len(dbCards)
		}

		s.updateJobMetadata(ctx, jobID, meta)
	}

	return meta, nil
}

func convertHydratedCards(
	cards []scryfall.Card,
	manifestByID map[string]manifestDiffItem,
	meta *IncrementalSyncMetadata,
) []*models.Card {
	dbCards := make([]*models.Card, 0, len(cards))
	for _, sc := range cards {
		card, err := models.FromScryfallCard(sc)
		if err != nil {
			meta.FailedCards++
			if len(meta.FailureExamples) < 10 {
				msg := fmt.Sprintf("Card %s (%s): %v", sc.ID, sc.Name, err)
				if len(msg) > 100 {
					msg = msg[:97] + "..."
				}
				meta.FailureExamples = append(meta.FailureExamples, msg)
			}
			slog.Warn("failed to convert hydrated card", "scryfall_id", sc.ID, "name", sc.Name, "error", err)
			continue
		}

		if item, ok := manifestByID[sc.ID]; ok {
			if card.ImageUpdatedAt == "" && item.ImageUpdatedAt != "" {
				card.ImageUpdatedAt = normalizeTimestampString(item.ImageUpdatedAt)
			}
			if item.DataUpdatedAt != "" {
				card.DataUpdatedAt = normalizeTimestampString(item.DataUpdatedAt)
			}
		}

		dbCards = append(dbCards, card)
	}
	return dbCards
}

func checkSyncFailureRate(meta IncrementalSyncMetadata) error {
	attempted := meta.ProcessedCards + meta.FailedCards
	if attempted == 0 {
		return nil
	}

	failureRate := float64(meta.FailedCards) / float64(attempted)
	if failureRate > BulkDataMaxFailureRate {
		return fmt.Errorf("incremental sync failed: %d/%d cards failed (%.2f%% > %.2f%% threshold)",
			meta.FailedCards, attempted, failureRate*100, BulkDataMaxFailureRate*100)
	}
	if meta.FailedCards > 0 {
		slog.Warn("incremental sync completed with warnings",
			"failed", meta.FailedCards, "attempted", attempted, "failure_rate_pct", fmt.Sprintf("%.2f", failureRate*100))
	}
	return nil
}

func (s *IncrementalSyncService) runFullFallback(ctx context.Context, jobID uint, baseMeta IncrementalSyncMetadata) error {
	result, err := s.bulkDataService.streamBulkImport(ctx, func(p BulkImportProgress) {
		meta := baseMeta
		meta.Phase = p.Phase
		meta.ProcessedCards = p.ProcessedCards
		meta.FailedCards = p.FailedCards
		meta.FailureExamples = p.FailureExamples
		s.updateJobMetadata(ctx, jobID, meta)
	})

	if result.TotalCards > 0 || err == nil {
		meta := baseMeta
		meta.Phase = "completed"
		meta.TotalCards = result.TotalCards
		meta.ProcessedCards = result.ProcessedCards
		meta.FailedCards = result.FailedCards
		meta.FailureExamples = result.FailureExamples
		s.updateJobMetadata(ctx, jobID, meta)
	}

	if err != nil {
		return fmt.Errorf("full bulk fallback: %w", err)
	}
	return nil
}

func (s *IncrementalSyncService) updateJobMetadata(ctx context.Context, jobID uint, metadata IncrementalSyncMetadata) {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		slog.Warn("failed to marshal incremental sync job metadata", "error", err)
		return
	}

	if err := s.jobService.UpdateMetadata(ctx, jobID, string(metadataJSON)); err != nil {
		slog.Warn("failed to update incremental sync job metadata", "error", err)
	}
}

func buildManifestURL(baseURL, lang string, page int) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	if lang != "" {
		q.Set("lang", lang)
	}
	q.Set("page", strconv.Itoa(page))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func toDiffItem(entry scryfallclient.ManifestEntry) manifestDiffItem {
	item := manifestDiffItem{ID: entry.ID}
	if entry.ImageUpdatedAt != nil {
		item.ImageUpdatedAt = *entry.ImageUpdatedAt
	}
	if entry.DataUpdatedAt != nil {
		item.DataUpdatedAt = *entry.DataUpdatedAt
	}
	return item
}

func classifyManifestEntry(entry scryfallclient.ManifestEntry, local models.CardSyncState, exists bool) manifestDiffKind {
	if !exists {
		return diffNew
	}
	if entry.ImageUpdatedAt != nil && *entry.ImageUpdatedAt != "" && imageTimestampChanged(local.ImageUpdatedAt, *entry.ImageUpdatedAt) {
		return diffUpdated
	}
	if entry.DataUpdatedAt != nil && *entry.DataUpdatedAt != "" && dataTimestampNewer(local.DataUpdatedAt, *entry.DataUpdatedAt) {
		return diffUpdated
	}
	return diffUnchanged
}

// imageTimestampChanged treats an empty local timestamp (e.g. pre-upgrade rows)
// as changed so the card is refreshed or triggers the one-time full fallback.
func imageTimestampChanged(local, remote string) bool {
	if local == "" {
		return true
	}
	lt, lErr := time.Parse(time.RFC3339Nano, local)
	rt, rErr := time.Parse(time.RFC3339Nano, remote)
	if lErr == nil && rErr == nil {
		return !lt.Equal(rt)
	}
	return local != remote
}

// dataTimestampNewer treats an empty local timestamp (e.g. pre-upgrade rows) as
// needing an update once Scryfall populates data_updated_at on the manifest.
func dataTimestampNewer(local, remote string) bool {
	if local == "" {
		return true
	}
	lt, lErr := time.Parse(time.RFC3339Nano, local)
	rt, rErr := time.Parse(time.RFC3339Nano, remote)
	if lErr == nil && rErr == nil {
		return rt.After(lt)
	}
	return local != remote
}

func normalizeTimestampString(ts string) string {
	if ts == "" {
		return ""
	}
	if parsed, err := time.Parse(time.RFC3339Nano, ts); err == nil {
		return parsed.UTC().Format(time.RFC3339)
	}
	return ts
}
