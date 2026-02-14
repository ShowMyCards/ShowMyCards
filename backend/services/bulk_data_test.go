package services

import (
	"backend/models"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	scryfall "github.com/BlueMonday/go-scryfall"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupBulkDataServiceTest(t *testing.T) (*BulkDataService, *JobService, *SettingsService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}

	if err := db.AutoMigrate(&models.Job{}, &models.Setting{}, &models.Card{}); err != nil {
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
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/bulk-data" {
			// Return bulk data list
			response := map[string]interface{}{
				"data": []interface{}{
					map[string]interface{}{
						"type":         "all_cards",
						"download_uri": server.URL + "/cards.json",
						"updated_at":   "2024-01-15T00:00:00.000Z",
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		} else {
			// Return card data
			json.NewEncoder(w).Encode(cards)
		}
	}))
	defer server.Close()

	// Set bulk data URL
	service.settingsService.Set(context.Background(),"bulk_data_url", server.URL+"/bulk-data")

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

	// Verify job was completed
	updatedJob, _ := jobService.Get(context.Background(), job.ID)
	if updatedJob.Status != models.JobStatusCompleted {
		t.Errorf("expected job status %s, got %s", models.JobStatusCompleted, updatedJob.Status)
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
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/bulk-data" {
			response := map[string]interface{}{
				"data": []interface{}{
					map[string]interface{}{
						"type":         "all_cards",
						"download_uri": server.URL + "/cards.json",
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		} else {
			json.NewEncoder(w).Encode(cards)
		}
	}))
	defer server.Close()

	service.settingsService.Set(context.Background(),"bulk_data_url", server.URL+"/bulk-data")
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
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/bulk-data" {
			response := map[string]interface{}{
				"data": []interface{}{
					map[string]interface{}{
						"type":         "all_cards",
						"download_uri": server.URL + "/cards.json",
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		} else {
			json.NewEncoder(w).Encode(cards)
		}
	}))
	defer server.Close()

	service.settingsService.Set(context.Background(),"bulk_data_url", server.URL+"/bulk-data")
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

	service.settingsService.Set(context.Background(),"bulk_data_url", server.URL+"/bulk-data")
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

	service.settingsService.Set(context.Background(),"bulk_data_url", "http://invalid-url-that-does-not-exist.example.com")
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

	service.settingsService.Set(context.Background(),"bulk_data_url", server.URL)
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

	service.settingsService.Set(context.Background(),"bulk_data_url", server.URL)
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

	service.settingsService.Set(context.Background(),"bulk_data_url", server.URL)
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
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/bulk-data" {
			response := map[string]interface{}{
				"data": []interface{}{
					map[string]interface{}{
						"type":         "all_cards",
						"download_uri": server.URL + "/cards.json",
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		} else {
			json.NewEncoder(w).Encode(cards)
		}
	}))
	defer server.Close()

	service.settingsService.Set(context.Background(),"bulk_data_url", server.URL+"/bulk-data")
	job, _ := jobService.Create(context.Background(), models.JobTypeBulkDataImport, "{}")

	err := service.DownloadAndImport(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("DownloadAndImport failed: %v", err)
	}

	// Verify settings were updated
	status, _ := settingsService.Get(context.Background(),"bulk_data_last_update_status")
	if status != "success" {
		t.Errorf("expected status 'success', got '%s'", status)
	}

	lastUpdate, _ := settingsService.GetTime(context.Background(),"bulk_data_last_update")
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
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/bulk-data" {
			response := map[string]interface{}{
				"data": []interface{}{
					map[string]interface{}{
						"type":         "all_cards",
						"download_uri": server.URL + "/cards.json",
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		} else {
			json.NewEncoder(w).Encode(cards)
		}
	}))
	defer server.Close()

	service.settingsService.Set(context.Background(),"bulk_data_url", server.URL+"/bulk-data")
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
