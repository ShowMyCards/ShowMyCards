package api

import (
	"backend/services"
	"backend/utils"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
)

// ScheduledTaskInfo represents information about a scheduled task
// tygo:export
type ScheduledTaskInfo struct {
	Name          string            `json:"name"` // registry identifier, e.g. "data_backup"
	DisplayName   string            `json:"display_name"`
	Description   string            `json:"description"`
	Enabled       bool              `json:"enabled"`
	EnabledKey    string            `json:"enabled_key"` // settings key to write to change Enabled
	Time          string            `json:"time"`        // time of day, "HH:MM"
	TimeKey       string            `json:"time_key"`    // settings key to write to change Time
	IntervalDays  int               `json:"interval_days"`
	Schedule      string            `json:"schedule"`           // e.g. "03:00 daily", "02:30 every 14 days"
	NextRun       *time.Time        `json:"next_run,omitempty"` // omitted when disabled or the time is invalid
	LastRun       *time.Time        `json:"last_run,omitempty"`
	LastStatus    *string           `json:"last_status,omitempty"` // "in_progress" | "success" | "failed"
	JobType       *string           `json:"job_type,omitempty"`    // type of Job record the task creates
	LastJobID     *uint             `json:"last_job_id,omitempty"`
	LastJobStatus *string           `json:"last_job_status,omitempty"`
	Extras        []TaskSettingInfo `json:"extras"`
}

// TaskSettingInfo is a task-specific setting and its current value
// tygo:export
type TaskSettingInfo struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Kind        string `json:"kind"` // "int" or "url"
	Value       string `json:"value"`
	Min         *int   `json:"min,omitempty"`
}

// RunTaskResponse is returned when a task is started on demand
// tygo:export
type RunTaskResponse struct {
	Message string `json:"message"`
	JobID   uint   `json:"job_id"`
}

// taskRunner exposes the scheduler's registered tasks and lets them be run on demand.
type taskRunner interface {
	Tasks() []services.ScheduledTask
	RunNow(ctx context.Context, name string) (uint, error)
}

// SchedulerHandler handles scheduler-related API requests
type SchedulerHandler struct {
	appCtx          context.Context
	tasks           taskRunner
	settingsService *services.SettingsService
	jobService      *services.JobService
}

// NewSchedulerHandler creates a new scheduler handler. appCtx is used for task
// runs, which outlive the request that started them.
func NewSchedulerHandler(appCtx context.Context, tasks taskRunner, settingsService *services.SettingsService, jobService *services.JobService) *SchedulerHandler {
	return &SchedulerHandler{
		appCtx:          appCtx,
		tasks:           tasks,
		settingsService: settingsService,
		jobService:      jobService,
	}
}

// RunTask starts the task named by the :name path parameter.
func (h *SchedulerHandler) RunTask(c fiber.Ctx) error {
	return h.runTask(c, c.Params("name"))
}

// RunNamedTask returns a handler that starts the given task. It backs the
// legacy per-type import endpoints (e.g. POST /api/bulk-data/import).
func (h *SchedulerHandler) RunNamedTask(name string) fiber.Handler {
	return func(c fiber.Ctx) error {
		return h.runTask(c, name)
	}
}

func (h *SchedulerHandler) runTask(c fiber.Ctx, name string) error {
	jobID, err := h.tasks.RunNow(h.appCtx, name)
	switch {
	case errors.Is(err, services.ErrTaskNotFound):
		return utils.ReturnError(c, fiber.StatusNotFound, fmt.Sprintf("unknown task %q", name))
	case errors.Is(err, services.ErrTaskRunning):
		return utils.ReturnError(c, fiber.StatusConflict, "task is already running")
	case err != nil:
		return utils.LogAndReturnError(c, fiber.StatusInternalServerError,
			"Failed to start task", "task run failed", err)
	}

	return c.Status(fiber.StatusAccepted).JSON(RunTaskResponse{
		Message: "Task started",
		JobID:   jobID,
	})
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
			EnabledKey:   task.EnabledKey(),
			TimeKey:      task.TimeKey(),
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

		info.Extras = make([]TaskSettingInfo, 0, len(task.Extras))
		for _, extra := range task.Extras {
			value, err := h.settingsService.Get(ctx, extra.Key)
			if err != nil {
				slog.Warn("failed to read task setting", "component", "scheduler", "key", extra.Key, "error", err)
			}
			setting := TaskSettingInfo{
				Key:         extra.Key,
				Label:       extra.Label,
				Description: extra.Description,
				Kind:        extra.Kind,
				Value:       value,
			}
			if min, ok := services.SettingMinimum(extra.Key); ok {
				setting.Min = &min
			}
			info.Extras = append(info.Extras, setting)
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
