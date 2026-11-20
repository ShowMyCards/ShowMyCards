package services

import (
	"backend/models"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// DefaultSchedulerCheckInterval is the default interval between scheduler checks
	DefaultSchedulerCheckInterval = 5 * time.Minute

	// DefaultJobCleanupRetentionDays is the default number of days to retain completed jobs
	DefaultJobCleanupRetentionDays = 30
)

// ScheduledTask defines a task that runs on a schedule. Its settings keys are
// derived from Name (see EnabledKey, TimeKey, LastRunKey, LastStatusKey), so
// registering a task is all that is needed to surface it on the jobs screen.
type ScheduledTask struct {
	// Name is the unique identifier for this task and the prefix of its settings keys
	Name string

	// DisplayName is the human-readable name shown in the UI
	DisplayName string

	// Description explains what the task does
	Description string

	// Interval is the minimum time between runs (e.g., 24*time.Hour for daily)
	Interval time.Duration

	// JobType is the type of Job record the task creates, if any
	JobType models.JobType

	// Run is the function to execute when the task should run
	Run func(ctx context.Context)
}

// EnabledKey is the settings key holding whether the task is enabled ("true"/"false").
func (t ScheduledTask) EnabledKey() string { return t.Name + "_enabled" }

// TimeKey is the settings key holding the task's time of day ("HH:MM").
func (t ScheduledTask) TimeKey() string { return t.Name + "_time" }

// LastRunKey is the settings key holding the task's last run time (RFC3339).
func (t ScheduledTask) LastRunKey() string { return t.Name + "_last_run" }

// LastStatusKey is the settings key holding the outcome of the last run.
func (t ScheduledTask) LastStatusKey() string { return t.Name + "_last_status" }

// Values written to a task's LastStatusKey.
const (
	TaskStatusInProgress = "in_progress"
	TaskStatusSuccess    = "success"
	TaskStatusFailed     = "failed"
)

// Scheduler handles scheduled tasks
type Scheduler struct {
	bulkDataService   *BulkDataService
	setDataService    *SetDataService
	symbolDataService *SymbolDataService
	jobService        *JobService
	settingsService   *SettingsService
	backupService     *BackupService
	ticker            *time.Ticker
	done              chan bool
	started           atomic.Bool
	lastRunMu         sync.RWMutex
	lastRun           map[string]time.Time
	runningTasks      sync.Map
	tasks             []ScheduledTask
}

// NewScheduler creates a new scheduler
func NewScheduler(bulkDataService *BulkDataService, setDataService *SetDataService, symbolDataService *SymbolDataService, jobService *JobService, settingsService *SettingsService, backupService *BackupService) *Scheduler {
	s := &Scheduler{
		bulkDataService:   bulkDataService,
		setDataService:    setDataService,
		symbolDataService: symbolDataService,
		jobService:        jobService,
		settingsService:   settingsService,
		backupService:     backupService,
		done:              make(chan bool, 1),
		lastRun:           make(map[string]time.Time),
	}

	// Register all scheduled tasks
	s.tasks = []ScheduledTask{
		{
			Name:        "bulk_data_update",
			DisplayName: "Card Data Update",
			Description: "Downloads Scryfall's bulk card data and refreshes the local card database.",
			Interval:    24 * time.Hour,
			JobType:     models.JobTypeBulkDataImport,
			Run:         s.runBulkDataUpdate,
		},
		{
			Name:        "set_data_update",
			DisplayName: "Set Data Update",
			Description: "Refreshes set information and set icons from Scryfall.",
			Interval:    14 * 24 * time.Hour, // Every 2 weeks
			JobType:     models.JobTypeSetDataImport,
			Run:         s.runSetDataUpdate,
		},
		{
			Name:        "symbol_data_update",
			DisplayName: "Symbol Data Update",
			Description: "Refreshes mana and card symbols from Scryfall.",
			Interval:    30 * 24 * time.Hour, // Monthly — symbols rarely change
			JobType:     models.JobTypeSymbolDataImport,
			Run:         s.runSymbolDataUpdate,
		},
		{
			Name:        "job_cleanup",
			DisplayName: "Job History Cleanup",
			Description: "Deletes job history records older than the retention period.",
			Interval:    24 * time.Hour,
			Run:         s.runJobCleanup,
		},
		{
			Name:        "data_backup",
			DisplayName: "Data Backup",
			Description: "Exports your collection, lists and decks to a compressed backup file.",
			Interval:    24 * time.Hour,
			Run:         s.runBackup,
		},
	}

	return s
}

// Tasks returns the registered scheduled tasks.
func (s *Scheduler) Tasks() []ScheduledTask {
	return s.tasks
}

