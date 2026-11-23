package services

import (
	"backend/models"
	"backend/scryfall"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSchedulerTest(t *testing.T) (*Scheduler, *BulkDataService, *JobService, *SettingsService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}

	// Tasks finish in background goroutines. Each pooled connection to
	// ":memory:" is a separate database, so pin the pool to one connection.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get database instance: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(&models.Job{}, &models.Setting{}, &models.Card{}, &models.Set{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	jobService := NewJobService(db)
	settingsService := NewSettingsService(db)
	bulkDataService := NewBulkDataService(db, jobService, settingsService)
	scryfallClient, err := scryfall.NewClient()
	if err != nil {
		t.Fatalf("failed to create scryfall client: %v", err)
	}
	setDataService := NewSetDataService(db, jobService, settingsService, scryfallClient, t.TempDir())
	symbolDataService := NewSymbolDataService(db, jobService, settingsService, scryfallClient)
	backupService := NewBackupService(stubExporter{}, settingsService, t.TempDir())
	scheduler := NewScheduler(bulkDataService, setDataService, symbolDataService, jobService, settingsService, backupService)

	return scheduler, bulkDataService, jobService, settingsService, db
}

// NewScheduler tests

func TestScheduler_NewScheduler_Initialization(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)

	if scheduler.bulkDataService == nil {
		t.Error("expected bulkDataService to be initialized")
	}

	if scheduler.jobService == nil {
		t.Error("expected jobService to be initialized")
	}

	if scheduler.settingsService == nil {
		t.Error("expected settingsService to be initialized")
	}

	if scheduler.lastRun == nil {
		t.Error("expected lastRun map to be initialized")
	}

	if scheduler.done == nil {
		t.Error("expected done channel to be initialized")
	}

	if len(scheduler.tasks) == 0 {
		t.Error("expected tasks to be registered")
	}
}

func TestScheduler_NewScheduler_TasksRegistered(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)

	expectedTasks := []string{"bulk_data_update", "set_data_update", "symbol_data_update", "job_cleanup", "data_backup"}
	if len(scheduler.tasks) != len(expectedTasks) {
		t.Errorf("expected %d tasks, got %d", len(expectedTasks), len(scheduler.tasks))
	}

	taskNames := make(map[string]bool)
	for _, task := range scheduler.tasks {
		taskNames[task.Name] = true
	}

	for _, expected := range expectedTasks {
		if !taskNames[expected] {
			t.Errorf("expected task %s to be registered", expected)
		}
	}
}

// Start and Stop tests

func TestScheduler_Start_AndStop(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)

	ctx, cancel := context.WithCancel(context.Background())
	scheduler.Start(ctx)

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// Cancel context to stop
	cancel()

	// Wait for graceful shutdown (with timeout)
	done := make(chan bool)
	go func() {
		scheduler.Stop()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("scheduler did not stop within timeout")
	}
}

// shouldRunTask tests

func TestScheduler_ShouldRunTask_NeverRun(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)

	task := ScheduledTask{
		Name:     "test_task",
		Interval: 24 * time.Hour,
	}

	// Task has never run, should return true
	if !scheduler.shouldRunTask(context.Background(), task, time.Now()) {
		t.Error("expected task that never ran to be runnable")
	}
}

func TestScheduler_ShouldRunTask_RecentlyRan(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)

	task := ScheduledTask{
		Name:     "test_task",
		Interval: 24 * time.Hour,
	}

	// Mark task as recently run
	scheduler.lastRunMu.Lock()
	scheduler.lastRun["test_task"] = time.Now()
	scheduler.lastRunMu.Unlock()

	// Task ran recently, should return false
	if scheduler.shouldRunTask(context.Background(), task, time.Now()) {
		t.Error("expected task that recently ran to not be runnable")
	}
}

