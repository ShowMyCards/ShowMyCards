package api

import (
	"backend/models"
	"backend/services"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeTaskProvider []services.ScheduledTask

func (f fakeTaskProvider) Tasks() []services.ScheduledTask { return f }

// RunNow mimics services.Scheduler.RunNow for the registered fake tasks.
func (f fakeTaskProvider) RunNow(ctx context.Context, name string) (uint, error) {
	for _, task := range f {
		if task.Name == name {
			return task.Run(ctx)
		}
	}
	return 0, services.ErrTaskNotFound
}

// testTasks mirrors the shape of the real registry without needing the
// services the real scheduler depends on.
var testTasks = fakeTaskProvider{
	{Name: "bulk_data_update", DisplayName: "Card Data Update", Interval: 24 * time.Hour, JobType: models.JobTypeBulkDataImport},
	{Name: "set_data_update", DisplayName: "Set Data Update", Interval: 14 * 24 * time.Hour, JobType: models.JobTypeSetDataImport},
	{Name: "data_backup", DisplayName: "Data Backup", Interval: 24 * time.Hour,
		Extras: []services.TaskSetting{{Key: "data_backup_retention_count", Label: "Backups to keep", Kind: "int"}}},
	{Name: "always_running", DisplayName: "Busy", Interval: 24 * time.Hour,
		Run: func(context.Context) (uint, error) { return 0, services.ErrTaskRunning }},
	{Name: "broken", DisplayName: "Broken", Interval: 24 * time.Hour,
		Run: func(context.Context) (uint, error) { return 0, errors.New("disk full") }},
	{Name: "quick", DisplayName: "Quick", Interval: 24 * time.Hour,
		Run: func(context.Context) (uint, error) { return 42, nil }},
}

func setupSchedulerTestApp(t *testing.T) (*fiber.App, *services.SettingsService, *services.JobService) {
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

	handler := NewSchedulerHandler(context.Background(), testTasks, settingsService, jobService)

	app := fiber.New()
	app.Get("/api/scheduler/tasks", handler.GetScheduledTasks)
	app.Post("/api/scheduler/tasks/:name/run", handler.RunTask)
	app.Post("/api/legacy/import", handler.RunNamedTask("quick"))

	return app, settingsService, jobService
}

func getTasks(t *testing.T, app *fiber.App) map[string]ScheduledTaskInfo {
	t.Helper()

	resp, err := app.Test(httptest.NewRequest("GET", "/api/scheduler/tasks", nil))
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var tasks []ScheduledTaskInfo
	if err := json.Unmarshal(body, &tasks); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	byName := make(map[string]ScheduledTaskInfo, len(tasks))
	for _, task := range tasks {
		byName[task.Name] = task
	}
	return byName
}

func TestScheduler_GetScheduledTasks_ReturnsEveryRegisteredTask(t *testing.T) {
	app, _, _ := setupSchedulerTestApp(t)

	tasks := getTasks(t, app)
	if len(tasks) != len(testTasks) {
		t.Fatalf("expected %d tasks, got %d", len(testTasks), len(tasks))
	}
	for _, registered := range testTasks {
		got, ok := tasks[registered.Name]
		if !ok {
			t.Errorf("task %s missing from response", registered.Name)
			continue
		}
		if got.DisplayName != registered.DisplayName {
			t.Errorf("task %s: display_name = %q, want %q", registered.Name, got.DisplayName, registered.DisplayName)
		}
	}
}

func TestScheduler_GetScheduledTasks_DefaultsFromSettings(t *testing.T) {
	app, _, _ := setupSchedulerTestApp(t)

	tests := []struct {
		name     string
		enabled  bool
		time     string
		schedule string
	}{
		{"bulk_data_update", true, "03:00", "03:00 daily"},
		{"set_data_update", true, "02:30", "02:30 every 14 days"},
		{"data_backup", true, "03:30", "03:30 daily"},
	}

	tasks := getTasks(t, app)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tasks[tt.name]
			if got.Enabled != tt.enabled {
				t.Errorf("enabled = %v, want %v", got.Enabled, tt.enabled)
			}
			if got.Time != tt.time {
				t.Errorf("time = %q, want %q", got.Time, tt.time)
			}
			if got.EnabledKey != tt.name+"_enabled" || got.TimeKey != tt.name+"_time" {
				t.Errorf("keys = %q/%q, want %s_enabled/%s_time", got.EnabledKey, got.TimeKey, tt.name, tt.name)
			}
			if got.Schedule != tt.schedule {
				t.Errorf("schedule = %q, want %q", got.Schedule, tt.schedule)
			}
			if got.NextRun == nil || !got.NextRun.After(time.Now()) {
				t.Errorf("expected next_run in the future, got %v", got.NextRun)
			}
		})
	}
}

