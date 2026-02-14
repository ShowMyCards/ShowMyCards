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

func setupSchedulerTestApp(t *testing.T) (*fiber.App, *services.SettingsService, *services.JobService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := db.AutoMigrate(&models.Job{}, &models.Setting{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	settingsService := services.NewSettingsService(db)
	jobService := services.NewJobService(db)

	handler := NewSchedulerHandler(settingsService, jobService)

	app := fiber.New()
	app.Get("/scheduler/tasks", handler.GetScheduledTasks)

	return app, settingsService, jobService, db
}

// GetScheduledTasks tests

func TestScheduler_GetScheduledTasks_Success(t *testing.T) {
	app, _, _, _ := setupSchedulerTestApp(t)

	req := httptest.NewRequest("GET", "/scheduler/tasks", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var tasks []ScheduledTaskInfo
	if err := json.Unmarshal(body, &tasks); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	// Verify bulk data update task
	bulkDataTask := tasks[0]
	if bulkDataTask.Name != "Bulk Data Auto-Update" {
		t.Errorf("expected task name 'Bulk Data Auto-Update', got '%s'", bulkDataTask.Name)
	}

	if bulkDataTask.Type != "bulk_data_update" {
		t.Errorf("expected task type 'bulk_data_update', got '%s'", bulkDataTask.Type)
	}

	if !bulkDataTask.Enabled {
		t.Error("expected bulk data task to be enabled by default")
	}

	if bulkDataTask.Schedule != "03:00 daily" {
		t.Errorf("expected schedule '03:00 daily', got '%s'", bulkDataTask.Schedule)
	}

	// Verify job cleanup task
	cleanupTask := tasks[1]
	if cleanupTask.Name != "Job History Cleanup" {
		t.Errorf("expected task name 'Job History Cleanup', got '%s'", cleanupTask.Name)
	}

	if cleanupTask.Type != "job_cleanup" {
		t.Errorf("expected task type 'job_cleanup', got '%s'", cleanupTask.Type)
	}

	if !cleanupTask.Enabled {
		t.Error("expected job cleanup task to be enabled")
	}

	if cleanupTask.Schedule != "00:00 daily" {
		t.Errorf("expected schedule '00:00 daily', got '%s'", cleanupTask.Schedule)
	}
}

func TestScheduler_GetScheduledTasks_WithBulkDataDisabled(t *testing.T) {
	app, settingsService, _, _ := setupSchedulerTestApp(t)

	// Disable bulk data auto-update
	settingsService.Set(context.Background(),"bulk_data_auto_update", "false")

	req := httptest.NewRequest("GET", "/scheduler/tasks", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var tasks []ScheduledTaskInfo
	json.Unmarshal(body, &tasks)

	bulkDataTask := tasks[0]
	if bulkDataTask.Enabled {
		t.Error("expected bulk data task to be disabled")
	}
}

func TestScheduler_GetScheduledTasks_WithCustomSchedule(t *testing.T) {
	app, settingsService, _, _ := setupSchedulerTestApp(t)

	// Set custom update time
	settingsService.Set(context.Background(),"bulk_data_update_time", "05:30")

	req := httptest.NewRequest("GET", "/scheduler/tasks", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var tasks []ScheduledTaskInfo
	json.Unmarshal(body, &tasks)

	bulkDataTask := tasks[0]
	if bulkDataTask.Schedule != "05:30 daily" {
		t.Errorf("expected schedule '05:30 daily', got '%s'", bulkDataTask.Schedule)
	}
}

func TestScheduler_GetScheduledTasks_WithLastJob(t *testing.T) {
	app, _, jobService, _ := setupSchedulerTestApp(t)

	// Create a completed bulk data import job
	ctx := context.Background()
	job, _ := jobService.Create(ctx, models.JobTypeBulkDataImport, "{}")
	jobService.Start(ctx, job.ID)
	jobService.Complete(ctx, job.ID)

	req := httptest.NewRequest("GET", "/scheduler/tasks", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var tasks []ScheduledTaskInfo
	json.Unmarshal(body, &tasks)

	bulkDataTask := tasks[0]
	if bulkDataTask.LastJobID == nil {
		t.Fatal("expected last_job_id to be set")
	}

	if *bulkDataTask.LastJobID != job.ID {
		t.Errorf("expected last_job_id %d, got %d", job.ID, *bulkDataTask.LastJobID)
	}

	if bulkDataTask.LastJobStatus == nil {
		t.Fatal("expected last_job_status to be set")
	}

	if *bulkDataTask.LastJobStatus != string(models.JobStatusCompleted) {
		t.Errorf("expected last_job_status 'completed', got '%s'", *bulkDataTask.LastJobStatus)
	}

	if bulkDataTask.LastRun == nil {
		t.Error("expected last_run to be set")
	}
}

func TestScheduler_GetScheduledTasks_NextRunCalculation(t *testing.T) {
	app, _, _, _ := setupSchedulerTestApp(t)

	req := httptest.NewRequest("GET", "/scheduler/tasks", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var tasks []ScheduledTaskInfo
	json.Unmarshal(body, &tasks)

	// Verify next_run is set for both tasks
	for _, task := range tasks {
		if task.NextRun.IsZero() {
			t.Errorf("expected next_run to be set for task '%s'", task.Name)
		}

		// Next run should be in the future
		if !task.NextRun.After(task.NextRun.Add(-24 * 60 * 60)) {
			t.Errorf("expected next_run to be in the future for task '%s'", task.Name)
		}
	}
}
