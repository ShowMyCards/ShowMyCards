package services

import (
	"context"
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

// ScheduledTask defines a task that runs on a schedule
type ScheduledTask struct {
	// Name is the unique identifier for this task
	Name string

	// Interval is how often the task should run (e.g., 24*time.Hour for daily)
	Interval time.Duration

	// TimeOfDay is the preferred time to run (format: "HH:MM", e.g., "03:00")
	// If empty, runs whenever interval has elapsed
	TimeOfDay string

	// EnabledSettingKey is the settings key to check if task is enabled (optional)
	// If empty, task is always enabled
	EnabledSettingKey string

	// LastRunSettingKey is the settings key where last run time is persisted
	LastRunSettingKey string

	// Run is the function to execute when the task should run
	Run func(ctx context.Context)
}

// Scheduler handles scheduled tasks
type Scheduler struct {
	bulkDataService *BulkDataService
	setDataService  *SetDataService
	jobService      *JobService
	settingsService *SettingsService
	ticker          *time.Ticker
	done            chan bool
	started         atomic.Bool
	lastRunMu       sync.RWMutex
	lastRun         map[string]time.Time
	runningTasks    sync.Map
	tasks           []ScheduledTask
}

// NewScheduler creates a new scheduler
func NewScheduler(bulkDataService *BulkDataService, setDataService *SetDataService, jobService *JobService, settingsService *SettingsService) *Scheduler {
	s := &Scheduler{
		bulkDataService: bulkDataService,
		setDataService:  setDataService,
		jobService:      jobService,
		settingsService: settingsService,
		done:            make(chan bool, 1),
		lastRun:         make(map[string]time.Time),
	}

	// Register all scheduled tasks
	s.tasks = []ScheduledTask{
		{
			Name:              "bulk_data_update",
			Interval:          24 * time.Hour,
			TimeOfDay:         "bulk_data_update_time",
			EnabledSettingKey: "bulk_data_auto_update",
			LastRunSettingKey: "bulk_data_last_update",
			Run:               s.runBulkDataUpdate,
		},
		{
			Name:              "set_data_update",
			Interval:          14 * 24 * time.Hour, // Every 2 weeks
			TimeOfDay:         "set_data_update_time",
			EnabledSettingKey: "set_data_auto_update",
			LastRunSettingKey: "set_data_last_update",
			Run:               s.runSetDataUpdate,
		},
		{
			Name:              "job_cleanup",
			Interval:          24 * time.Hour,
			TimeOfDay:         "00:00", // Midnight
			LastRunSettingKey: "job_cleanup_last_run",
			Run:               s.runJobCleanup,
		},
	}

	return s
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
	if task.EnabledSettingKey != "" {
		if !s.settingsService.GetBool(ctx, task.EnabledSettingKey, false) {
			return
		}
	}

	now := time.Now()

	// For scheduled runs (not catchup), check if we're in the time window
	if !isCatchup {
		if !s.isInTimeWindow(ctx, task.TimeOfDay, now) {
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

// isInTimeWindow checks if we're within 5 minutes of the configured time
func (s *Scheduler) isInTimeWindow(ctx context.Context, timeOfDaySetting string, now time.Time) bool {
	// Get the time string - either from settings or use as literal
	var timeStr string
	if timeOfDaySetting == "" {
		return true // No specific time required
	}

	// Check if it's a settings key or a literal time
	if len(timeOfDaySetting) == 5 && timeOfDaySetting[2] == ':' {
		// Looks like a literal time (e.g., "00:00")
		timeStr = timeOfDaySetting
	} else {
		// It's a settings key
		var err error
		timeStr, err = s.settingsService.Get(ctx, timeOfDaySetting)
		if err != nil {
			slog.Warn("failed to get time setting", "component", "scheduler", "setting", timeOfDaySetting, "error", err)
			return false
		}
	}

	// Parse the time
	targetTime, err := time.Parse("15:04", timeStr)
	if err != nil {
		slog.Warn("invalid time format", "component", "scheduler", "setting", timeOfDaySetting, "error", err)
		return false
	}

	// Convert to minutes since midnight for comparison
	currentMinutes := now.Hour()*60 + now.Minute()
	targetMinutes := targetTime.Hour()*60 + targetTime.Minute()

	// Check if we're within the 5-minute window
	return currentMinutes >= targetMinutes && currentMinutes < targetMinutes+5
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
	if task.LastRunSettingKey != "" {
		lastUpdate, err := s.settingsService.GetTime(ctx, task.LastRunSettingKey)
		if err == nil && lastUpdate != nil {
			if time.Since(*lastUpdate) < task.Interval {
				// Update in-memory cache
				s.lastRun[task.Name] = *lastUpdate
				return false
			}
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

func (s *Scheduler) runJobCleanup(ctx context.Context) {
	retentionDays := s.settingsService.GetInt(ctx, "job_cleanup_retention_days", DefaultJobCleanupRetentionDays)
	deletedCount, err := s.jobService.CleanupOldJobs(ctx, retentionDays)
	if err != nil {
		slog.Error("error cleaning up jobs", "component", "scheduler", "error", err)
		return
	}

	// Persist completion time
	if err := s.settingsService.SetTime(ctx, "job_cleanup_last_run", time.Now()); err != nil {
		slog.Warn("failed to persist job_cleanup_last_run", "component", "scheduler", "error", err)
	}

	slog.Info("cleaned up old jobs", "component", "scheduler", "deleted_count", deletedCount)
}