func TestScheduler_ShouldRunTask_IntervalElapsed(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)

	task := ScheduledTask{
		Name:     "test_task",
		Interval: 1 * time.Hour,
	}

	// Mark task as run 2 hours ago
	scheduler.lastRunMu.Lock()
	scheduler.lastRun["test_task"] = time.Now().Add(-2 * time.Hour)
	scheduler.lastRunMu.Unlock()

	// Interval elapsed, should return true
	if !scheduler.shouldRunTask(context.Background(), task, time.Now()) {
		t.Error("expected task with elapsed interval to be runnable")
	}
}

func TestScheduler_ShouldRunTask_PersistedLastRun(t *testing.T) {
	scheduler, _, _, settingsService, _ := setupSchedulerTest(t)

	task := ScheduledTask{
		Name:     "test_task",
		Interval: 24 * time.Hour,
	}

	// Set persisted last run to recent time
	settingsService.SetTime(context.Background(), "test_task_last_run", time.Now())

	// Should check persisted time and return false
	if scheduler.shouldRunTask(context.Background(), task, time.Now()) {
		t.Error("expected task with recent persisted run to not be runnable")
	}
}

// isInTimeWindow tests

func TestScheduler_IsInTimeWindow_LiteralTime_InWindow(t *testing.T) {
	now := time.Now()
	timeStr := now.Format("15:04")

	if !isInTimeWindow(timeStr, now) {
		t.Error("expected current time to be in window")
	}
}

func TestScheduler_IsInTimeWindow_LiteralTime_OutOfWindow(t *testing.T) {
	now := time.Now()
	// Set target to 2 hours ago
	targetTime := now.Add(-2 * time.Hour)
	timeStr := targetTime.Format("15:04")

	if isInTimeWindow(timeStr, now) {
		t.Error("expected past time to be out of window")
	}
}

// Task interval tests

func TestScheduler_TaskIntervals(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)

	expectedIntervals := map[string]time.Duration{
		"bulk_data_update": 24 * time.Hour,
		"set_data_update":  14 * 24 * time.Hour,
		"job_cleanup":      24 * time.Hour,
	}

	for _, task := range scheduler.tasks {
		expected, ok := expectedIntervals[task.Name]
		if !ok {
			continue
		}
		if task.Interval != expected {
			t.Errorf("task %s: expected interval %v, got %v", task.Name, expected, task.Interval)
		}
	}
}

// shouldRunTask edge case tests

func TestScheduler_ShouldRunTask_IntervalExactlyElapsed(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)

	task := ScheduledTask{
		Name:     "test_task",
		Interval: 1 * time.Hour,
	}

	// Mark task as run exactly 1 hour ago (interval boundary)
	scheduler.lastRunMu.Lock()
	scheduler.lastRun["test_task"] = time.Now().Add(-1 * time.Hour)
	scheduler.lastRunMu.Unlock()

	// At the exact boundary, time.Since(lastRun) >= interval, so should return true
	if !scheduler.shouldRunTask(context.Background(), task, time.Now()) {
		t.Error("expected task at exact interval boundary to be runnable")
	}
}

func TestScheduler_ShouldRunTask_IntervalNotQuiteElapsed(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)

	task := ScheduledTask{
		Name:     "test_task",
		Interval: 1 * time.Hour,
	}

	// Mark task as run 59 minutes ago (just under the 1-hour interval)
	scheduler.lastRunMu.Lock()
	scheduler.lastRun["test_task"] = time.Now().Add(-59 * time.Minute)
	scheduler.lastRunMu.Unlock()

	if scheduler.shouldRunTask(context.Background(), task, time.Now()) {
		t.Error("expected task with interval not yet elapsed to not be runnable")
	}
}

func TestScheduler_ShouldRunTask_PersistedOverdue(t *testing.T) {
	scheduler, _, _, settingsService, _ := setupSchedulerTest(t)

	task := ScheduledTask{
		Name:     "test_task",
		Interval: 24 * time.Hour,
	}

	// Set persisted last run to 48 hours ago (well past interval)
	settingsService.SetTime(context.Background(), "test_task_last_run", time.Now().Add(-48*time.Hour))

	if !scheduler.shouldRunTask(context.Background(), task, time.Now()) {
		t.Error("expected task with overdue persisted run to be runnable")
	}
}

