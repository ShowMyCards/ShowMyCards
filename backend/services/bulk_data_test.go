package services

import (
	"backend/models"
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	scryfall "github.com/BlueMonday/go-scryfall"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// bulkDataListResponse builds a /bulk-data list payload pointing all_cards at
// the given gzipped-JSONL URL, matching Scryfall's response shape.
func bulkDataListResponse(jsonlURL string) map[string]interface{} {
	return map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{
				"type":               "all_cards",
				"jsonl_download_uri": jsonlURL,
			},
		},
	}
}

// writeJSONLGzip writes cards as gzipped JSONL (one card object per line,
// gzip-compressed), matching Scryfall's bulk data file format.
func writeJSONLGzip(t *testing.T, w http.ResponseWriter, cards []scryfall.Card) {
	t.Helper()
	w.Header().Set("Content-Type", "application/gzip")
	gz := gzip.NewWriter(w)
	defer gz.Close()
	// json.Encoder.Encode writes a trailing newline after each value, so
	// encoding cards in sequence yields JSONL. Encode &cards[i] (an
	// addressable pointer) so scryfall.Date's pointer-receiver MarshalJSON is
	// honored and dates serialize date-only, as real Scryfall data does.
	enc := json.NewEncoder(gz)
	for i := range cards {
		if err := enc.Encode(&cards[i]); err != nil {
			t.Fatalf("failed to encode card as JSONL: %v", err)
		}
	}
}