func TestScheduler_GetScheduledTasks_Disabled(t *testing.T) {
	app, settingsService, _ := setupSchedulerTestApp(t)

	settingsService.Set(context.Background(), "data_backup_enabled", "false")

	got := getTasks(t, app)["data_backup"]
	if got.Enabled {
		t.Error("expected data_backup to be disabled")
	}
	if got.NextRun != nil {
		t.Errorf("expected no next_run for a disabled task, got %v", got.NextRun)
	}
}

func TestScheduler_GetScheduledTasks_CustomTime(t *testing.T) {
	app, settingsService, _ := setupSchedulerTestApp(t)

	settingsService.Set(context.Background(), "bulk_data_update_time", "05:30")

	got := getTasks(t, app)["bulk_data_update"]
	if got.Schedule != "05:30 daily" {
		t.Errorf("expected schedule '05:30 daily', got %q", got.Schedule)
	}
	if got.NextRun == nil || got.NextRun.Hour() != 5 || got.NextRun.Minute() != 30 {
		t.Errorf("expected next_run at 05:30, got %v", got.NextRun)
	}
}

func TestScheduler_GetScheduledTasks_LastRunAndStatus(t *testing.T) {
	app, settingsService, _ := setupSchedulerTestApp(t)

	ctx := context.Background()
	lastRun := time.Now().Add(-time.Hour).Truncate(time.Second)
	settingsService.SetTime(ctx, "data_backup_last_run", lastRun)
	settingsService.Set(ctx, "data_backup_last_status", "success")

	got := getTasks(t, app)["data_backup"]
	if got.LastRun == nil || !got.LastRun.Equal(lastRun) {
		t.Errorf("last_run = %v, want %v", got.LastRun, lastRun)
	}
	if got.LastStatus == nil || *got.LastStatus != "success" {
		t.Errorf("last_status = %v, want success", got.LastStatus)
	}
	if got.JobType != nil || got.LastJobID != nil {
		t.Error("expected no job fields for a task without a job type")
	}
}

func TestScheduler_GetScheduledTasks_WithLastJob(t *testing.T) {
	app, _, jobService := setupSchedulerTestApp(t)

	ctx := context.Background()
	job, _ := jobService.Create(ctx, models.JobTypeBulkDataImport, "{}")
	jobService.Start(ctx, job.ID)
	jobService.Complete(ctx, job.ID)

	got := getTasks(t, app)["bulk_data_update"]
	if got.JobType == nil || *got.JobType != string(models.JobTypeBulkDataImport) {
		t.Errorf("job_type = %v, want %s", got.JobType, models.JobTypeBulkDataImport)
	}
	if got.LastJobID == nil || *got.LastJobID != job.ID {
		t.Fatalf("last_job_id = %v, want %d", got.LastJobID, job.ID)
	}
	if got.LastJobStatus == nil || *got.LastJobStatus != string(models.JobStatusCompleted) {
		t.Errorf("last_job_status = %v, want completed", got.LastJobStatus)
	}
}

// RunTask tests

func TestScheduler_RunTask(t *testing.T) {
	app, _, _ := setupSchedulerTestApp(t)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantJobID  uint
	}{
		{"starts the task", "/api/scheduler/tasks/quick/run", fiber.StatusAccepted, 42},
		{"legacy alias starts its task", "/api/legacy/import", fiber.StatusAccepted, 42},
		{"unknown task", "/api/scheduler/tasks/nope/run", fiber.StatusNotFound, 0},
		{"already running", "/api/scheduler/tasks/always_running/run", fiber.StatusConflict, 0},
		{"run failure", "/api/scheduler/tasks/broken/run", fiber.StatusInternalServerError, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest("POST", tt.path, nil))
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", resp.StatusCode, tt.wantStatus, body)
			}
			if tt.wantStatus != fiber.StatusAccepted {
				return
			}
			var result RunTaskResponse
			if err := json.Unmarshal(body, &result); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			if result.JobID != tt.wantJobID {
				t.Errorf("job_id = %d, want %d", result.JobID, tt.wantJobID)
			}
		})
	}
}

func TestScheduler_GetScheduledTasks_Extras(t *testing.T) {
	app, settingsService, _ := setupSchedulerTestApp(t)
	settingsService.Set(context.Background(), "data_backup_retention_count", "5")

	tasks := getTasks(t, app)

	extras := tasks["data_backup"].Extras
	if len(extras) != 1 {
		t.Fatalf("expected 1 extra, got %d", len(extras))
	}
	got := extras[0]
	if got.Key != "data_backup_retention_count" || got.Value != "5" || got.Kind != "int" {
		t.Errorf("extra = %+v, want data_backup_retention_count=5 (int)", got)
	}
	if got.Min == nil || *got.Min != 0 {
		t.Errorf("min = %v, want 0", got.Min)
	}

	if extras := tasks["bulk_data_update"].Extras; extras == nil || len(extras) != 0 {
		t.Errorf("expected an empty (non-null) extras list, got %v", extras)
	}
}