func TestScheduler_ShouldRunTask_InMemoryOverridesPersistedRecent(t *testing.T) {
	scheduler, _, _, settingsService, _ := setupSchedulerTest(t)

	task := ScheduledTask{
		Name:     "test_task",
		Interval: 1 * time.Hour,
	}

	// Persisted says it ran 2 hours ago (overdue)
	settingsService.SetTime(context.Background(), "test_task_last_run", time.Now().Add(-2*time.Hour))

	// But in-memory says it ran just now
	scheduler.lastRunMu.Lock()
	scheduler.lastRun["test_task"] = time.Now()
	scheduler.lastRunMu.Unlock()

	// In-memory check happens first, should return false
	if scheduler.shouldRunTask(context.Background(), task, time.Now()) {
		t.Error("expected in-memory recent run to prevent task from running")
	}
}

// isInTimeWindow edge case tests

func TestScheduler_IsInTimeWindow_JustInsideWindow(t *testing.T) {
	// Create a time at exactly 14:00
	now := time.Date(2024, 6, 15, 14, 3, 0, 0, time.Local)

	// Window is 14:00 to 14:05, so 14:03 is inside
	if !isInTimeWindow("14:00", now) {
		t.Error("expected time 14:03 to be within 5-minute window of 14:00")
	}
}

func TestScheduler_IsInTimeWindow_AtWindowStart(t *testing.T) {
	// Time exactly at the window start
	now := time.Date(2024, 6, 15, 14, 0, 0, 0, time.Local)

	if !isInTimeWindow("14:00", now) {
		t.Error("expected time exactly at window start to be in window")
	}
}

func TestScheduler_IsInTimeWindow_JustOutsideWindow(t *testing.T) {
	// Time at 14:05, window is 14:00 to 14:05 (exclusive end)
	now := time.Date(2024, 6, 15, 14, 5, 0, 0, time.Local)

	if isInTimeWindow("14:00", now) {
		t.Error("expected time at 14:05 to be outside the 5-minute window ending at 14:05")
	}
}

func TestScheduler_IsInTimeWindow_WellOutsideWindow(t *testing.T) {
	// Morning time, target is afternoon
	now := time.Date(2024, 6, 15, 8, 0, 0, 0, time.Local)

	if isInTimeWindow("14:00", now) {
		t.Error("expected morning time to be outside afternoon window")
	}
}

func TestScheduler_IsInTimeWindow_InvalidFormat(t *testing.T) {
	now := time.Now()

	if isInTimeWindow("bad", now) {
		t.Error("expected an unparseable time to be out of window")
	}
}

// checkTask tests

func TestScheduler_CheckTask_DisabledBySettings(t *testing.T) {
	scheduler, _, _, settingsService, _ := setupSchedulerTest(t)

	ran := false
	task := ScheduledTask{
		Name:     "test_disabled",
		Interval: 1 * time.Millisecond,
		Run: func(ctx context.Context) (uint, error) {
			ran = true
			return 0, nil
		},
	}

	// Disable the task via settings
	settingsService.Set(context.Background(), "test_disabled_enabled", "false")

	scheduler.checkTask(context.Background(), task, false)

	if ran {
		t.Error("expected disabled task to not run")
	}
}

func TestScheduler_CheckTask_EnabledBySettings(t *testing.T) {
	scheduler, _, _, settingsService, _ := setupSchedulerTest(t)

	ran := false
	task := ScheduledTask{
		Name:     "test_enabled",
		Interval: 1 * time.Millisecond,
		Run: func(ctx context.Context) (uint, error) {
			ran = true
			return 0, nil
		},
	}

	// Enable the task via settings
	settingsService.Set(context.Background(), "test_enabled_enabled", "true")
	settingsService.Set(context.Background(), "test_enabled_time", time.Now().Format("15:04"))

	scheduler.checkTask(context.Background(), task, false)

	if !ran {
		t.Error("expected enabled task to run")
	}
}

