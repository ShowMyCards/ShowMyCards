package api

import (
	"backend/models"
	"backend/services"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupBulkDataTestApp(t *testing.T) (*fiber.App, *services.BulkDataService, *services.JobService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	// The async import goroutine and the test goroutine both touch this db.
	// Pinning the pool to a single connection serialises them on the same
	// in-memory database — matching backend/database/client.go — and avoids
	// the SQLITE_LOCKED contention that cache=shared exposed on Linux CI.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get database instance: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(&models.Job{}, &models.Setting{}, &models.Card{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	jobService := services.NewJobService(db)
	settingsService := services.NewSettingsService(db)
	bulkDataService := services.NewBulkDataService(db, jobService, settingsService)

	handler := NewBulkDataHandler(bulkDataService)
	appCtx := context.Background()

	app := fiber.New()
	app.Post("/api/bulk-data/import", func(c fiber.Ctx) error {
		return handler.TriggerImport(c, appCtx)
	})

	return app, bulkDataService, jobService, db
}

// TriggerImport tests

func TestBulkDataTriggerImport_Success(t *testing.T) {
	app, _, jobService, _ := setupBulkDataTestApp(t)

	req := httptest.NewRequest("POST", "/api/bulk-data/import", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d. Body: %s", fiber.StatusAccepted, resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	message, ok := result["message"].(string)
	if !ok {
		t.Fatal("expected message field in response")
	}

	if message != "Import job started" {
		t.Errorf("expected message 'Import job started', got '%s'", message)
	}

	jobID, ok := result["job_id"].(float64)
	if !ok {
		t.Fatal("expected job_id field in response")
	}

	if jobID <= 0 {
		t.Errorf("expected positive job_id, got %f", jobID)
	}

	// Verify job was created
	job, err := jobService.Get(context.Background(), uint(jobID))
	if err != nil {
		t.Fatalf("failed to get created job: %v", err)
	}

	if job.Type != models.JobTypeBulkDataImport {
		t.Errorf("expected job type %s, got %s", models.JobTypeBulkDataImport, job.Type)
	}

	// Status may be pending or in_progress depending on goroutine timing
	if job.Status != models.JobStatusPending && job.Status != models.JobStatusInProgress {
		t.Errorf("expected job status %s or %s, got %s", models.JobStatusPending, models.JobStatusInProgress, job.Status)
	}
}

// Note: Duplicate prevention and async behavior are tested at the service layer
// Handler tests focus on API contract: request/response format and job creation