// Start begins the scheduler loop
func (s *Scheduler) Start(ctx context.Context) {
	s.started.Store(true)

	checkInterval := time.Duration(s.settingsService.GetInt(ctx, "scheduler_check_interval_minutes", int(DefaultSchedulerCheckInterval.Minutes()))) * time.Minute
	s.ticker = time.NewTicker(checkInterval)

	slog.Info("scheduler started", "component", "scheduler", "check_interval", checkInterval)

	// Run initial checks immediately on startup
	go s.checkAndRunTasks(ctx)

	// Run catch-up tasks after a delay (handles missed scheduled windows)
	go s.runCatchupTasks(ctx)

	go func() {
		for {
			select {
			case <-ctx.Done():
				slog.Info("scheduler stopping", "component", "scheduler")
				s.ticker.Stop()
				s.done <- true
				return
			case <-s.ticker.C:
				s.checkAndRunTasks(ctx)
			}
		}
	}()
}

// Stop stops the scheduler gracefully
func (s *Scheduler) Stop() {
	if !s.started.Load() {
		return
	}
	if s.ticker != nil {
		s.ticker.Stop()
	}
	<-s.done
	slog.Info("scheduler stopped", "component", "scheduler")
}

// checkAndRunTasks checks if any scheduled tasks need to run
func (s *Scheduler) checkAndRunTasks(ctx context.Context) {
	for _, task := range s.tasks {
		s.checkTask(ctx, task, false)
	}
}

// checkTask checks if a specific task should run
func (s *Scheduler) checkTask(ctx context.Context, task ScheduledTask, isCatchup bool) {
	// Prevent duplicate execution of the same task
	if _, loaded := s.runningTasks.LoadOrStore(task.Name, true); loaded {
		return
	}
	defer s.runningTasks.Delete(task.Name)

	// Check if task is enabled
	if !s.settingsService.GetBool(ctx, task.EnabledKey(), false) {
		return
	}

	now := time.Now()

	// For scheduled runs (not catchup), check if we're in the time window
	if !isCatchup {
		timeStr, err := s.settingsService.Get(ctx, task.TimeKey())
		if err != nil {
			slog.Warn("failed to get time setting", "component", "scheduler", "setting", task.TimeKey(), "error", err)
			return
		}
		if !isInTimeWindow(timeStr, now) {
			return
		}
	}

	// Check if task should run based on interval
	if !s.shouldRunTask(ctx, task, now) {
		return
	}

	// Set lastRun before execution to prevent re-entry during long-running tasks.
	// The runningTasks sync.Map also guards against concurrent execution, but
	// this ensures shouldRunTask won't pass for the same interval window.
	s.lastRunMu.Lock()
	s.lastRun[task.Name] = now
	s.lastRunMu.Unlock()

	// Log and run
	if isCatchup {
		slog.Info("running catch-up task", "component", "scheduler", "task", task.Name, "interval", task.Interval)
	} else {
		slog.Info("running scheduled task", "component", "scheduler", "task", task.Name, "interval", task.Interval)
	}

	task.Run(ctx)
}

// isInTimeWindow reports whether now falls within the 5-minute window that
// starts at timeStr ("HH:MM").
func isInTimeWindow(timeStr string, now time.Time) bool {
	targetTime, err := time.Parse("15:04", timeStr)
	if err != nil {
		slog.Warn("invalid time format", "component", "scheduler", "time", timeStr, "error", err)
		return false
	}

	// Convert to minutes since midnight for comparison
	currentMinutes := now.Hour()*60 + now.Minute()
	targetMinutes := targetTime.Hour()*60 + targetTime.Minute()

	return currentMinutes >= targetMinutes && currentMinutes < targetMinutes+scheduleWindowMinutes
}

// scheduleWindowMinutes is how long after a task's time of day it may still start.
const scheduleWindowMinutes = 5

// NextRun returns the next time a task will run: the first occurrence of
// timeStr ("HH:MM") after now at which at least interval has elapsed since
// lastRun. It mirrors the checks in checkTask (time window + shouldRunTask).
func NextRun(timeStr string, interval time.Duration, lastRun *time.Time, now time.Time) (time.Time, error) {
	target, err := time.Parse("15:04", timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing time of day %q: %w", timeStr, err)
	}

	next := time.Date(now.Year(), now.Month(), now.Day(), target.Hour(), target.Minute(), 0, 0, now.Location())
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}

	if lastRun != nil {
		// Bounded: an interval can never push the next run out more than
		// interval/24h + 1 days.
		for next.Sub(*lastRun) < interval {
			next = next.AddDate(0, 0, 1)
		}
	}

	return next, nil
}

// shouldRunTask checks if a task should run based on its interval and last run time
func (s *Scheduler) shouldRunTask(ctx context.Context, task ScheduledTask, now time.Time) bool {
	s.lastRunMu.Lock()
	defer s.lastRunMu.Unlock()

	// Check in-memory last run
	if lastRun, exists := s.lastRun[task.Name]; exists {
		if time.Since(lastRun) < task.Interval {
			return false
		}
	}

	// Check persisted last run time
	lastUpdate, err := s.settingsService.GetTime(ctx, task.LastRunKey())
	if err == nil && lastUpdate != nil {
		if time.Since(*lastUpdate) < task.Interval {
			// Update in-memory cache
			s.lastRun[task.Name] = *lastUpdate
			return false
		}
	}

	return true
}