func setupBulkDataServiceTest(t *testing.T) (*BulkDataService, *JobService, *SettingsService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}

	// Deck tables and sets are migrated too: a successful import reclassifies
	// deck extras, which reads deck items and joins sets.
	if err := db.AutoMigrate(&models.Job{}, &models.Setting{}, &models.Card{},
		&models.Deck{}, &models.DeckItem{}, &models.Set{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	jobService := NewJobService(db)
	settingsService := NewSettingsService(db)
	bulkDataService := NewBulkDataService(db, jobService, settingsService)

	return bulkDataService, jobService, settingsService, db
}

// HasBulkData tests

func TestBulkDataService_HasBulkData_WithData(t *testing.T) {
	service, _, _, db := setupBulkDataServiceTest(t)

	// Insert test card
	card := models.Card{
		ScryfallID: "test-id",
		OracleID:   "test-oracle",
		RawJSON:    `{"id": "test-id", "name": "Test"}`,
	}
	db.Create(&card)

	hasData, err := service.HasBulkData()
	if err != nil {
		t.Fatalf("HasBulkData failed: %v", err)
	}

	if !hasData {
		t.Error("expected HasBulkData to return true when cards exist")
	}
}

func TestBulkDataService_HasBulkData_WithoutData(t *testing.T) {
	service, _, _, _ := setupBulkDataServiceTest(t)

	hasData, err := service.HasBulkData()
	if err != nil {
		t.Fatalf("HasBulkData failed: %v", err)
	}

	if hasData {
		t.Error("expected HasBulkData to return false when no cards exist")
	}
}

// TriggerInitialImport tests

func TestBulkDataService_TriggerInitialImport_WhenNeeded(t *testing.T) {
	service, jobService, _, _ := setupBulkDataServiceTest(t)

	err := service.TriggerInitialImport(context.Background())
	if err != nil {
		t.Fatalf("TriggerInitialImport failed: %v", err)
	}

	// Give goroutine a moment to create the job
	time.Sleep(50 * time.Millisecond)

	// Verify job was created
	jobs, total, err := jobService.List(context.Background(), 1, 10, nil, nil)
	if err != nil {
		t.Fatalf("failed to list jobs: %v", err)
	}

	if total != 1 {
		t.Errorf("expected 1 job created, got %d", total)
	}

	if len(jobs) > 0 && jobs[0].Type != models.JobTypeBulkDataImport {
		t.Errorf("expected job type %s, got %s", models.JobTypeBulkDataImport, jobs[0].Type)
	}
}

func TestBulkDataService_TriggerInitialImport_WhenDataExists(t *testing.T) {
	service, jobService, _, db := setupBulkDataServiceTest(t)

	// Insert test card
	card := models.Card{
		ScryfallID: "existing-card",
		OracleID:   "existing-oracle",
		RawJSON:    `{"id": "existing-card"}`,
	}
	db.Create(&card)

	err := service.TriggerInitialImport(context.Background())
	if err != nil {
		t.Fatalf("TriggerInitialImport failed: %v", err)
	}

	// Verify no job was created
	_, total, err := jobService.List(context.Background(), 1, 10, nil, nil)
	if err != nil {
		t.Fatalf("failed to list jobs: %v", err)
	}

	if total != 0 {
		t.Errorf("expected 0 jobs when data exists, got %d", total)
	}
}

// CreateImportJob tests

func TestBulkDataService_CreateImportJob_Success(t *testing.T) {
	service, jobService, _, _ := setupBulkDataServiceTest(t)

	job, err := service.CreateImportJob(context.Background())
	if err != nil {
		t.Fatalf("CreateImportJob failed: %v", err)
	}

	if job.Type != models.JobTypeBulkDataImport {
		t.Errorf("expected type %s, got %s", models.JobTypeBulkDataImport, job.Type)
	}

	if job.Status != models.JobStatusPending {
		t.Errorf("expected status %s, got %s", models.JobStatusPending, job.Status)
	}

	// Verify in database
	retrieved, err := jobService.Get(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	if retrieved.Type != models.JobTypeBulkDataImport {
		t.Errorf("expected type %s, got %s", models.JobTypeBulkDataImport, retrieved.Type)
	}
}

// DownloadAndImport integration tests

func TestBulkDataService_DownloadAndImport_SuccessfulImport(t *testing.T) {
	service, jobService, _, db := setupBulkDataServiceTest(t)

	// Create mock card data (small set for testing)
	cards := []scryfall.Card{
		{
			ID:       "card-1",
			OracleID: "oracle-1",
			Name:     "Card One",
			Set:      "tst",
		},
		{
			ID:       "card-2",
			OracleID: "oracle-2",
			Name:     "Card Two",
			Set:      "tst",
		},
	}

	// Mock download server for both bulk data list and actual data
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bulk-data" {
			// Return bulk data list
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(bulkDataListResponse(server.URL + "/cards.jsonl.gz"))
		} else {
			// Return gzipped JSONL card data
			writeJSONLGzip(t, w, cards)
		}
	}))
	defer server.Close()

	// Set bulk data URL
	service.settingsService.Set(context.Background(), "bulk_data_url", server.URL+"/bulk-data")

	// Create job
	job, _ := jobService.Create(context.Background(), models.JobTypeBulkDataImport, "{}")

	// Run import
	err := service.DownloadAndImport(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("DownloadAndImport failed: %v", err)
	}

	// Verify cards were imported
	var count int64
	db.Model(&models.Card{}).Count(&count)
	if count != 2 {
		t.Errorf("expected 2 cards imported, got %d", count)
	}

	// Verify job was completed and recorded zero deck items checked/moved
	updatedJob, _ := jobService.Get(context.Background(), job.ID)
	if updatedJob.Status != models.JobStatusCompleted {
		t.Errorf("expected job status %s, got %s", models.JobStatusCompleted, updatedJob.Status)
	}
	md := decodeJobMetadata(t, updatedJob)
	if md.DeckItemsChecked == nil || *md.DeckItemsChecked != 0 {
		t.Errorf("deck_items_checked = %v, want 0", md.DeckItemsChecked)
	}
	if md.DeckItemsMoved == nil || *md.DeckItemsMoved != 0 {
		t.Errorf("deck_items_moved = %v, want 0", md.DeckItemsMoved)
	}
	if md.DeckResortError != "" {
		t.Errorf("deck_resort_error = %q, want empty", md.DeckResortError)
	}
}

func TestBulkDataService_DownloadAndImport_LowFailureRate(t *testing.T) {
	service, jobService, _, db := setupBulkDataServiceTest(t)

	// Create all valid cards to test successful completion
	// In real-world usage, failures would come from JSON marshaling errors
	// which are extremely rare with the go-scryfall library
	cards := []scryfall.Card{
		{ID: "valid-1", OracleID: "oracle-1", Name: "Valid 1", Set: "tst"},
		{ID: "valid-2", OracleID: "oracle-2", Name: "Valid 2", Set: "tst"},
		{ID: "valid-3", OracleID: "oracle-3", Name: "Valid 3", Set: "tst"},
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bulk-data" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(bulkDataListResponse(server.URL + "/cards.jsonl.gz"))
		} else {
			writeJSONLGzip(t, w, cards)
		}
	}))
	defer server.Close()

	service.settingsService.Set(context.Background(), "bulk_data_url", server.URL+"/bulk-data")
	job, _ := jobService.Create(context.Background(), models.JobTypeBulkDataImport, "{}")

	err := service.DownloadAndImport(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("DownloadAndImport failed: %v", err)
	}

	// Verify all cards were saved
	var count int64
	db.Model(&models.Card{}).Count(&count)
	if count != 3 {
		t.Errorf("expected 3 cards in database, got %d", count)
	}

	// Job should complete successfully
	updatedJob, _ := jobService.Get(context.Background(), job.ID)
	if updatedJob.Status != models.JobStatusCompleted {
		t.Errorf("expected job status %s, got %s", models.JobStatusCompleted, updatedJob.Status)
	}
}

