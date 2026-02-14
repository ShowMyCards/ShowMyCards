package services

import (
	"backend/models"
	"backend/scryfall"
	"context"
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
	scheduler := NewScheduler(bulkDataService, setDataService, jobService, settingsService)

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

	expectedTasks := []string{"bulk_data_update", "set_data_update", "job_cleanup"}
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
		Name:              "test_task",
		Interval:          24 * time.Hour,
		LastRunSettingKey: "test_task_last_run",
	}

	// Set persisted last run to recent time
	settingsService.SetTime(context.Background(),"test_task_last_run", time.Now())

	// Should check persisted time and return false
	if scheduler.shouldRunTask(context.Background(), task, time.Now()) {
		t.Error("expected task with recent persisted run to not be runnable")
	}
}

// isInTimeWindow tests

func TestScheduler_IsInTimeWindow_EmptyTime(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)

	// Empty time means always in window
	if !scheduler.isInTimeWindow(context.Background(),"", time.Now()) {
		t.Error("expected empty time to always be in window")
	}
}

func TestScheduler_IsInTimeWindow_LiteralTime_InWindow(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)

	now := time.Now()
	timeStr := now.Format("15:04")

	if !scheduler.isInTimeWindow(context.Background(),timeStr, now) {
		t.Error("expected current time to be in window")
	}
}

func TestScheduler_IsInTimeWindow_LiteralTime_OutOfWindow(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)

	now := time.Now()
	// Set target to 2 hours ago
	targetTime := now.Add(-2 * time.Hour)
	timeStr := targetTime.Format("15:04")

	if scheduler.isInTimeWindow(context.Background(),timeStr, now) {
		t.Error("expected past time to be out of window")
	}
}

func TestScheduler_IsInTimeWindow_SettingsKey(t *testing.T) {
	scheduler, _, _, settingsService, _ := setupSchedulerTest(t)

	now := time.Now()
	timeStr := now.Format("15:04")

	// Set time via settings
	settingsService.Set(context.Background(),"test_time_setting", timeStr)

	if !scheduler.isInTimeWindow(context.Background(),"test_time_setting", now) {
		t.Error("expected settings-based time to be in window")
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
		Name:              "test_task",
		Interval:          24 * time.Hour,
		LastRunSettingKey: "test_task_last_run",
	}

	// Set persisted last run to 48 hours ago (well past interval)
	settingsService.SetTime(context.Background(),"test_task_last_run", time.Now().Add(-48*time.Hour))

	if !scheduler.shouldRunTask(context.Background(), task, time.Now()) {
		t.Error("expected task with overdue persisted run to be runnable")
	}
}

func TestScheduler_ShouldRunTask_InMemoryOverridesPersistedRecent(t *testing.T) {
	scheduler, _, _, settingsService, _ := setupSchedulerTest(t)

	task := ScheduledTask{
		Name:              "test_task",
		Interval:          1 * time.Hour,
		LastRunSettingKey: "test_task_last_run",
	}

	// Persisted says it ran 2 hours ago (overdue)
	settingsService.SetTime(context.Background(),"test_task_last_run", time.Now().Add(-2*time.Hour))

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
	scheduler, _, _, _, _ := setupSchedulerTest(t)

	// Create a time at exactly 14:00
	now := time.Date(2024, 6, 15, 14, 3, 0, 0, time.Local)

	// Window is 14:00 to 14:05, so 14:03 is inside
	if !scheduler.isInTimeWindow(context.Background(),"14:00", now) {
		t.Error("expected time 14:03 to be within 5-minute window of 14:00")
	}
}

func TestScheduler_IsInTimeWindow_AtWindowStart(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)

	// Time exactly at the window start
	now := time.Date(2024, 6, 15, 14, 0, 0, 0, time.Local)

	if !scheduler.isInTimeWindow(context.Background(),"14:00", now) {
		t.Error("expected time exactly at window start to be in window")
	}
}

func TestScheduler_IsInTimeWindow_JustOutsideWindow(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)

	// Time at 14:05, window is 14:00 to 14:05 (exclusive end)
	now := time.Date(2024, 6, 15, 14, 5, 0, 0, time.Local)

	if scheduler.isInTimeWindow(context.Background(),"14:00", now) {
		t.Error("expected time at 14:05 to be outside the 5-minute window ending at 14:05")
	}
}

