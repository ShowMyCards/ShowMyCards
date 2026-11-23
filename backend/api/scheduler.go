package api

import (
	"backend/services"
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
)

// ScheduledTaskInfo represents information about a scheduled task
// tygo:export
type ScheduledTaskInfo struct {
	Name          string     `json:"name"` // registry identifier, e.g. "data_backup"
	DisplayName   string     `json:"display_name"`
	Description   string     `json:"description"`
	Enabled       bool       `json:"enabled"`
	Time          string     `json:"time"` // time of day, "HH:MM"
	IntervalDays  int        `json:"interval_days"`
	Schedule      string     `json:"schedule"`           // e.g. "03:00 daily", "02:30 every 14 days"
	NextRun       *time.Time `json:"next_run,omitempty"` // omitted when disabled or the time is invalid
	LastRun       *time.Time `json:"last_run,omitempty"`
	LastStatus    *string    `json:"last_status,omitempty"` // "in_progress" | "success" | "failed"
	JobType       *string    `json:"job_type,omitempty"`    // type of Job record the task creates
	LastJobID     *uint      `json:"last_job_id,omitempty"`
	LastJobStatus *string    `json:"last_job_status,omitempty"`
}

// taskProvider exposes the scheduler's registered tasks.
type taskProvider interface {
	Tasks() []services.ScheduledTask
}

// SchedulerHandler handles scheduler-related API requests
type SchedulerHandler struct {
	tasks           taskProvider
	settingsService *services.SettingsService
	jobService      *services.JobService
}

// NewSchedulerHandler creates a new scheduler handler
func NewSchedulerHandler(tasks taskProvider, settingsService *services.SettingsService, jobService *services.JobService) *SchedulerHandler {
	return &SchedulerHandler{
		tasks:           tasks,
		settingsService: settingsService,
		jobService:      jobService,
	}
}

// GetScheduledTasks returns information about every registered scheduled task
func (h *SchedulerHandler) GetScheduledTasks(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	now := time.Now()
	registered := h.tasks.Tasks()
	tasks := make([]ScheduledTaskInfo, 0, len(registered))

	for _, task := range registered {
		info := ScheduledTaskInfo{
			Name:         task.Name,
			DisplayName:  task.DisplayName,
			Description:  task.Description,
			Enabled:      h.settingsService.GetBool(ctx, task.EnabledKey(), false),
			IntervalDays: int(task.Interval / (24 * time.Hour)),
		}

		if timeStr, err := h.settingsService.Get(ctx, task.TimeKey()); err == nil {
			info.Time = timeStr
		}
		info.Schedule = describeSchedule(info.Time, info.IntervalDays)

		lastRun, err := h.settingsService.GetTime(ctx, task.LastRunKey())
		if err != nil {
			slog.Warn("failed to read task last run", "component", "scheduler", "task", task.Name, "error", err)
		}
		info.LastRun = lastRun

		if status, err := h.settingsService.Get(ctx, task.LastStatusKey()); err == nil && status != "" {
			info.LastStatus = &status
		}

		if info.Enabled {
			next, err := services.NextRun(info.Time, task.Interval, lastRun, now)
			if err != nil {
				slog.Warn("failed to calculate next run", "component", "scheduler", "task", task.Name, "error", err)
			} else {
				info.NextRun = &next
			}
		}

		if task.JobType != "" {
			jobType := string(task.JobType)
			info.JobType = &jobType

			lastJob, err := h.jobService.GetLastJobByType(ctx, task.JobType)
			if err != nil {
				slog.Warn("failed to get last job", "component", "scheduler", "task", task.Name, "error", err)
			}
			if lastJob != nil {
				info.LastJobID = &lastJob.ID
				status := string(lastJob.Status)
				info.LastJobStatus = &status
			}
		}

		tasks = append(tasks, info)
	}

	return c.JSON(tasks)
}

// describeSchedule renders a human-readable schedule, e.g. "03:00 daily".
func describeSchedule(timeStr string, intervalDays int) string {
	switch intervalDays {
	case 0, 1:
		return timeStr + " daily"
	default:
		return fmt.Sprintf("%s every %d days", timeStr, intervalDays)
	}
}
