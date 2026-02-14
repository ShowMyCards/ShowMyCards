package models

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupJobTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	if err := db.AutoMigrate(&Job{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return db
}

func TestJobType_Valid(t *testing.T) {
	tests := []struct {
		name     string
		jobType  JobType
		expected bool
	}{
		{"BulkDataImport", JobTypeBulkDataImport, true},
		{"SetDataImport", JobTypeSetDataImport, true},
		{"Empty", JobType(""), false},
		{"InvalidType", JobType("invalid_type"), false},
		{"CaseSensitive", JobType("Bulk_Data_Import"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.jobType.Valid()
			if result != tt.expected {
				t.Errorf("Valid() = %v, expected %v for JobType %q", result, tt.expected, tt.jobType)
			}
		})
	}
}

func TestJobStatus_Valid(t *testing.T) {
	tests := []struct {
		name     string
		status   JobStatus
		expected bool
	}{
		{"Pending", JobStatusPending, true},
		{"InProgress", JobStatusInProgress, true},
		{"Completed", JobStatusCompleted, true},
		{"Failed", JobStatusFailed, true},
		{"Cancelled", JobStatusCancelled, true},
		{"Empty", JobStatus(""), false},
		{"InvalidStatus", JobStatus("running"), false},
		{"CaseSensitive", JobStatus("Pending"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.Valid()
			if result != tt.expected {
				t.Errorf("Valid() = %v, expected %v for JobStatus %q", result, tt.expected, tt.status)
			}
		})
	}
}

func TestJob_BeforeCreate_ValidatesJobType(t *testing.T) {
	db := setupJobTestDB(t)

	tests := []struct {
		name        string
		job         *Job
		expectError bool
	}{
		{
			name: "Valid BulkDataImport Type",
			job: &Job{
				Type:   JobTypeBulkDataImport,
				Status: JobStatusPending,
			},
			expectError: false,
		},
		{
			name: "Valid SetDataImport Type",
			job: &Job{
				Type:   JobTypeSetDataImport,
				Status: JobStatusPending,
			},
			expectError: false,
		},
		{
			name: "Invalid Job Type",
			job: &Job{
				Type:   JobType("nonexistent"),
				Status: JobStatusPending,
			},
			expectError: true,
		},
		{
			name: "Empty Job Type",
			job: &Job{
				Type:   JobType(""),
				Status: JobStatusPending,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Create(tt.job).Error
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestJob_BeforeCreate_ValidatesJobStatus(t *testing.T) {
	db := setupJobTestDB(t)

	tests := []struct {
		name        string
		job         *Job
		expectError bool
	}{
		{
			name: "Valid Pending Status",
			job: &Job{
				Type:   JobTypeBulkDataImport,
				Status: JobStatusPending,
			},
			expectError: false,
		},
		{
			name: "Valid InProgress Status",
			job: &Job{
				Type:   JobTypeBulkDataImport,
				Status: JobStatusInProgress,
			},
			expectError: false,
		},
		{
			name: "Valid Completed Status",
			job: &Job{
				Type:   JobTypeBulkDataImport,
				Status: JobStatusCompleted,
			},
			expectError: false,
		},
		{
			name: "Valid Failed Status",
			job: &Job{
				Type:   JobTypeBulkDataImport,
				Status: JobStatusFailed,
			},
			expectError: false,
		},
		{
			name: "Valid Cancelled Status",
			job: &Job{
				Type:   JobTypeBulkDataImport,
				Status: JobStatusCancelled,
			},
			expectError: false,
		},
		{
			name: "Invalid Status",
			job: &Job{
				Type:   JobTypeBulkDataImport,
				Status: JobStatus("running"),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Create(tt.job).Error
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestJob_CreateSuccess(t *testing.T) {
	db := setupJobTestDB(t)

	job := &Job{
		Type:   JobTypeBulkDataImport,
		Status: JobStatusPending,
	}

	if err := db.Create(job).Error; err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	if job.ID == 0 {
		t.Error("expected ID to be set after create")
	}
	if job.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set after create")
	}

	// Verify retrieval
	var retrieved Job
	if err := db.First(&retrieved, job.ID).Error; err != nil {
		t.Fatalf("failed to retrieve job: %v", err)
	}

	if retrieved.Type != JobTypeBulkDataImport {
		t.Errorf("expected type %q, got %q", JobTypeBulkDataImport, retrieved.Type)
	}
	if retrieved.Status != JobStatusPending {
		t.Errorf("expected status %q, got %q", JobStatusPending, retrieved.Status)
	}
}

func TestJobType_Value_InvalidReturnsError(t *testing.T) {
	jt := JobType("invalid")
	_, err := jt.Value()
	if err == nil {
		t.Error("expected error for invalid job type Value(), got nil")
	}
}

func TestJobType_Value_ValidReturnsString(t *testing.T) {
	jt := JobTypeBulkDataImport
	val, err := jt.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != string(JobTypeBulkDataImport) {
		t.Errorf("expected %q, got %q", string(JobTypeBulkDataImport), val)
	}
}

func TestJobStatus_Value_InvalidReturnsError(t *testing.T) {
	js := JobStatus("invalid")
	_, err := js.Value()
	if err == nil {
		t.Error("expected error for invalid job status Value(), got nil")
	}
}

func TestJobStatus_Value_ValidReturnsString(t *testing.T) {
	js := JobStatusCompleted
	val, err := js.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != string(JobStatusCompleted) {
		t.Errorf("expected %q, got %q", string(JobStatusCompleted), val)
	}
}

func TestJobType_Scan_ValidString(t *testing.T) {
	var jt JobType
	err := jt.Scan("bulk_data_import")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jt != JobTypeBulkDataImport {
		t.Errorf("expected %q, got %q", JobTypeBulkDataImport, jt)
	}
}

func TestJobType_Scan_Nil(t *testing.T) {
	var jt JobType
	err := jt.Scan(nil)
	if err == nil {
		t.Error("expected error for nil value")
	}
}

func TestJobType_Scan_NonString(t *testing.T) {
	var jt JobType
	err := jt.Scan(123)
	if err == nil {
		t.Error("expected error for non-string value")
	}
}

func TestJobStatus_Scan_ValidString(t *testing.T) {
	var js JobStatus
	err := js.Scan("completed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if js != JobStatusCompleted {
		t.Errorf("expected %q, got %q", JobStatusCompleted, js)
	}
}

func TestJobStatus_Scan_Nil(t *testing.T) {
	var js JobStatus
	err := js.Scan(nil)
	if err == nil {
		t.Error("expected error for nil value")
	}
}

func TestJobStatus_Scan_NonString(t *testing.T) {
	var js JobStatus
	err := js.Scan(42)
	if err == nil {
		t.Error("expected error for non-string value")
	}
}