func TestScheduler_CheckTask_CatchupSkipsTimeWindow(t *testing.T) {
	scheduler, _, _, settingsService, _ := setupSchedulerTest(t)

	ran := false
	task := ScheduledTask{
		Name:     "test_catchup",
		Interval: 1 * time.Millisecond,
		Run: func(ctx context.Context) (uint, error) {
			ran = true
			return 0, nil
		},
	}

	settingsService.Set(context.Background(), "test_catchup_enabled", "true")
	settingsService.Set(context.Background(), "test_catchup_time", time.Now().Add(-2*time.Hour).Format("15:04")) // not now

	// Running as catchup should skip the time window check
	scheduler.checkTask(context.Background(), task, true)

	if !ran {
		t.Error("expected catchup task to run even outside time window")
	}
}

func TestScheduler_CheckTask_PreventsDuplicateExecution(t *testing.T) {
	scheduler, _, _, settingsService, _ := setupSchedulerTest(t)
	settingsService.Set(context.Background(), "test_dedup_enabled", "true")

	var runCount atomic.Int32
	blocker := make(chan struct{})
	task := ScheduledTask{
		Name:     "test_dedup",
		Interval: 1 * time.Millisecond,
		Run: func(ctx context.Context) (uint, error) {
			runCount.Add(1)
			<-blocker // Block until released
			return 0, nil
		},
	}

	// Run the task in a goroutine (it will block)
	go scheduler.checkTask(context.Background(), task, true)
	time.Sleep(10 * time.Millisecond)

	// Try to run same task again while first is still running
	scheduler.checkTask(context.Background(), task, true)

	// Release the blocker
	close(blocker)
	time.Sleep(10 * time.Millisecond)

	if runCount.Load() != 1 {
		t.Errorf("expected task to run exactly once due to dedup, ran %d times", runCount.Load())
	}
}

// Integration test

func TestScheduler_Integration_DisabledTasks(t *testing.T) {
	scheduler, _, jobService, settingsService, _ := setupSchedulerTest(t)

	// Disable all auto-updates
	settingsService.Set(context.Background(), "bulk_data_update_enabled", "false")
	settingsService.Set(context.Background(), "set_data_update_enabled", "false")
	settingsService.Set(context.Background(), "symbol_data_update_enabled", "false")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	scheduler.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Wait for context timeout
	<-ctx.Done()
	scheduler.Stop()

	// Verify no jobs were created (tasks are disabled)
	_, total, _ := jobService.List(context.Background(), 1, 10, nil, nil)
	if total != 0 {
		t.Errorf("expected 0 jobs with disabled tasks, got %d", total)
	}
}

func TestScheduledTask_DerivedKeys(t *testing.T) {
	task := ScheduledTask{Name: "data_backup"}
	tests := []struct{ got, want string }{
		{task.EnabledKey(), "data_backup_enabled"},
		{task.TimeKey(), "data_backup_time"},
		{task.LastRunKey(), "data_backup_last_run"},
		{task.LastStatusKey(), "data_backup_last_status"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("got %q, want %q", tt.got, tt.want)
		}
	}
}

// Every registered task's derived keys must be seeded and whitelisted, or the
// task can never run (enabled defaults to false) and can't be configured.
func TestScheduler_TaskKeysAreSeededAndWhitelisted(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)
	valid := ValidSettingKeys()

	for _, task := range scheduler.Tasks() {
		for _, key := range []string{task.EnabledKey(), task.TimeKey(), task.LastRunKey(), task.LastStatusKey()} {
			if !valid[key] {
				t.Errorf("task %s: key %s is not a valid setting key", task.Name, key)
			}
		}
		if task.DisplayName == "" {
			t.Errorf("task %s: missing DisplayName", task.Name)
		}
	}
}

