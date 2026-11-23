package services

import (
	"backend/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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

	// Extras are task-specific settings shown alongside the schedule
	Extras []TaskSetting

	// Run starts the task: it creates the task's Job record synchronously,
	// continues the work in the background, and returns the job's ID.
	Run func(ctx context.Context) (uint, error)
}

// TaskSetting describes a task-specific setting the UI can edit.
type TaskSetting struct {
	Key         string
	Label       string
	Description string
	Kind        string // "int" or "url"
}

// EnabledKey is the settings key holding whether the task is enabled ("true"/"false").
func (t ScheduledTask) EnabledKey() string { return t.Name + "_enabled" }

// TimeKey is the settings key holding the task's time of day ("HH:MM").
func (t ScheduledTask) TimeKey() string { return t.Name + "_time" }

// LastRunKey is the settings key holding the task's last run time (RFC3339).
func (t ScheduledTask) LastRunKey() string { return t.Name + "_last_run" }

// LastStatusKey is the settings key holding the outcome of the last run.
func (t ScheduledTask) LastStatusKey() string { return t.Name + "_last_status" }

// Errors returned by Scheduler.RunNow.
var (
	ErrTaskNotFound = errors.New("scheduled task not found")
	ErrTaskRunning  = errors.New("scheduled task is already running")
)

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
			Extras: []TaskSetting{{
				Key:         "bulk_data_url",
				Label:       "Bulk data source",
				Description: "Scryfall bulk data API endpoint the update downloads from.",
				Kind:        "url",
			}},
			Run: s.runBulkDataUpdate,
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
			JobType:     models.JobTypeJobCleanup,
			Extras: []TaskSetting{{
				Key:         "job_cleanup_retention_days",
				Label:       "Keep history for (days)",
				Description: "Finished jobs older than this are deleted.",
				Kind:        "int",
			}},
			Run: s.runJobCleanup,
		},
		{
			Name:        "data_backup",
			DisplayName: "Data Backup",
			Description: "Exports your collection, lists and decks to a compressed backup file.",
			Interval:    24 * time.Hour,
			JobType:     models.JobTypeDataBackup,
			Extras: []TaskSetting{{
				Key:         "data_backup_retention_count",
				Label:       "Backups to keep",
				Description: "Older backups are deleted after each run. 0 keeps every backup.",
				Kind:        "int",
			}},
			Run: s.runBackup,
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

// RunNow starts the named task immediately, regardless of whether it is
// enabled or inside its time window, and returns the ID of the Job it created.
// It returns ErrTaskNotFound for an unknown name and ErrTaskRunning if the
// task is already running. ctx must outlive the request (use the app context),
// since the task keeps running in the background.
func (s *Scheduler) RunNow(ctx context.Context, name string) (uint, error) {
	task, ok := s.task(name)
	if !ok {
		return 0, ErrTaskNotFound
	}
	slog.Info("running task on demand", "component", "scheduler", "task", task.Name)
	return s.start(ctx, task)
}

// start runs task unless it is already running. The runningTasks guard makes
// the active-job check and job creation atomic; the active-job check covers
// the background phase after Run has returned (and imports started outside the
// scheduler, such as the initial import on startup).
func (s *Scheduler) start(ctx context.Context, task ScheduledTask) (uint, error) {
	if _, loaded := s.runningTasks.LoadOrStore(task.Name, true); loaded {
		return 0, ErrTaskRunning
	}
	defer s.runningTasks.Delete(task.Name)

	if task.JobType != "" {
		active, err := s.jobService.HasActive(ctx, task.JobType)
		if err != nil {
			return 0, fmt.Errorf("checking for running %s job: %w", task.JobType, err)
		}
		if active {
			return 0, ErrTaskRunning
		}
	}

	return task.Run(ctx)
}

// checkTask checks if a specific task should run
func (s *Scheduler) checkTask(ctx context.Context, task ScheduledTask, isCatchup bool) {
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

	if _, err := s.start(ctx, task); err != nil {
		if errors.Is(err, ErrTaskRunning) {
			slog.Info("skipping task, already running", "component", "scheduler", "task", task.Name)
			return
		}
		slog.Error("error starting task", "component", "scheduler", "task", task.Name, "error", err)
	}
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

func (s *Scheduler) runBulkDataUpdate(ctx context.Context) (uint, error) {
	task, _ := s.task("bulk_data_update")
	job, err := s.bulkDataService.CreateImportJob(ctx)
	if err != nil {
		return 0, fmt.Errorf("creating bulk data import job: %w", err)
	}
	s.inBackground(ctx, task, job.ID, func(ctx context.Context) error {
		return s.bulkDataService.DownloadAndImport(ctx, job.ID)
	})
	return job.ID, nil
}

func (s *Scheduler) runSetDataUpdate(ctx context.Context) (uint, error) {
	task, _ := s.task("set_data_update")
	job, err := s.setDataService.CreateImportJob(ctx)
	if err != nil {
		return 0, fmt.Errorf("creating set data import job: %w", err)
	}
	s.inBackground(ctx, task, job.ID, func(ctx context.Context) error {
		return s.setDataService.DownloadAndImport(ctx, job.ID)
	})
	return job.ID, nil
}

func (s *Scheduler) runSymbolDataUpdate(ctx context.Context) (uint, error) {
	task, _ := s.task("symbol_data_update")
	job, err := s.symbolDataService.CreateImportJob(ctx)
	if err != nil {
		return 0, fmt.Errorf("creating symbol data import job: %w", err)
	}
	s.inBackground(ctx, task, job.ID, func(ctx context.Context) error {
		return s.symbolDataService.DownloadAndImport(ctx, job.ID)
	})
	return job.ID, nil
}

// inBackground runs fn in a goroutine. fn is responsible for recording its own
// outcome on the job; a panic is recovered and recorded as a failure so the job
// never stays in progress (which would block the task until restart).
func (s *Scheduler) inBackground(ctx context.Context, task ScheduledTask, jobID uint, fn func(ctx context.Context) error) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in scheduled task", "component", "scheduler", "task", task.Name, "job_id", jobID, "panic", r)
				s.setStatus(ctx, task, TaskStatusFailed)
				if err := s.jobService.Fail(ctx, jobID, fmt.Sprintf("panic: %v", r)); err != nil {
					slog.Warn("failed to mark job failed", "component", "scheduler", "job_id", jobID, "error", err)
				}
			}
		}()
		if err := fn(ctx); err != nil {
			slog.Error("scheduled task failed", "component", "scheduler", "task", task.Name, "job_id", jobID, "error", err)
		}
	}()
}

