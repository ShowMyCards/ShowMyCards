package api

import (
	"backend/models"
	"backend/services"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupJobsTestApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := db.AutoMigrate(&models.Job{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	jobService := services.NewJobService(db)
	handler := NewJobsHandler(jobService)

	app := fiber.New()
	app.Get("/jobs", handler.GetAll)
	app.Get("/jobs/:id", handler.Get)

	return app, db
}

// List tests

func TestJobsList_Empty(t *testing.T) {
	app, _ := setupJobsTestApp(t)

	req := httptest.NewRequest("GET", "/jobs", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	jobs, ok := result["data"].([]interface{})
	if !ok {
		t.Fatal("expected data array in response")
	}

	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}

	totalItems, ok := result["total_items"].(float64)
	if !ok {
		t.Fatal("expected total_items field in response")
	}

	if totalItems != 0 {
		t.Errorf("expected total_items 0, got %f", totalItems)
	}
}

func TestJobsList_WithJobs(t *testing.T) {
	app, db := setupJobsTestApp(t)

	// Create multiple jobs with different statuses
	job1 := &models.Job{
		Type:     models.JobTypeBulkDataImport,
		Status:   models.JobStatusPending,
		Metadata: "{}",
	}
	db.Create(job1)

	job2 := &models.Job{
		Type:     models.JobTypeBulkDataImport,
		Status:   models.JobStatusInProgress,
		Metadata: `{"total_cards": 100}`,
	}
	db.Create(job2)

	job3 := &models.Job{
		Type:     models.JobTypeBulkDataImport,
		Status:   models.JobStatusCompleted,
		Metadata: `{"total_cards": 100, "processed_cards": 100}`,
	}
	db.Create(job3)

	req := httptest.NewRequest("GET", "/jobs", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	jobs, ok := result["data"].([]interface{})
	if !ok {
		t.Fatal("expected data array in response")
	}

	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(jobs))
	}

	totalItems, ok := result["total_items"].(float64)
	if !ok {
		t.Fatal("expected total_items field in response")
	}

	if totalItems != 3 {
		t.Errorf("expected total_items 3, got %f", totalItems)
	}
}

func TestJobsList_Pagination(t *testing.T) {
	app, db := setupJobsTestApp(t)

	// Create 5 jobs
	for i := 0; i < 5; i++ {
		job := &models.Job{
			Type:     models.JobTypeBulkDataImport,
			Status:   models.JobStatusPending,
			Metadata: "{}",
		}
		db.Create(job)
		// Small delay to ensure different created_at times
		time.Sleep(1 * time.Millisecond)
	}

	// Get page 2 with page size 2
	req := httptest.NewRequest("GET", "/jobs?page=2&page_size=2", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	jobs, ok := result["data"].([]interface{})
	if !ok {
		t.Fatal("expected data array in response")
	}

	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs on page 2, got %d", len(jobs))
	}

	totalItems, ok := result["total_items"].(float64)
	if !ok {
		t.Fatal("expected total_items field in response")
	}

	if totalItems != 5 {
		t.Errorf("expected total_items 5, got %f", totalItems)
	}
}

func TestJobsList_StatusFilter_Pending(t *testing.T) {
	app, db := setupJobsTestApp(t)

	// Create jobs with different statuses
	pending1 := &models.Job{
		Type:     models.JobTypeBulkDataImport,
		Status:   models.JobStatusPending,
		Metadata: "{}",
	}
	db.Create(pending1)

	pending2 := &models.Job{
		Type:     models.JobTypeBulkDataImport,
		Status:   models.JobStatusPending,
		Metadata: "{}",
	}
	db.Create(pending2)

	completed := &models.Job{
		Type:     models.JobTypeBulkDataImport,
		Status:   models.JobStatusCompleted,
		Metadata: "{}",
	}
	db.Create(completed)

	// Filter by pending status
	req := httptest.NewRequest("GET", "/jobs?status=pending", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	jobs, ok := result["data"].([]interface{})
	if !ok {
		t.Fatal("expected data array in response")
	}

	if len(jobs) != 2 {
		t.Errorf("expected 2 pending jobs, got %d", len(jobs))
	}

	// Verify all jobs are pending
	for _, j := range jobs {
		jobMap, ok := j.(map[string]interface{})
		if !ok {
			t.Fatal("expected job to be object")
		}

		status, ok := jobMap["status"].(string)
		if !ok {
			t.Fatal("expected status field")
		}

		if status != string(models.JobStatusPending) {
			t.Errorf("expected status pending, got %s", status)
		}
	}
}

func TestJobsList_StatusFilter_InProgress(t *testing.T) {
	app, db := setupJobsTestApp(t)

	// Create jobs with different statuses
	inProgress := &models.Job{
		Type:     models.JobTypeBulkDataImport,
		Status:   models.JobStatusInProgress,
		Metadata: "{}",
	}
	db.Create(inProgress)

	pending := &models.Job{
		Type:     models.JobTypeBulkDataImport,
		Status:   models.JobStatusPending,
		Metadata: "{}",
	}
	db.Create(pending)

	// Filter by in_progress status
	req := httptest.NewRequest("GET", "/jobs?status=in_progress", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	jobs, ok := result["data"].([]interface{})
	if !ok {
		t.Fatal("expected data array in response")
	}

	if len(jobs) != 1 {
		t.Errorf("expected 1 in_progress job, got %d", len(jobs))
	}
}

func TestJobsList_StatusFilter_Completed(t *testing.T) {
	app, db := setupJobsTestApp(t)

	// Create jobs with different statuses
	completed1 := &models.Job{
		Type:     models.JobTypeBulkDataImport,
		Status:   models.JobStatusCompleted,
		Metadata: "{}",
	}
	db.Create(completed1)

	completed2 := &models.Job{
		Type:     models.JobTypeBulkDataImport,
		Status:   models.JobStatusCompleted,
		Metadata: "{}",
	}
	db.Create(completed2)

	failed := &models.Job{
		Type:     models.JobTypeBulkDataImport,
		Status:   models.JobStatusFailed,
		Metadata: "{}",
		Error:    "Test error",
	}
	db.Create(failed)

	// Filter by completed status
	req := httptest.NewRequest("GET", "/jobs?status=completed", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	jobs, ok := result["data"].([]interface{})
	if !ok {
		t.Fatal("expected data array in response")
	}

	if len(jobs) != 2 {
		t.Errorf("expected 2 completed jobs, got %d", len(jobs))
	}
}

func TestJobsList_StatusFilter_Failed(t *testing.T) {
	app, db := setupJobsTestApp(t)

	// Create jobs with different statuses
	failed := &models.Job{
		Type:     models.JobTypeBulkDataImport,
		Status:   models.JobStatusFailed,
		Metadata: "{}",
		Error:    "Download failed",
	}
	db.Create(failed)

	completed := &models.Job{
		Type:     models.JobTypeBulkDataImport,
		Status:   models.JobStatusCompleted,
		Metadata: "{}",
	}
	db.Create(completed)

	// Filter by failed status
	req := httptest.NewRequest("GET", "/jobs?status=failed", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	jobs, ok := result["data"].([]interface{})
	if !ok {
		t.Fatal("expected data array in response")
	}

	if len(jobs) != 1 {
		t.Errorf("expected 1 failed job, got %d", len(jobs))
	}

	// Verify error field is present
	jobMap, ok := jobs[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected job to be object")
	}

	errorMsg, ok := jobMap["error"].(string)
	if !ok {
		t.Fatal("expected error field")
	}

	if errorMsg != "Download failed" {
		t.Errorf("expected error 'Download failed', got '%s'", errorMsg)
	}
}

// Get tests

func TestJobsGet_Success(t *testing.T) {
	app, db := setupJobsTestApp(t)

	// Create job with metadata
	now := time.Now()
	metadata := `{"total_cards": 1000, "processed_cards": 500, "phase": "importing"}`
	job := &models.Job{
		Type:     models.JobTypeBulkDataImport,
		Status:   models.JobStatusInProgress,
		Metadata: metadata,
	}
	job.StartedAt = &now
	db.Create(job)

	req := httptest.NewRequest("GET", "/jobs/"+strconv.Itoa(int(job.ID)), nil)
	req.SetPathValue("id", strconv.Itoa(int(job.ID)))

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d. Body: %s", fiber.StatusOK, resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	var returnedJob models.Job
	if err := json.Unmarshal(body, &returnedJob); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if returnedJob.ID != job.ID {
		t.Errorf("expected job ID %d, got %d", job.ID, returnedJob.ID)
	}

	if returnedJob.Type != models.JobTypeBulkDataImport {
		t.Errorf("expected type %s, got %s", models.JobTypeBulkDataImport, returnedJob.Type)
	}

	if returnedJob.Status != models.JobStatusInProgress {
		t.Errorf("expected status %s, got %s", models.JobStatusInProgress, returnedJob.Status)
	}

	if returnedJob.Metadata != metadata {
		t.Errorf("expected metadata %s, got %s", metadata, returnedJob.Metadata)
	}

	if returnedJob.StartedAt == nil {
		t.Error("expected started_at to be set")
	}
}

func TestJobsGet_NotFound(t *testing.T) {
	app, _ := setupJobsTestApp(t)

	req := httptest.NewRequest("GET", "/jobs/999", nil)
	req.SetPathValue("id", "999")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("expected status %d, got %d", fiber.StatusNotFound, resp.StatusCode)
	}
}

func TestJobsGet_InvalidID(t *testing.T) {
	app, _ := setupJobsTestApp(t)

	req := httptest.NewRequest("GET", "/jobs/invalid", nil)
	req.SetPathValue("id", "invalid")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected status %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}
}

// Test job ordering (newest first)

func TestJobsList_OrderedByCreatedAtDesc(t *testing.T) {
	app, db := setupJobsTestApp(t)

	// Create jobs with explicit timing
	oldJob := &models.Job{
		Type:     models.JobTypeBulkDataImport,
		Status:   models.JobStatusCompleted,
		Metadata: `{"order": "old"}`,
	}
	db.Create(oldJob)
	time.Sleep(10 * time.Millisecond)

	middleJob := &models.Job{
		Type:     models.JobTypeBulkDataImport,
		Status:   models.JobStatusInProgress,
		Metadata: `{"order": "middle"}`,
	}
	db.Create(middleJob)
	time.Sleep(10 * time.Millisecond)

	newJob := &models.Job{
		Type:     models.JobTypeBulkDataImport,
		Status:   models.JobStatusPending,
		Metadata: `{"order": "new"}`,
	}
	db.Create(newJob)

	req := httptest.NewRequest("GET", "/jobs", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	jobs, ok := result["data"].([]interface{})
	if !ok {
		t.Fatal("expected data array in response")
	}

	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(jobs))
	}

	// Verify newest job is first
	firstJob, ok := jobs[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected job to be object")
	}

	firstMetadata, ok := firstJob["metadata"].(string)
	if !ok {
		t.Fatal("expected metadata field")
	}

	if firstMetadata != `{"order": "new"}` {
		t.Errorf("expected newest job first, got metadata: %s", firstMetadata)
	}

	// Verify oldest job is last
	lastJob, ok := jobs[2].(map[string]interface{})
	if !ok {
		t.Fatal("expected job to be object")
	}

	lastMetadata, ok := lastJob["metadata"].(string)
	if !ok {
		t.Fatal("expected metadata field")
	}

	if lastMetadata != `{"order": "old"}` {
		t.Errorf("expected oldest job last, got metadata: %s", lastMetadata)
	}
}