func TestScheduler_IsInTimeWindow_WellOutsideWindow(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)

	// Morning time, target is afternoon
	now := time.Date(2024, 6, 15, 8, 0, 0, 0, time.Local)

	if scheduler.isInTimeWindow(context.Background(),"14:00", now) {
		t.Error("expected morning time to be outside afternoon window")
	}
}

func TestScheduler_IsInTimeWindow_InvalidFormat(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)

	now := time.Now()

	// "invalid" is not 5 chars with ':' at position 2, so it's treated as a settings key
	// Since the settings key doesn't exist, it should return false
	if scheduler.isInTimeWindow(context.Background(),"bad_setting_key", now) {
		t.Error("expected non-existent settings key to be out of window")
	}
}

func TestScheduler_IsInTimeWindow_SettingsKeyOutOfWindow(t *testing.T) {
	scheduler, _, _, settingsService, _ := setupSchedulerTest(t)

	// Set time via settings to a time well in the past relative to now
	now := time.Date(2024, 6, 15, 20, 0, 0, 0, time.Local)
	settingsService.Set(context.Background(),"test_time_setting", "08:00")

	if scheduler.isInTimeWindow(context.Background(),"test_time_setting", now) {
		t.Error("expected settings-based time 08:00 to be out of window when current time is 20:00")
	}
}

// checkTask tests

func TestScheduler_CheckTask_DisabledBySettings(t *testing.T) {
	scheduler, _, _, settingsService, _ := setupSchedulerTest(t)

	ran := false
	task := ScheduledTask{
		Name:              "test_disabled",
		Interval:          1 * time.Millisecond,
		EnabledSettingKey: "test_disabled_setting",
		Run: func(ctx context.Context) {
			ran = true
		},
	}

	// Disable the task via settings
	settingsService.Set(context.Background(),"test_disabled_setting", "false")

	scheduler.checkTask(context.Background(), task, false)

	if ran {
		t.Error("expected disabled task to not run")
	}
}

func TestScheduler_CheckTask_EnabledBySettings(t *testing.T) {
	scheduler, _, _, settingsService, _ := setupSchedulerTest(t)

	ran := false
	task := ScheduledTask{
		Name:              "test_enabled",
		Interval:          1 * time.Millisecond,
		TimeOfDay:         "",
		EnabledSettingKey: "test_enabled_setting",
		Run: func(ctx context.Context) {
			ran = true
		},
	}

	// Enable the task via settings
	settingsService.Set(context.Background(),"test_enabled_setting", "true")

	scheduler.checkTask(context.Background(), task, false)

	if !ran {
		t.Error("expected enabled task to run")
	}
}

func TestScheduler_CheckTask_CatchupSkipsTimeWindow(t *testing.T) {
	scheduler, _, _, settingsService, _ := setupSchedulerTest(t)

	ran := false
	task := ScheduledTask{
		Name:              "test_catchup",
		Interval:          1 * time.Millisecond,
		TimeOfDay:         "03:00", // Specific time that is not now
		EnabledSettingKey: "test_catchup_enabled",
		Run: func(ctx context.Context) {
			ran = true
		},
	}

	settingsService.Set(context.Background(),"test_catchup_enabled", "true")

	// Running as catchup should skip the time window check
	scheduler.checkTask(context.Background(), task, true)

	if !ran {
		t.Error("expected catchup task to run even outside time window")
	}
}

func TestScheduler_CheckTask_PreventsDuplicateExecution(t *testing.T) {
	scheduler, _, _, _, _ := setupSchedulerTest(t)

	var runCount atomic.Int32
	blocker := make(chan struct{})
	task := ScheduledTask{
		Name:     "test_dedup",
		Interval: 1 * time.Millisecond,
		Run: func(ctx context.Context) {
			runCount.Add(1)
			<-blocker // Block until released
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
	settingsService.Set(context.Background(),"bulk_data_auto_update", "false")
	settingsService.Set(context.Background(),"set_data_auto_update", "false")

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