// runCatchupTasks checks for overdue tasks after startup
func (s *Scheduler) runCatchupTasks(ctx context.Context) {
	if !s.settingsService.GetBool(ctx, "scheduler_catchup_enabled", true) {
		slog.Info("scheduler catch-up disabled", "component", "scheduler")
		return
	}

	delaySeconds := s.settingsService.GetInt(ctx, "scheduler_catchup_delay_seconds", 60)
	delay := time.Duration(delaySeconds) * time.Second

	slog.Info("scheduler catch-up will check for overdue tasks", "component", "scheduler", "delay", delay)

	select {
	case <-ctx.Done():
		return
	case <-time.After(delay):
	}

	slog.Info("checking for overdue scheduled tasks", "component", "scheduler")

	for _, task := range s.tasks {
		s.checkTask(ctx, task, true)
	}
}

// Task execution functions

func (s *Scheduler) runBulkDataUpdate(ctx context.Context) {
	job, err := s.bulkDataService.CreateImportJob(ctx)
	if err != nil {
		slog.Error("error creating bulk data import job", "component", "scheduler", "error", err)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in bulk data import", "component", "scheduler", "panic", r)
			}
		}()
		if err := s.bulkDataService.DownloadAndImport(ctx, job.ID); err != nil {
			slog.Error("error in bulk data import", "component", "scheduler", "error", err)
		}
	}()
}

func (s *Scheduler) runSetDataUpdate(ctx context.Context) {
	job, err := s.setDataService.CreateImportJob(ctx)
	if err != nil {
		slog.Error("error creating set data import job", "component", "scheduler", "error", err)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in set data import", "component", "scheduler", "panic", r)
			}
		}()
		if err := s.setDataService.DownloadAndImport(ctx, job.ID); err != nil {
			slog.Error("error in set data import", "component", "scheduler", "error", err)
		}
	}()
}

func (s *Scheduler) runSymbolDataUpdate(ctx context.Context) {
	job, err := s.symbolDataService.CreateImportJob(ctx)
	if err != nil {
		slog.Error("error creating symbol data import job", "component", "scheduler", "error", err)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in symbol data import", "component", "scheduler", "panic", r)
			}
		}()
		if err := s.symbolDataService.DownloadAndImport(ctx, job.ID); err != nil {
			slog.Error("error in symbol data import", "component", "scheduler", "error", err)
		}
	}()
}

func (s *Scheduler) runJobCleanup(ctx context.Context) {
	task, _ := s.task("job_cleanup")
	retentionDays := s.settingsService.GetInt(ctx, "job_cleanup_retention_days", DefaultJobCleanupRetentionDays)
	deletedCount, err := s.jobService.CleanupOldJobs(ctx, retentionDays)
	if err != nil {
		slog.Error("error cleaning up jobs", "component", "scheduler", "error", err)
		s.setStatus(ctx, task, TaskStatusFailed)
		return
	}

	s.recordRun(ctx, task, TaskStatusSuccess)
	slog.Info("cleaned up old jobs", "component", "scheduler", "deleted_count", deletedCount)
}

func (s *Scheduler) runBackup(ctx context.Context) {
	if s.backupService == nil {
		return
	}
	task, _ := s.task("data_backup")

	path, err := s.backupService.CreateBackup(ctx)
	if err != nil {
		slog.Error("error creating data backup", "component", "scheduler", "error", err)
		// last_run is deliberately not updated so catch-up retries the backup.
		s.setStatus(ctx, task, TaskStatusFailed)
		return
	}

	retention := s.settingsService.GetInt(ctx, "data_backup_retention_count", DefaultBackupRetention)
	pruned, err := s.backupService.PruneBackups(retention)
	if err != nil {
		// The backup itself succeeded; a prune failure is non-fatal.
		slog.Warn("failed to prune old backups", "component", "scheduler", "error", err)
	}

	s.recordRun(ctx, task, TaskStatusSuccess)
	slog.Info("created data backup", "component", "scheduler", "path", path, "pruned", pruned, "retention", retention)
}

// task returns the registered task with the given name.
func (s *Scheduler) task(name string) (ScheduledTask, bool) {
	for _, t := range s.tasks {
		if t.Name == name {
			return t, true
		}
	}
	return ScheduledTask{}, false
}

// recordRun persists a task's last run time (now) and status.
func (s *Scheduler) recordRun(ctx context.Context, task ScheduledTask, status string) {
	if err := s.settingsService.SetTime(ctx, task.LastRunKey(), time.Now()); err != nil {
		slog.Warn("failed to persist task last run", "component", "scheduler", "key", task.LastRunKey(), "error", err)
	}
	s.setStatus(ctx, task, status)
}

// setStatus persists a task's last run status.
func (s *Scheduler) setStatus(ctx context.Context, task ScheduledTask, status string) {
	if err := s.settingsService.Set(ctx, task.LastStatusKey(), status); err != nil {
		slog.Warn("failed to persist task status", "component", "scheduler", "key", task.LastStatusKey(), "error", err)
	}
}