func TestNextRun(t *testing.T) {
	loc := time.Local
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, loc)
	at := func(day, hour, min int) time.Time { return time.Date(2026, 6, day, hour, min, 0, 0, loc) }
	ptr := func(t time.Time) *time.Time { return &t }

	tests := []struct {
		name     string
		timeStr  string
		interval time.Duration
		lastRun  *time.Time
		want     time.Time
	}{
		{"never run, later today", "15:00", 24 * time.Hour, nil, at(15, 15, 0)},
		{"never run, already passed today", "03:00", 24 * time.Hour, nil, at(16, 3, 0)},
		{"daily, ran this morning", "03:00", 24 * time.Hour, ptr(at(15, 3, 0)), at(16, 3, 0)},
		{"daily, finished late is pushed a day", "03:00", 24 * time.Hour, ptr(at(15, 3, 20)), at(17, 3, 0)},
		{"14 days", "02:30", 14 * 24 * time.Hour, ptr(at(10, 2, 30)), at(24, 2, 30)},
		{"30 days", "02:00", 30 * 24 * time.Hour, ptr(at(1, 2, 0)), time.Date(2026, 7, 1, 2, 0, 0, 0, loc)},
		{"overdue runs at next window", "15:00", 14 * 24 * time.Hour, ptr(at(1, 15, 0)), at(15, 15, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NextRun(tt.timeStr, tt.interval, tt.lastRun, now)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("NextRun() = %v, want %v", got, tt.want)
			}
		})
	}

	if _, err := NextRun("bad", 24*time.Hour, nil, now); err == nil {
		t.Error("expected error for invalid time")
	}
}

// Scheduled-task job records

