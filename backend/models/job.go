package models

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// JobType represents the type of job
// tygo:export
type JobType string

const (
	JobTypeBulkDataImport JobType = "bulk_data_import"
	JobTypeSetDataImport  JobType = "set_data_import"
)

// Valid checks if the job type is valid
func (jt JobType) Valid() bool {
	switch jt {
	case JobTypeBulkDataImport, JobTypeSetDataImport:
		return true
	default:
		return false
	}
}

// Scan implements the sql.Scanner interface
func (jt *JobType) Scan(value interface{}) error {
	if value == nil {
		return errors.New("job type cannot be null")
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("job type must be a string, got %T", value)
	}

	*jt = JobType(str)
	return nil
}

// Value implements the driver.Valuer interface
func (jt JobType) Value() (driver.Value, error) {
	if !jt.Valid() {
		return nil, fmt.Errorf("invalid job type: %s", jt)
	}
	return string(jt), nil
}

// JobStatus represents the status of a job
// tygo:export
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusInProgress JobStatus = "in_progress"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCancelled  JobStatus = "cancelled"
)

// Valid checks if the job status is valid
func (js JobStatus) Valid() bool {
	switch js {
	case JobStatusPending, JobStatusInProgress, JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
		return true
	default:
		return false
	}
}

// Scan implements the sql.Scanner interface
func (js *JobStatus) Scan(value interface{}) error {
	if value == nil {
		return errors.New("job status cannot be null")
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("job status must be a string, got %T", value)
	}

	*js = JobStatus(str)
	return nil
}

// Value implements the driver.Valuer interface
func (js JobStatus) Value() (driver.Value, error) {
	if !js.Valid() {
		return nil, fmt.Errorf("invalid job status: %s", js)
	}
	return string(js), nil
}

// Job represents a long-running background job
// tygo:export
type Job struct {
	BaseModel
	Type        JobType    `gorm:"type:varchar(50);not null" json:"type"`
	Status      JobStatus  `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string     `gorm:"type:text" json:"error,omitempty"`
	Metadata    string     `gorm:"type:text" json:"metadata,omitempty"` // JSON stored as string
}

// BeforeCreate validates the job before creation
func (j *Job) BeforeCreate(tx *gorm.DB) error {
	if !j.Type.Valid() {
		return fmt.Errorf("invalid job type: %s", j.Type)
	}
	if !j.Status.Valid() {
		return fmt.Errorf("invalid job status: %s", j.Status)
	}
	// Ensure status is pending for new jobs
	if j.Status == "" {
		j.Status = JobStatusPending
	}
	return nil
}

// BeforeUpdate validates the job before update
func (j *Job) BeforeUpdate(tx *gorm.DB) error {
	// Skip validation for partial updates (when using Updates with map)
	// Validation is enforced at creation time
	return nil
}