func TestBulkDataService_DownloadAndImport_HighFailureRate(t *testing.T) {
	service, jobService, _, _ := setupBulkDataServiceTest(t)

	// Create mostly invalid cards (>5% failure rate)
	cards := []scryfall.Card{
		{ID: "valid-1", OracleID: "oracle-1", Name: "Valid", Set: "tst"},
		{OracleID: "oracle-2", Name: "invalid-1"}, // Missing ID
		{OracleID: "oracle-3", Name: "invalid-2"},
		{OracleID: "oracle-4", Name: "invalid-3"},
		{OracleID: "oracle-5", Name: "invalid-4"},
		{OracleID: "oracle-6", Name: "invalid-5"},
		{OracleID: "oracle-7", Name: "invalid-6"},
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bulk-data" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(bulkDataListResponse(server.URL + "/cards.jsonl.gz"))
		} else {
			writeJSONLGzip(t, w, cards)
		}
	}))
	defer server.Close()

	service.settingsService.Set(context.Background(), "bulk_data_url", server.URL+"/bulk-data")
	job, _ := jobService.Create(context.Background(), models.JobTypeBulkDataImport, "{}")

	err := service.DownloadAndImport(context.Background(), job.ID)
	if err == nil {
		t.Error("expected error for high failure rate")
	}

	// Verify job was marked as failed
	updatedJob, _ := jobService.Get(context.Background(), job.ID)
	if updatedJob.Status != models.JobStatusFailed {
		t.Errorf("expected job status %s, got %s", models.JobStatusFailed, updatedJob.Status)
	}

	// Verify error message mentions failure rate
	if updatedJob.Error == "" {
		t.Error("expected error message to be set")
	}
}

func TestBulkDataService_DownloadAndImport_ContextCancellation(t *testing.T) {
	service, jobService, _, _ := setupBulkDataServiceTest(t)

	// Create server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		json.NewEncoder(w).Encode([]scryfall.Card{})
	}))
	defer server.Close()

	service.settingsService.Set(context.Background(), "bulk_data_url", server.URL+"/bulk-data")
	job, _ := jobService.Create(context.Background(), models.JobTypeBulkDataImport, "{}")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := service.DownloadAndImport(ctx, job.ID)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestBulkDataService_DownloadAndImport_InvalidURL(t *testing.T) {
	service, jobService, _, _ := setupBulkDataServiceTest(t)

	service.settingsService.Set(context.Background(), "bulk_data_url", "http://invalid-url-that-does-not-exist.example.com")
	job, _ := jobService.Create(context.Background(), models.JobTypeBulkDataImport, "{}")

	err := service.DownloadAndImport(context.Background(), job.ID)
	if err == nil {
		t.Error("expected error for invalid URL")
	}

	// Verify job was marked as failed
	updatedJob, _ := jobService.Get(context.Background(), job.ID)
	if updatedJob.Status != models.JobStatusFailed {
		t.Errorf("expected job status %s, got %s", models.JobStatusFailed, updatedJob.Status)
	}
}

func TestBulkDataService_DownloadAndImport_HTTPError(t *testing.T) {
	service, jobService, _, _ := setupBulkDataServiceTest(t)

	// Server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	service.settingsService.Set(context.Background(), "bulk_data_url", server.URL)
	job, _ := jobService.Create(context.Background(), models.JobTypeBulkDataImport, "{}")

	err := service.DownloadAndImport(context.Background(), job.ID)
	if err == nil {
		t.Error("expected error for HTTP failure")
	}

	updatedJob, _ := jobService.Get(context.Background(), job.ID)
	if updatedJob.Status != models.JobStatusFailed {
		t.Errorf("expected job status %s, got %s", models.JobStatusFailed, updatedJob.Status)
	}
}