// waitForJob polls until the job reaches a terminal status, since task work
// runs in the background after Run returns.
func waitForJob(t *testing.T, jobService *JobService, id uint) *models.Job {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		job, err := jobService.Get(context.Background(), id)
		if err != nil {
			t.Fatalf("failed to get job %d: %v", id, err)
		}
		if job.Status != models.JobStatusPending && job.Status != models.JobStatusInProgress {
			return job
		}
		if time.Now().After(deadline) {
			t.Fatalf("job %d did not finish, status %q", id, job.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func mustStart(t *testing.T, id uint, err error) uint {
	t.Helper()
	if err != nil {
		t.Fatalf("failed to start task: %v", err)
	}
	if id == 0 {
		t.Fatal("expected a job ID")
	}
	return id
}

func TestScheduler_RunBackup_RecordsCompletedJob(t *testing.T) {
	scheduler, _, jobService, settingsService, _ := setupSchedulerTest(t)
	ctx := context.Background()

	id, err := scheduler.runBackup(ctx)
	job := waitForJob(t, jobService, mustStart(t, id, err))
	if job.Type != models.JobTypeDataBackup {
		t.Errorf("type = %q, want %q", job.Type, models.JobTypeDataBackup)
	}
	if job.Status != models.JobStatusCompleted {
		t.Fatalf("status = %q, want completed (error: %q)", job.Status, job.Error)
	}
	if job.StartedAt == nil || job.CompletedAt == nil {
		t.Error("expected started_at and completed_at to be set")
	}

	var md backupMetadata
	if err := json.Unmarshal([]byte(job.Metadata), &md); err != nil {
		t.Fatalf("failed to parse metadata %q: %v", job.Metadata, err)
	}
	if !strings.HasPrefix(md.FileName, backupFilePrefix) {
		t.Errorf("file_name = %q, want prefix %q", md.FileName, backupFilePrefix)
	}
	if md.SizeBytes <= 0 {
		t.Errorf("size_bytes = %d, want > 0", md.SizeBytes)
	}
	if md.Retention != DefaultBackupRetention {
		t.Errorf("retention = %d, want %d", md.Retention, DefaultBackupRetention)
	}

	if status, _ := settingsService.Get(ctx, "data_backup_last_status"); status != TaskStatusSuccess {
		t.Errorf("data_backup_last_status = %q, want success", status)
	}
	if lastRun, _ := settingsService.GetTime(ctx, "data_backup_last_run"); lastRun == nil {
		t.Error("expected data_backup_last_run to be set")
	}
}

func TestScheduler_RunBackup_FailureRecordsFailedJob(t *testing.T) {
	scheduler, _, jobService, settingsService, _ := setupSchedulerTest(t)
	ctx := context.Background()
	scheduler.backupService = NewBackupService(stubExporter{err: errors.New("export exploded")}, settingsService, t.TempDir())

	id, err := scheduler.runBackup(ctx)
	job := waitForJob(t, jobService, mustStart(t, id, err))
	if job.Status != models.JobStatusFailed {
		t.Fatalf("status = %q, want failed", job.Status)
	}
	if !strings.Contains(job.Error, "export exploded") {
		t.Errorf("error = %q, want it to contain the cause", job.Error)
	}
	if status, _ := settingsService.Get(ctx, "data_backup_last_status"); status != TaskStatusFailed {
		t.Errorf("data_backup_last_status = %q, want failed", status)
	}
	// last_run stays unset so catch-up retries the backup.
	if lastRun, _ := settingsService.GetTime(ctx, "data_backup_last_run"); lastRun != nil {
		t.Errorf("expected data_backup_last_run to stay unset, got %v", lastRun)
	}
}

func TestScheduler_RunJobCleanup_RecordsJobAndSparesActiveJobs(t *testing.T) {
	scheduler, _, jobService, settingsService, db := setupSchedulerTest(t)
	ctx := context.Background()
	old := time.Now().AddDate(0, 0, -(DefaultJobCleanupRetentionDays + 1))

	finished := &models.Job{Type: models.JobTypeBulkDataImport, Status: models.JobStatusCompleted}
	running := &models.Job{Type: models.JobTypeSetDataImport, Status: models.JobStatusInProgress}
	db.Create(finished)
	db.Create(running)
	db.Model(&models.Job{}).Where("id IN ?", []uint{finished.ID, running.ID}).Update("created_at", old)

	id, err := scheduler.runJobCleanup(ctx)
	job := waitForJob(t, jobService, mustStart(t, id, err))
	if job.Status != models.JobStatusCompleted {
		t.Fatalf("status = %q, want completed (error: %q)", job.Status, job.Error)
	}

	var md jobCleanupMetadata
	if err := json.Unmarshal([]byte(job.Metadata), &md); err != nil {
		t.Fatalf("failed to parse metadata %q: %v", job.Metadata, err)
	}
	if md.Deleted != 1 || md.RetentionDays != DefaultJobCleanupRetentionDays {
		t.Errorf("metadata = %+v, want deleted=1 retention_days=%d", md, DefaultJobCleanupRetentionDays)
	}

	if _, err := jobService.Get(ctx, running.ID); err != nil {
		t.Errorf("expected the old in-progress job to survive cleanup: %v", err)
	}
	if status, _ := settingsService.Get(ctx, "job_cleanup_last_status"); status != TaskStatusSuccess {
		t.Errorf("job_cleanup_last_status = %q, want success", status)
	}
}

func TestScheduler_EveryTaskRecordsJobs(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)
	for _, task := range scheduler.Tasks() {
		if !task.JobType.Valid() {
			t.Errorf("task %s: JobType %q is not a valid job type", task.Name, task.JobType)
		}
	}
}

// RunNow tests

func TestScheduler_RunNow_UnknownTask(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)

	if _, err := scheduler.RunNow(context.Background(), "does_not_exist"); !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("err = %v, want ErrTaskNotFound", err)
	}
}

func TestScheduler_RunNow_RunsDisabledTaskOutsideWindow(t *testing.T) {
	scheduler, _, jobService, settingsService, _ := setupSchedulerTest(t)
	ctx := context.Background()
	settingsService.Set(ctx, "data_backup_enabled", "false")
	settingsService.Set(ctx, "data_backup_time", time.Now().Add(-2*time.Hour).Format("15:04"))

	id, err := scheduler.RunNow(ctx, "data_backup")
	job := waitForJob(t, jobService, mustStart(t, id, err))
	if job.Status != models.JobStatusCompleted {
		t.Errorf("status = %q, want completed (error: %q)", job.Status, job.Error)
	}
}

