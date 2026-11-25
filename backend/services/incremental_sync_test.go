package services

import (
	"backend/models"
	scryfallclient "backend/scryfall"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	scryfall "github.com/BlueMonday/go-scryfall"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeManifestClient struct {
	getManifestPageFunc    func(ctx context.Context, pageURL string) (scryfallclient.ManifestPage, error)
	getCollectionByIDsFunc func(ctx context.Context, ids []string) ([]scryfall.Card, []string, error)
	manifestCalls          []string
	collectionBatches      [][]string
}

func (f *fakeManifestClient) GetManifestPage(ctx context.Context, pageURL string) (scryfallclient.ManifestPage, error) {
	f.manifestCalls = append(f.manifestCalls, pageURL)
	if f.getManifestPageFunc != nil {
		return f.getManifestPageFunc(ctx, pageURL)
	}
	return scryfallclient.ManifestPage{}, nil
}

func (f *fakeManifestClient) GetCollectionByIDs(ctx context.Context, ids []string) ([]scryfall.Card, []string, error) {
	copied := append([]string(nil), ids...)
	f.collectionBatches = append(f.collectionBatches, copied)
	if f.getCollectionByIDsFunc != nil {
		return f.getCollectionByIDsFunc(ctx, ids)
	}
	return nil, nil, nil
}

func setupIncrementalSyncTest(t *testing.T, client manifestCardClient) (*IncrementalSyncService, *BulkDataService, *JobService, *SettingsService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}

	if err := db.AutoMigrate(
		&models.Job{},
		&models.Setting{},
		&models.Card{},
		&models.Deck{},
		&models.DeckItem{},
		&models.Set{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	jobService := NewJobService(db)
	settingsService := NewSettingsService(db)
	bulkDataService := NewBulkDataService(db, jobService, settingsService)
	syncService := NewIncrementalSyncService(db, jobService, settingsService, client, bulkDataService)

	return syncService, bulkDataService, jobService, settingsService, db
}

func strPtr(s string) *string { return &s }

func seedLocalCard(t *testing.T, db *gorm.DB, id, name, imageUpdatedAt, dataUpdatedAt string) {
	t.Helper()
	card := models.Card{
		ScryfallID:     id,
		OracleID:       "oracle-" + id,
		RawJSON:        fmt.Sprintf(`{"id":%q,"name":%q,"set":"tst","collector_number":"1","lang":"en"}`, id, name),
		ImageUpdatedAt: imageUpdatedAt,
		DataUpdatedAt:  dataUpdatedAt,
	}
	if err := db.Create(&card).Error; err != nil {
		t.Fatalf("failed to seed card %s: %v", id, err)
	}
}

func TestClassifyManifestEntry(t *testing.T) {
	tests := []struct {
		name   string
		entry  scryfallclient.ManifestEntry
		local  models.CardSyncState
		exists bool
		want   manifestDiffKind
	}{
		{
			name:   "new card missing locally",
			entry:  scryfallclient.ManifestEntry{ID: "new-1", ImageUpdatedAt: strPtr("2026-10-10T13:01:28Z")},
			exists: false,
			want:   diffNew,
		},
		{
			name:   "unchanged when image_updated_at matches and data_updated_at nil",
			entry:  scryfallclient.ManifestEntry{ID: "c1", ImageUpdatedAt: strPtr("2026-10-10T13:01:28Z"), DataUpdatedAt: nil},
			local:  models.CardSyncState{ImageUpdatedAt: "2026-10-10T13:01:28Z"},
			exists: true,
			want:   diffUnchanged,
		},
		{
			name:   "unchanged when timestamps match across timezone offsets",
			entry:  scryfallclient.ManifestEntry{ID: "c1", ImageUpdatedAt: strPtr("2026-10-10T15:01:28+02:00")},
			local:  models.CardSyncState{ImageUpdatedAt: "2026-10-10T13:01:28Z"},
			exists: true,
			want:   diffUnchanged,
		},
		{
			name:   "unchanged when manifest image_updated_at is nil and data_updated_at is nil",
			entry:  scryfallclient.ManifestEntry{ID: "c1"},
			local:  models.CardSyncState{ImageUpdatedAt: ""},
			exists: true,
			want:   diffUnchanged,
		},
		{
			name:   "updated when image_updated_at changed",
			entry:  scryfallclient.ManifestEntry{ID: "c1", ImageUpdatedAt: strPtr("2026-11-01T00:00:00Z")},
			local:  models.CardSyncState{ImageUpdatedAt: "2026-10-10T13:01:28Z"},
			exists: true,
			want:   diffUpdated,
		},
		{
			name:   "updated when local image_updated_at was empty and remote is populated",
			entry:  scryfallclient.ManifestEntry{ID: "c1", ImageUpdatedAt: strPtr("2026-10-10T13:01:28Z")},
			local:  models.CardSyncState{ImageUpdatedAt: ""},
			exists: true,
			want:   diffUpdated,
		},
		{
			name:   "updated when data_updated_at is newer than local",
			entry:  scryfallclient.ManifestEntry{ID: "c1", ImageUpdatedAt: strPtr("2026-10-10T13:01:28Z"), DataUpdatedAt: strPtr("2026-11-26T00:00:00Z")},
			local:  models.CardSyncState{ImageUpdatedAt: "2026-10-10T13:01:28Z", DataUpdatedAt: "2026-11-25T10:00:00Z"},
			exists: true,
			want:   diffUpdated,
		},
		{
			name:   "unchanged when data_updated_at is older or equal to local bulk timestamp",
			entry:  scryfallclient.ManifestEntry{ID: "c1", ImageUpdatedAt: strPtr("2026-10-10T13:01:28Z"), DataUpdatedAt: strPtr("2026-11-24T00:00:00Z")},
			local:  models.CardSyncState{ImageUpdatedAt: "2026-10-10T13:01:28Z", DataUpdatedAt: "2026-11-25T10:00:00Z"},
			exists: true,
			want:   diffUnchanged,
		},
		{
			name:   "updated when data_updated_at is populated and local data_updated_at is empty",
			entry:  scryfallclient.ManifestEntry{ID: "c1", DataUpdatedAt: strPtr("2026-11-25T10:00:00Z")},
			local:  models.CardSyncState{},
			exists: true,
			want:   diffUpdated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyManifestEntry(tt.entry, tt.local, tt.exists); got != tt.want {
				t.Errorf("classifyManifestEntry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIncrementalSyncService_NoChanges(t *testing.T) {
	client := &fakeManifestClient{
		getManifestPageFunc: func(ctx context.Context, pageURL string) (scryfallclient.ManifestPage, error) {
			u, _ := url.Parse(pageURL)
			lang := u.Query().Get("lang")
			if lang == "en" {
				return scryfallclient.ManifestPage{
					TotalCards: 1,
					HasMore:    false,
					Data: []scryfallclient.ManifestEntry{
						{ID: "card-1", ImageUpdatedAt: strPtr("2026-10-10T13:01:28Z")},
					},
				}, nil
			}
			return scryfallclient.ManifestPage{HasMore: false}, nil
		},
	}

	syncService, _, jobService, settingsService, db := setupIncrementalSyncTest(t, client)
	syncService.languages = []string{"en", "de"}
	seedLocalCard(t, db, "card-1", "Unchanged Card", "2026-10-10T13:01:28Z", "")

	job, err := syncService.CreateSyncJob(context.Background())
	if err != nil {
		t.Fatalf("CreateSyncJob failed: %v", err)
	}
	if err := syncService.Sync(context.Background(), job.ID); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(client.collectionBatches) != 0 {
		t.Errorf("expected 0 collection calls, got %d", len(client.collectionBatches))
	}
	if len(client.manifestCalls) != 2 {
		t.Errorf("expected 2 manifest calls (en, de), got %d", len(client.manifestCalls))
	}

	updatedJob, _ := jobService.Get(context.Background(), job.ID)
	if updatedJob.Status != models.JobStatusCompleted {
		t.Errorf("job status = %s, want completed", updatedJob.Status)
	}

	var meta IncrementalSyncMetadata
	if err := json.Unmarshal([]byte(updatedJob.Metadata), &meta); err != nil {
		t.Fatalf("failed to unmarshal metadata: %v", err)
	}
	if meta.Mode != "incremental" || meta.Phase != "completed" || meta.UnchangedCards != 1 {
		t.Errorf("unexpected metadata: %+v", meta)
	}

	status, _ := settingsService.Get(context.Background(), "incremental_card_sync_last_status")
	if status != TaskStatusSuccess {
		t.Errorf("last_status = %q, want %q", status, TaskStatusSuccess)
	}
	lastRun, _ := settingsService.GetTime(context.Background(), "incremental_card_sync_last_run")
	if lastRun == nil {
		t.Error("expected incremental_card_sync_last_run to be set")
	}
}

func TestIncrementalSyncService_MultiPageAndBatchedHydration(t *testing.T) {
	const totalChanged = 160 // Exercises 3 collection batches: 75 + 75 + 10.
	pages := make(map[string]scryfallclient.ManifestPage)

	// Build 2 pages for "en" plus 1 page for "de" with a duplicate ID across pages.
	page1Entries := make([]scryfallclient.ManifestEntry, 0, 81)
	page1Entries = append(page1Entries, scryfallclient.ManifestEntry{
		ID:             "unchanged-1",
		ImageUpdatedAt: strPtr("2026-10-01T00:00:00Z"),
	})
	for i := 0; i < 80; i++ {
		page1Entries = append(page1Entries, scryfallclient.ManifestEntry{
			ID:             fmt.Sprintf("new-%03d", i),
			ImageUpdatedAt: strPtr("2026-11-20T00:00:00Z"),
			DataUpdatedAt:  strPtr("2026-11-21T00:00:00Z"),
		})
	}

	page2Entries := make([]scryfallclient.ManifestEntry, 0, 80)
	// Duplicate of new-000 should be deduplicated.
	page2Entries = append(page2Entries, scryfallclient.ManifestEntry{
		ID:             "new-000",
		ImageUpdatedAt: strPtr("2026-11-20T00:00:00Z"),
	})
	for i := 80; i < totalChanged-1; i++ {
		page2Entries = append(page2Entries, scryfallclient.ManifestEntry{
			ID:             fmt.Sprintf("new-%03d", i),
			ImageUpdatedAt: strPtr("2026-11-20T00:00:00Z"),
		})
	}
	page2Entries = append(page2Entries, scryfallclient.ManifestEntry{
		ID:             "updated-1",
		ImageUpdatedAt: strPtr("2026-11-24T12:00:00Z"),
	})

	pages["en:1"] = scryfallclient.ManifestPage{
		HasMore:  true,
		NextPage: "https://api.scryfall.com/cards/manifest?lang=en&page=2",
		Data:     page1Entries,
	}
	pages["en:2"] = scryfallclient.ManifestPage{
		HasMore: false,
		Data:    page2Entries,
	}
	pages["de:1"] = scryfallclient.ManifestPage{
		HasMore: false,
		Data: []scryfallclient.ManifestEntry{
			{ID: "new-de-1", ImageUpdatedAt: strPtr("2026-11-20T00:00:00Z")},
		},
	}

	client := &fakeManifestClient{
		getManifestPageFunc: func(ctx context.Context, pageURL string) (scryfallclient.ManifestPage, error) {
			u, _ := url.Parse(pageURL)
			key := u.Query().Get("lang") + ":" + u.Query().Get("page")
			p, ok := pages[key]
			if !ok {
				return scryfallclient.ManifestPage{}, fmt.Errorf("unexpected page %s", key)
			}
			return p, nil
		},
		getCollectionByIDsFunc: func(ctx context.Context, ids []string) ([]scryfall.Card, []string, error) {
			cards := make([]scryfall.Card, 0, len(ids))
			for _, id := range ids {
				if id == "updated-1" {
					cards = append(cards, scryfall.Card{
						ID:       id,
						OracleID: "oracle-updated",
						Name:     "Updated Token",
						Set:      "tblb",
						Layout:   scryfall.LayoutToken,
						TypeLine: "Token Creature — Rabbit",
					})
					continue
				}
				cards = append(cards, scryfall.Card{
					ID:       id,
					OracleID: "oracle-" + id,
					Name:     "Card " + id,
					Set:      "tst",
				})
			}
			return cards, nil, nil
		},
	}

	syncService, _, jobService, _, db := setupIncrementalSyncTest(t, client)
	syncService.languages = []string{"en", "de"}
	seedLocalCard(t, db, "unchanged-1", "Original Unchanged", "2026-10-01T00:00:00Z", "")
	seedLocalCard(t, db, "updated-1", "Old Name", "2026-10-01T00:00:00Z", "")

	// Seed a deck item pointing to updated-1 in main zone to verify ReclassifyDeckExtras runs.
	deck := models.Deck{Name: "Test Deck"}
	if err := db.Create(&deck).Error; err != nil {
		t.Fatalf("failed to create deck: %v", err)
	}
	deckItem := models.DeckItem{
		DeckID:          deck.ID,
		OracleID:        "oracle-updated",
		ScryfallID:      "updated-1",
		Zone:            models.ZoneMain,
		DesiredQuantity: 1,
	}
	if err := db.Create(&deckItem).Error; err != nil {
		t.Fatalf("failed to create deck item: %v", err)
	}

	job, _ := syncService.CreateSyncJob(context.Background())
	if err := syncService.Sync(context.Background(), job.ID); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// 159 new-XXX + 1 updated-1 + 1 new-de-1 = 161 cards => 75 + 75 + 11.
	if len(client.collectionBatches) != 3 {
		t.Fatalf("expected 3 collection batches, got %d", len(client.collectionBatches))
	}
	if len(client.collectionBatches[0]) != 75 || len(client.collectionBatches[1]) != 75 || len(client.collectionBatches[2]) != 11 {
		t.Errorf("unexpected batch sizes: %d, %d, %d",
			len(client.collectionBatches[0]), len(client.collectionBatches[1]), len(client.collectionBatches[2]))
	}

	var totalCount int64
	db.Model(&models.Card{}).Count(&totalCount)
	if totalCount != 162 { // 161 hydrated + 1 unchanged
		t.Errorf("expected 162 cards in DB, got %d", totalCount)
	}

	// Verify unchanged card was not touched.
	var unchanged models.Card
	db.First(&unchanged, "scryfall_id = ?", "unchanged-1")
	if !strings.Contains(unchanged.RawJSON, "Original Unchanged") {
		t.Errorf("unchanged card RawJSON was unexpectedly overwritten: %s", unchanged.RawJSON)
	}

	// Verify manifest timestamps persisted onto hydrated card when card JSON lacked them.
	var new0 models.Card
	db.First(&new0, "scryfall_id = ?", "new-000")
	if new0.ImageUpdatedAt != "2026-11-20T00:00:00Z" {
		t.Errorf("new-000 ImageUpdatedAt = %q, want 2026-11-20T00:00:00Z", new0.ImageUpdatedAt)
	}
	if new0.DataUpdatedAt != "2026-11-21T00:00:00Z" {
		t.Errorf("new-000 DataUpdatedAt = %q, want 2026-11-21T00:00:00Z", new0.DataUpdatedAt)
	}

	// Verify deck extra reclassification ran.
	if zone := deckItemZone(t, db, deckItem.ID); zone != models.ZoneOther {
		t.Errorf("deck item zone = %q, want %q", zone, models.ZoneOther)
	}

	updatedJob, _ := jobService.Get(context.Background(), job.ID)
	var meta IncrementalSyncMetadata
	_ = json.Unmarshal([]byte(updatedJob.Metadata), &meta)
	if meta.ManifestPagesFetched != 3 || meta.NewCards != 160 || meta.UpdatedCards != 1 || meta.UnchangedCards != 1 || meta.ProcessedCards != 161 {
		t.Errorf("unexpected metadata: %+v", meta)
	}
}

func TestIncrementalSyncService_AllNotFoundBatchSucceeds(t *testing.T) {
	client := &fakeManifestClient{
		getManifestPageFunc: func(ctx context.Context, pageURL string) (scryfallclient.ManifestPage, error) {
			return scryfallclient.ManifestPage{
				HasMore: false,
				Data: []scryfallclient.ManifestEntry{
					{ID: "ghost-1"},
					{ID: "ghost-2"},
				},
			}, nil
		},
		getCollectionByIDsFunc: func(ctx context.Context, ids []string) ([]scryfall.Card, []string, error) {
			return nil, ids, nil
		},
	}

	syncService, _, jobService, _, db := setupIncrementalSyncTest(t, client)
	syncService.languages = []string{"en"}
	seedLocalCard(t, db, "existing-1", "Existing", "2026-10-01T00:00:00Z", "")

	job, _ := syncService.CreateSyncJob(context.Background())
	if err := syncService.Sync(context.Background(), job.ID); err != nil {
		t.Fatalf("Sync should succeed when collection returns not_found: %v", err)
	}

	updatedJob, _ := jobService.Get(context.Background(), job.ID)
	if updatedJob.Status != models.JobStatusCompleted {
		t.Errorf("job status = %s, want completed", updatedJob.Status)
	}
	var meta IncrementalSyncMetadata
	_ = json.Unmarshal([]byte(updatedJob.Metadata), &meta)
	if meta.NotFoundCards != 2 || meta.ProcessedCards != 0 {
		t.Errorf("unexpected metadata: %+v", meta)
	}
}

func TestIncrementalSyncService_ColdStartEmptyDBFallback(t *testing.T) {
	client := &fakeManifestClient{}
	syncService, bulkService, jobService, _, db := setupIncrementalSyncTest(t, client)
	serveBulkData(t, bulkService, []scryfall.Card{
		{ID: "bulk-1", OracleID: "o1", Name: "Bulk One", Set: "tst"},
		{ID: "bulk-2", OracleID: "o2", Name: "Bulk Two", Set: "tst"},
	})

	job, _ := syncService.CreateSyncJob(context.Background())
	if err := syncService.Sync(context.Background(), job.ID); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(client.manifestCalls) != 0 {
		t.Errorf("expected 0 manifest calls on empty DB fallback, got %d", len(client.manifestCalls))
	}

	var count int64
	db.Model(&models.Card{}).Count(&count)
	if count != 2 {
		t.Errorf("expected 2 cards from bulk fallback, got %d", count)
	}

	updatedJob, _ := jobService.Get(context.Background(), job.ID)
	var meta IncrementalSyncMetadata
	_ = json.Unmarshal([]byte(updatedJob.Metadata), &meta)
	if meta.Mode != "full_fallback" || meta.FallbackReason != "empty_database" || meta.ProcessedCards != 2 {
		t.Errorf("unexpected metadata: %+v", meta)
	}
}

func TestIncrementalSyncService_ThresholdExceededEarlyFallback(t *testing.T) {
	client := &fakeManifestClient{
		getManifestPageFunc: func(ctx context.Context, pageURL string) (scryfallclient.ManifestPage, error) {
			u, _ := url.Parse(pageURL)
			if u.Query().Get("lang") != "en" || u.Query().Get("page") != "1" {
				t.Errorf("expected early fallback after en page 1, should not have fetched %s", pageURL)
			}
			return scryfallclient.ManifestPage{
				HasMore:  true,
				NextPage: "https://api.scryfall.com/cards/manifest?lang=en&page=2",
				Data: []scryfallclient.ManifestEntry{
					{ID: "n1"}, {ID: "n2"}, {ID: "n3"},
				},
			}, nil
		},
	}

	syncService, bulkService, jobService, settingsService, db := setupIncrementalSyncTest(t, client)
	syncService.languages = []string{"en", "de"}
	seedLocalCard(t, db, "existing", "Existing", "", "")
	if err := settingsService.Set(context.Background(), "incremental_card_sync_fallback_threshold", "2"); err != nil {
		t.Fatalf("failed to set threshold: %v", err)
	}
	serveBulkData(t, bulkService, []scryfall.Card{
		{ID: "existing", OracleID: "o0", Name: "Existing", Set: "tst"},
		{ID: "n1", OracleID: "o1", Name: "N1", Set: "tst"},
		{ID: "n2", OracleID: "o2", Name: "N2", Set: "tst"},
		{ID: "n3", OracleID: "o3", Name: "N3", Set: "tst"},
	})

	job, _ := syncService.CreateSyncJob(context.Background())
	if err := syncService.Sync(context.Background(), job.ID); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(client.manifestCalls) != 1 {
		t.Errorf("expected manifest scan to abort after 1 page, got %d calls", len(client.manifestCalls))
	}
	if len(client.collectionBatches) != 0 {
		t.Errorf("expected 0 collection calls on fallback, got %d", len(client.collectionBatches))
	}

	var count int64
	db.Model(&models.Card{}).Count(&count)
	if count != 4 {
		t.Errorf("expected 4 cards in DB, got %d", count)
	}

	updatedJob, _ := jobService.Get(context.Background(), job.ID)
	var meta IncrementalSyncMetadata
	_ = json.Unmarshal([]byte(updatedJob.Metadata), &meta)
	if meta.Mode != "full_fallback" || !strings.Contains(meta.FallbackReason, "diff_exceeded_threshold") || meta.ProcessedCards != 4 {
		t.Errorf("unexpected metadata: %+v", meta)
	}
}

func TestIncrementalSyncService_ZeroThresholdDisablesFallback(t *testing.T) {
	client := &fakeManifestClient{
		getManifestPageFunc: func(ctx context.Context, pageURL string) (scryfallclient.ManifestPage, error) {
			return scryfallclient.ManifestPage{
				HasMore: false,
				Data: []scryfallclient.ManifestEntry{
					{ID: "n1"}, {ID: "n2"}, {ID: "n3"},
				},
			}, nil
		},
		getCollectionByIDsFunc: func(ctx context.Context, ids []string) ([]scryfall.Card, []string, error) {
			out := make([]scryfall.Card, 0, len(ids))
			for _, id := range ids {
				out = append(out, scryfall.Card{ID: id, OracleID: "o-" + id, Name: id, Set: "tst"})
			}
			return out, nil, nil
		},
	}

	syncService, _, jobService, settingsService, db := setupIncrementalSyncTest(t, client)
	syncService.languages = []string{"en"}
	seedLocalCard(t, db, "existing", "Existing", "", "")
	if err := settingsService.Set(context.Background(), "incremental_card_sync_fallback_threshold", "0"); err != nil {
		t.Fatalf("failed to set threshold=0: %v", err)
	}

	job, _ := syncService.CreateSyncJob(context.Background())
	if err := syncService.Sync(context.Background(), job.ID); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(client.collectionBatches) != 1 {
		t.Errorf("expected 1 collection call when fallback disabled, got %d", len(client.collectionBatches))
	}
	updatedJob, _ := jobService.Get(context.Background(), job.ID)
	var meta IncrementalSyncMetadata
	_ = json.Unmarshal([]byte(updatedJob.Metadata), &meta)
	if meta.Mode != "incremental" || meta.ProcessedCards != 3 {
		t.Errorf("unexpected metadata: %+v", meta)
	}
}

func TestIncrementalSyncService_ErrorPaths(t *testing.T) {
	t.Run("manifest error marks job failed", func(t *testing.T) {
		client := &fakeManifestClient{
			getManifestPageFunc: func(ctx context.Context, pageURL string) (scryfallclient.ManifestPage, error) {
				return scryfallclient.ManifestPage{}, errors.New("manifest unreachable")
			},
		}
		syncService, _, jobService, settingsService, db := setupIncrementalSyncTest(t, client)
		syncService.languages = []string{"en"}
		seedLocalCard(t, db, "existing", "Existing", "", "")

		job, _ := syncService.CreateSyncJob(context.Background())
		if err := syncService.Sync(context.Background(), job.ID); err == nil {
			t.Fatal("expected error")
		}
		gotJob, _ := jobService.Get(context.Background(), job.ID)
		if gotJob.Status != models.JobStatusFailed || !strings.Contains(gotJob.Error, "manifest unreachable") {
			t.Errorf("unexpected job state: status=%s err=%q", gotJob.Status, gotJob.Error)
		}
		if status, _ := settingsService.Get(context.Background(), "incremental_card_sync_last_status"); status != TaskStatusFailed {
			t.Errorf("last_status = %q, want failed", status)
		}
	})

	t.Run("collection error marks job failed", func(t *testing.T) {
		client := &fakeManifestClient{
			getManifestPageFunc: func(ctx context.Context, pageURL string) (scryfallclient.ManifestPage, error) {
				return scryfallclient.ManifestPage{Data: []scryfallclient.ManifestEntry{{ID: "new-1"}}}, nil
			},
			getCollectionByIDsFunc: func(ctx context.Context, ids []string) ([]scryfall.Card, []string, error) {
				return nil, nil, errors.New("collection 500")
			},
		}
		syncService, _, jobService, _, db := setupIncrementalSyncTest(t, client)
		syncService.languages = []string{"en"}
		seedLocalCard(t, db, "existing", "Existing", "", "")

		job, _ := syncService.CreateSyncJob(context.Background())
		if err := syncService.Sync(context.Background(), job.ID); err == nil {
			t.Fatal("expected error")
		}
		gotJob, _ := jobService.Get(context.Background(), job.ID)
		if gotJob.Status != models.JobStatusFailed || !strings.Contains(gotJob.Error, "collection 500") {
			t.Errorf("unexpected job state: status=%s err=%q", gotJob.Status, gotJob.Error)
		}
	})

	t.Run("pagination loop detected", func(t *testing.T) {
		client := &fakeManifestClient{
			getManifestPageFunc: func(ctx context.Context, pageURL string) (scryfallclient.ManifestPage, error) {
				return scryfallclient.ManifestPage{
					HasMore:  true,
					NextPage: pageURL,
				}, nil
			},
		}
		syncService, _, _, _, db := setupIncrementalSyncTest(t, client)
		syncService.languages = []string{"en"}
		seedLocalCard(t, db, "existing", "Existing", "", "")

		job, _ := syncService.CreateSyncJob(context.Background())
		err := syncService.Sync(context.Background(), job.ID)
		if err == nil || !strings.Contains(err.Error(), "loop") {
			t.Errorf("expected loop error, got %v", err)
		}
	})

	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		client := &fakeManifestClient{
			getManifestPageFunc: func(ctx context.Context, pageURL string) (scryfallclient.ManifestPage, error) {
				cancel()
				return scryfallclient.ManifestPage{}, ctx.Err()
			},
		}
		syncService, _, jobService, _, db := setupIncrementalSyncTest(t, client)
		syncService.languages = []string{"en"}
		seedLocalCard(t, db, "existing", "Existing", "", "")

		job, _ := syncService.CreateSyncJob(context.Background())
		if err := syncService.Sync(ctx, job.ID); !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
		gotJob, _ := jobService.Get(context.Background(), job.ID)
		if gotJob.Status != models.JobStatusFailed {
			t.Errorf("job status = %s, want failed", gotJob.Status)
		}
	})
}