func TestBulkDataService_DownloadAndImport_InvalidJSON(t *testing.T) {
	service, jobService, _, _ := setupBulkDataServiceTest(t)

	// Server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	service.settingsService.Set(context.Background(), "bulk_data_url", server.URL)
	job, _ := jobService.Create(context.Background(), models.JobTypeBulkDataImport, "{}")

	err := service.DownloadAndImport(context.Background(), job.ID)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	updatedJob, _ := jobService.Get(context.Background(), job.ID)
	if updatedJob.Status != models.JobStatusFailed {
		t.Errorf("expected job status %s, got %s", models.JobStatusFailed, updatedJob.Status)
	}
}

func TestBulkDataService_DownloadAndImport_InvalidGzip(t *testing.T) {
	service, jobService, _, _ := setupBulkDataServiceTest(t)

	// Bulk data list is valid, but the card file is not gzip-compressed.
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bulk-data" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(bulkDataListResponse(server.URL + "/cards.jsonl.gz"))
		} else {
			w.Write([]byte("not gzip data"))
		}
	}))
	defer server.Close()

	service.settingsService.Set(context.Background(), "bulk_data_url", server.URL+"/bulk-data")
	job, _ := jobService.Create(context.Background(), models.JobTypeBulkDataImport, "{}")

	err := service.DownloadAndImport(context.Background(), job.ID)
	if err == nil {
		t.Error("expected error for non-gzip card data")
	}

	updatedJob, _ := jobService.Get(context.Background(), job.ID)
	if updatedJob.Status != models.JobStatusFailed {
		t.Errorf("expected job status %s, got %s", models.JobStatusFailed, updatedJob.Status)
	}
}

func TestBulkDataService_DownloadAndImport_NoDefaultCardsInList(t *testing.T) {
	service, jobService, _, _ := setupBulkDataServiceTest(t)

	// Server with no all_cards entry
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"data": []interface{}{
				map[string]interface{}{
					"type":         "oracle_cards",
					"download_uri": "https://example.com/oracle.json",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	service.settingsService.Set(context.Background(), "bulk_data_url", server.URL)
	job, _ := jobService.Create(context.Background(), models.JobTypeBulkDataImport, "{}")

	err := service.DownloadAndImport(context.Background(), job.ID)
	if err == nil {
		t.Error("expected error when all_cards not found")
	}

	updatedJob, _ := jobService.Get(context.Background(), job.ID)
	if updatedJob.Status != models.JobStatusFailed {
		t.Errorf("expected job status %s, got %s", models.JobStatusFailed, updatedJob.Status)
	}
}

func TestBulkDataService_DownloadAndImport_UpdatesSettings(t *testing.T) {
	service, jobService, settingsService, _ := setupBulkDataServiceTest(t)

	cards := []scryfall.Card{
		{ID: "test", OracleID: "oracle", Name: "Test", Set: "tst"},
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bulk-data" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(bulkDataListResponse(server.URL + "/cards.jsonl.gz"))
		} else {
			writeJSONLGzip(t, w, cards)
		}
	}))
	defer server.Close()

	service.settingsService.Set(context.Background(), "bulk_data_url", server.URL+"/bulk-data")
	job, _ := jobService.Create(context.Background(), models.JobTypeBulkDataImport, "{}")

	err := service.DownloadAndImport(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("DownloadAndImport failed: %v", err)
	}

	// Verify settings were updated
	status, _ := settingsService.Get(context.Background(), "bulk_data_update_last_status")
	if status != "success" {
		t.Errorf("expected status 'success', got '%s'", status)
	}

	lastUpdate, _ := settingsService.GetTime(context.Background(), "bulk_data_update_last_run")
	if lastUpdate == nil {
		t.Error("expected last_update timestamp to be set")
	}
}

func TestBulkDataService_DownloadAndImport_PreservesExistingData(t *testing.T) {
	service, jobService, _, db := setupBulkDataServiceTest(t)

	// Insert existing card that won't be in the new import
	existingCard := models.Card{
		ScryfallID: "old-card",
		OracleID:   "old-oracle",
		RawJSON:    `{"id": "old-card", "name": "Old Card"}`,
	}
	db.Create(&existingCard)

	// New card data (does not include old-card)
	cards := []scryfall.Card{
		{ID: "new-card", OracleID: "new-oracle", Name: "New", Set: "tst"},
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bulk-data" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(bulkDataListResponse(server.URL + "/cards.jsonl.gz"))
		} else {
			writeJSONLGzip(t, w, cards)
		}
	}))
	defer server.Close()

	service.settingsService.Set(context.Background(), "bulk_data_url", server.URL+"/bulk-data")
	job, _ := jobService.Create(context.Background(), models.JobTypeBulkDataImport, "{}")

	err := service.DownloadAndImport(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("DownloadAndImport failed: %v", err)
	}

	// Verify old card is preserved (UPSERT strategy does not delete existing cards)
	var count int64
	db.Model(&models.Card{}).Where("scryfall_id = ?", "old-card").Count(&count)
	if count != 1 {
		t.Error("expected old card to be preserved")
	}

	// Verify new card was imported
	db.Model(&models.Card{}).Where("scryfall_id = ?", "new-card").Count(&count)
	if count != 1 {
		t.Error("expected new card to be imported")
	}

	// Verify total card count
	db.Model(&models.Card{}).Count(&count)
	if count != 2 {
		t.Errorf("expected 2 total cards, got %d", count)
	}
}