func TestScheduler_RunNow_RejectsWhileJobActive(t *testing.T) {
	scheduler, _, jobService, _, _ := setupSchedulerTest(t)
	ctx := context.Background()

	// An in-progress import (e.g. started on startup or by an earlier run).
	active, _ := jobService.Create(ctx, models.JobTypeSetDataImport, "{}")
	jobService.Start(ctx, active.ID)

	if _, err := scheduler.RunNow(ctx, "set_data_update"); !errors.Is(err, ErrTaskRunning) {
		t.Errorf("err = %v, want ErrTaskRunning", err)
	}

	var count int64
	jobService.db.Model(&models.Job{}).Where("type = ?", models.JobTypeSetDataImport).Count(&count)
	if count != 1 {
		t.Errorf("expected no new job to be created, found %d set import jobs", count)
	}
}

func TestScheduler_RunNow_RejectsConcurrentStart(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)
	ctx := context.Background()

	entered := make(chan struct{})
	release := make(chan struct{})
	scheduler.tasks = append(scheduler.tasks, ScheduledTask{
		Name: "slow_start",
		Run: func(ctx context.Context) (uint, error) {
			close(entered)
			<-release
			return 1, nil
		},
	})

	done := make(chan error, 1)
	go func() {
		_, err := scheduler.RunNow(ctx, "slow_start")
		done <- err
	}()
	<-entered

	if _, err := scheduler.RunNow(ctx, "slow_start"); !errors.Is(err, ErrTaskRunning) {
		t.Errorf("err = %v, want ErrTaskRunning", err)
	}

	close(release)
	if err := <-done; err != nil {
		t.Errorf("first run failed: %v", err)
	}
}

func TestScheduler_InBackground_PanicFailsJob(t *testing.T) {
	scheduler, _, jobService, settingsService, _ := setupSchedulerTest(t)
	ctx := context.Background()
	task, _ := scheduler.task("data_backup")

	job, _ := jobService.Create(ctx, models.JobTypeDataBackup, "{}")
	jobService.Start(ctx, job.ID)
	scheduler.inBackground(ctx, task, job.ID, func(context.Context) error { panic("boom") })

	got := waitForJob(t, jobService, job.ID)
	if got.Status != models.JobStatusFailed || !strings.Contains(got.Error, "boom") {
		t.Errorf("job = %q / %q, want failed with the panic message", got.Status, got.Error)
	}
	if status, _ := settingsService.Get(ctx, "data_backup_last_status"); status != TaskStatusFailed {
		t.Errorf("data_backup_last_status = %q, want failed", status)
	}
}

func TestScheduler_TaskExtrasAreValidSettings(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)
	valid := ValidSettingKeys()

	for _, task := range scheduler.Tasks() {
		for _, extra := range task.Extras {
			if !valid[extra.Key] {
				t.Errorf("task %s: extra %s is not a valid setting key", task.Name, extra.Key)
			}
			switch extra.Kind {
			case "int":
				if _, ok := SettingMinimum(extra.Key); !ok {
					t.Errorf("task %s: int extra %s has no minimum", task.Name, extra.Key)
				}
			case "url":
				// *_url keys are what ValidateSetting checks as URLs.
				if !strings.HasSuffix(extra.Key, "_url") {
					t.Errorf("task %s: url extra %s must end in _url", task.Name, extra.Key)
				}
				if err := ValidateSetting(extra.Key, defaultSettings()[extra.Key]); err != nil {
					t.Errorf("task %s: default for %s is invalid: %v", task.Name, extra.Key, err)
				}
			default:
				t.Errorf("task %s: extra %s has unknown kind %q", task.Name, extra.Key, extra.Kind)
			}
		}
	}
}