// jobCleanupMetadata is the Job metadata recorded by the job_cleanup task.
type jobCleanupMetadata struct {
	Deleted       int64 `json:"deleted"`
	RetentionDays int   `json:"retention_days"`
}

func (s *Scheduler) runJobCleanup(ctx context.Context) (uint, error) {
	task, _ := s.task("job_cleanup")
	return s.runAsJob(ctx, task, func(ctx context.Context) (any, error) {
		retentionDays := s.settingsService.GetInt(ctx, "job_cleanup_retention_days", DefaultJobCleanupRetentionDays)
		deleted, err := s.jobService.CleanupOldJobs(ctx, retentionDays)
		if err != nil {
			return nil, err
		}
		slog.Info("cleaned up old jobs", "component", "scheduler", "deleted_count", deleted)
		return jobCleanupMetadata{Deleted: deleted, RetentionDays: retentionDays}, nil
	})
}

// backupMetadata is the Job metadata recorded by the data_backup task.
type backupMetadata struct {
	FileName   string `json:"file_name"`
	SizeBytes  int64  `json:"size_bytes"`
	Retention  int    `json:"retention"`
	Pruned     int    `json:"pruned"`
	PruneError string `json:"prune_error,omitempty"`
}

func (s *Scheduler) runBackup(ctx context.Context) (uint, error) {
	if s.backupService == nil {
		return 0, errors.New("backups are not configured")
	}
	task, _ := s.task("data_backup")
	return s.runAsJob(ctx, task, func(ctx context.Context) (any, error) {
		path, err := s.backupService.CreateBackup(ctx)
		if err != nil {
			return nil, err
		}

		md := backupMetadata{FileName: filepath.Base(path)}
		if info, err := os.Stat(path); err == nil {
			md.SizeBytes = info.Size()
		} else {
			slog.Warn("failed to stat backup file", "component", "scheduler", "path", path, "error", err)
		}

		md.Retention = s.settingsService.GetInt(ctx, "data_backup_retention_count", DefaultBackupRetention)
		md.Pruned, err = s.backupService.PruneBackups(md.Retention)
		if err != nil {
			// The backup itself succeeded; a prune failure is non-fatal.
			slog.Warn("failed to prune old backups", "component", "scheduler", "error", err)
			md.PruneError = err.Error()
		}

		slog.Info("created data backup", "component", "scheduler", "path", path, "pruned", md.Pruned, "retention", md.Retention)
		return md, nil
	})
}

// runAsJob creates a Job record of task.JobType and runs fn in the background,
// storing fn's metadata on the job and mirroring the outcome into the task's
// settings. On failure last_run is deliberately left unchanged so catch-up
// retries.
func (s *Scheduler) runAsJob(ctx context.Context, task ScheduledTask, fn func(ctx context.Context) (any, error)) (uint, error) {
	job, err := s.jobService.Create(ctx, task.JobType, "{}")
	if err != nil {
		s.setStatus(ctx, task, TaskStatusFailed)
		return 0, fmt.Errorf("creating %s job: %w", task.JobType, err)
	}
	if err := s.jobService.Start(ctx, job.ID); err != nil {
		slog.Warn("failed to mark job started", "component", "scheduler", "task", task.Name, "job_id", job.ID, "error", err)
	}
	s.setStatus(ctx, task, TaskStatusInProgress)

	s.inBackground(ctx, task, job.ID, func(ctx context.Context) error {
		metadata, runErr := fn(ctx)

		if metadata != nil {
			if b, err := json.Marshal(metadata); err != nil {
				slog.Warn("failed to marshal job metadata", "component", "scheduler", "task", task.Name, "error", err)
			} else if err := s.jobService.UpdateMetadata(ctx, job.ID, string(b)); err != nil {
				slog.Warn("failed to store job metadata", "component", "scheduler", "task", task.Name, "job_id", job.ID, "error", err)
			}
		}

		// Task settings are written before the job is finalised, so anyone who
		// observes a finished job also sees the task's updated status.
		if runErr != nil {
			s.setStatus(ctx, task, TaskStatusFailed)
			if err := s.jobService.Fail(ctx, job.ID, runErr.Error()); err != nil {
				slog.Warn("failed to mark job failed", "component", "scheduler", "job_id", job.ID, "error", err)
			}
			return runErr
		}

		s.recordRun(ctx, task, TaskStatusSuccess)
		if err := s.jobService.Complete(ctx, job.ID); err != nil {
			slog.Warn("failed to mark job completed", "component", "scheduler", "job_id", job.ID, "error", err)
		}
		return nil
	})

	return job.ID, nil
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