// Deck extras reclassification runs after a successful import only.

// serveBulkData starts a mock Scryfall serving cards and points the service at it.
func serveBulkData(t *testing.T, service *BulkDataService, cards []scryfall.Card) {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bulk-data" {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(bulkDataListResponse(server.URL + "/cards.jsonl.gz")); err != nil {
				t.Errorf("failed to write bulk data list: %v", err)
			}
		} else {
			writeJSONLGzip(t, w, cards)
		}
	}))
	t.Cleanup(server.Close)
	if err := service.settingsService.Set(context.Background(), "bulk_data_url", server.URL+"/bulk-data"); err != nil {
		t.Fatalf("failed to set bulk data URL: %v", err)
	}
}

// seedTokenInMain creates a deck holding a token printing in the main zone, as
// decks did before tokens had their own zone.
func seedTokenInMain(t *testing.T, db *gorm.DB) models.DeckItem {
	t.Helper()
	deck := models.Deck{Name: "Rabbits"}
	if err := db.Create(&deck).Error; err != nil {
		t.Fatalf("failed to create deck: %v", err)
	}
	item := models.DeckItem{DeckID: deck.ID, OracleID: "oracle-rabbit", ScryfallID: "rabbit-1", Zone: models.ZoneMain, DesiredQuantity: 2}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("failed to create deck item: %v", err)
	}
	return item
}

var rabbitToken = scryfall.Card{
	ID: "rabbit-1", OracleID: "oracle-rabbit", Name: "Rabbit", Set: "tblb",
	Layout: scryfall.LayoutToken, TypeLine: "Token Creature — Rabbit",
}

var normalCard = scryfall.Card{
	ID: "bolt-1", OracleID: "oracle-bolt", Name: "Lightning Bolt", Set: "lea",
	Layout: scryfall.LayoutNormal, TypeLine: "Instant",
}

func decodeJobMetadata(t *testing.T, job *models.Job) JobMetadata {
	t.Helper()
	var md JobMetadata
	if err := json.Unmarshal([]byte(job.Metadata), &md); err != nil {
		t.Fatalf("failed to decode job metadata %q: %v", job.Metadata, err)
	}
	return md
}

func deckItemZone(t *testing.T, db *gorm.DB, id uint) models.DeckZone {
	t.Helper()
	var item models.DeckItem
	if err := db.First(&item, id).Error; err != nil {
		t.Fatalf("failed to reload deck item: %v", err)
	}
	return item.Zone
}

func TestBulkDataService_DownloadAndImport_ReclassifiesDeckExtras(t *testing.T) {
	service, jobService, _, db := setupBulkDataServiceTest(t)
	item := seedTokenInMain(t, db)
	normalItem := models.DeckItem{
		DeckID:          item.DeckID,
		OracleID:        normalCard.OracleID,
		ScryfallID:      normalCard.ID,
		Zone:            models.ZoneMain,
		DesiredQuantity: 4,
	}
	if err := db.Create(&normalItem).Error; err != nil {
		t.Fatalf("failed to create normal deck item: %v", err)
	}
	serveBulkData(t, service, []scryfall.Card{rabbitToken, normalCard})

	job, _ := jobService.Create(context.Background(), models.JobTypeBulkDataImport, "{}")
	if err := service.DownloadAndImport(context.Background(), job.ID); err != nil {
		t.Fatalf("DownloadAndImport failed: %v", err)
	}

	if zone := deckItemZone(t, db, item.ID); zone != models.ZoneOther {
		t.Errorf("token zone after import = %q, want %q", zone, models.ZoneOther)
	}
	if zone := deckItemZone(t, db, normalItem.ID); zone != models.ZoneMain {
		t.Errorf("normal card zone after import = %q, want %q", zone, models.ZoneMain)
	}

	updatedJob, err := jobService.Get(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("failed to reload job: %v", err)
	}
	if updatedJob.Status != models.JobStatusCompleted {
		t.Errorf("job status = %q, want %q", updatedJob.Status, models.JobStatusCompleted)
	}
	md := decodeJobMetadata(t, updatedJob)
	if md.Phase != "completed" {
		t.Errorf("phase = %q, want completed", md.Phase)
	}
	if md.DeckItemsChecked == nil || *md.DeckItemsChecked != 2 {
		t.Errorf("deck_items_checked = %v, want 2", md.DeckItemsChecked)
	}
	if md.DeckItemsMoved == nil || *md.DeckItemsMoved != 1 {
		t.Errorf("deck_items_moved = %v, want 1", md.DeckItemsMoved)
	}
	if md.DeckResortError != "" {
		t.Errorf("deck_resort_error = %q, want empty", md.DeckResortError)
	}
}

func TestBulkDataService_DownloadAndImport_RecordsReclassifyError(t *testing.T) {
	service, jobService, settingsService, db := setupBulkDataServiceTest(t)
	serveBulkData(t, service, []scryfall.Card{rabbitToken})

	// Drop deck_items so card import succeeds but Step 4 reclassification fails.
	if err := db.Migrator().DropTable(&models.DeckItem{}); err != nil {
		t.Fatalf("failed to drop deck_items: %v", err)
	}

	job, _ := jobService.Create(context.Background(), models.JobTypeBulkDataImport, "{}")
	if err := service.DownloadAndImport(context.Background(), job.ID); err != nil {
		t.Fatalf("DownloadAndImport returned error on reclassify failure: %v", err)
	}

	var cardCount int64
	db.Model(&models.Card{}).Count(&cardCount)
	if cardCount != 1 {
		t.Errorf("imported card count = %d, want 1", cardCount)
	}

	updatedJob, err := jobService.Get(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("failed to reload job: %v", err)
	}
	if updatedJob.Status != models.JobStatusCompleted {
		t.Errorf("job status = %q, want %q", updatedJob.Status, models.JobStatusCompleted)
	}
	if updatedJob.Error != "" {
		t.Errorf("job error = %q, want empty", updatedJob.Error)
	}

	md := decodeJobMetadata(t, updatedJob)
	if md.Phase != "completed" {
		t.Errorf("phase = %q, want completed", md.Phase)
	}
	if md.DeckItemsChecked != nil {
		t.Errorf("deck_items_checked = %d, want nil on error", *md.DeckItemsChecked)
	}
	if md.DeckItemsMoved != nil {
		t.Errorf("deck_items_moved = %d, want nil on error", *md.DeckItemsMoved)
	}
	if !strings.Contains(md.DeckResortError, "deck_items") {
		t.Errorf("deck_resort_error = %q, want error mentioning deck_items", md.DeckResortError)
	}

	if status, _ := settingsService.Get(context.Background(), "bulk_data_update_last_status"); status != "success" {
		t.Errorf("bulk_data_update_last_status = %q, want success", status)
	}
}

func TestBulkDataService_DownloadAndImport_FailedImportDoesNotReclassify(t *testing.T) {
	service, jobService, _, db := setupBulkDataServiceTest(t)
	item := seedTokenInMain(t, db)
	// The token imports fine, but the failure rate fails the whole job.
	serveBulkData(t, service, []scryfall.Card{
		rabbitToken,
		{OracleID: "oracle-2", Name: "invalid-1"}, // missing ID
		{OracleID: "oracle-3", Name: "invalid-2"},
		{OracleID: "oracle-4", Name: "invalid-3"},
	})

	job, _ := jobService.Create(context.Background(), models.JobTypeBulkDataImport, "{}")
	if err := service.DownloadAndImport(context.Background(), job.ID); err == nil {
		t.Fatal("expected the import to fail on its failure rate")
	}

	if zone := deckItemZone(t, db, item.ID); zone != models.ZoneMain {
		t.Errorf("token zone after failed import = %q, want %q", zone, models.ZoneMain)
	}

	updatedJob, err := jobService.Get(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("failed to reload job: %v", err)
	}
	md := decodeJobMetadata(t, updatedJob)
	if md.DeckItemsChecked != nil || md.DeckItemsMoved != nil || md.DeckResortError != "" {
		t.Errorf("expected no deck resort metadata on failed import, got checked=%v moved=%v err=%q",
			md.DeckItemsChecked, md.DeckItemsMoved, md.DeckResortError)
	}
}
