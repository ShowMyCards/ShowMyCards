package services

import (
	"backend/models"
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSettingsServiceTest(t *testing.T) (*SettingsService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}

	if err := db.AutoMigrate(&models.Setting{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return NewSettingsService(db), db
}

// InitializeDefaults tests

func TestSettingsService_InitializeDefaults(t *testing.T) {
	_, db := setupSettingsServiceTest(t)

	// Verify all default settings were created
	expectedDefaults := map[string]string{
		"bulk_data_auto_update":           "true",
		"bulk_data_update_time":           "03:00",
		"bulk_data_url":                   "https://api.scryfall.com/bulk-data",
		"bulk_data_last_update":           "",
		"bulk_data_last_update_status":    "",
		"set_data_auto_update":            "true",
		"set_data_update_time":            "02:30",
		"set_data_last_update":            "",
		"set_data_last_update_status":     "",
		"scryfall_default_search":         "game:paper",
		"scryfall_unique_mode":            "cards",
		"job_cleanup_last_run":            "",
		"scheduler_catchup_enabled":       "true",
		"scheduler_catchup_delay_seconds": "60",
	}

	for key, expectedValue := range expectedDefaults {
		var setting models.Setting
		err := db.Where("key = ?", key).First(&setting).Error
		if err != nil {
			t.Errorf("expected default setting %s to exist, but got error: %v", key, err)
			continue
		}

		if setting.Value != expectedValue {
			t.Errorf("setting %s: expected value %s, got %s", key, expectedValue, setting.Value)
		}
	}

	// Verify count
	var count int64
	db.Model(&models.Setting{}).Count(&count)
	if count != int64(len(expectedDefaults)) {
		t.Errorf("expected %d default settings, got %d", len(expectedDefaults), count)
	}
}

func TestSettingsService_InitializeDefaults_IdempotentOnSecondCall(t *testing.T) {
	_, db := setupSettingsServiceTest(t)

	// Count settings after first initialization
	var countBefore int64
	db.Model(&models.Setting{}).Count(&countBefore)

	// Create service again (should not duplicate)
	NewSettingsService(db)

	// Count settings after second initialization
	var countAfter int64
	db.Model(&models.Setting{}).Count(&countAfter)

	if countBefore != countAfter {
		t.Errorf("expected idempotent initialization, before: %d, after: %d", countBefore, countAfter)
	}
}

// Get tests

func TestSettingsService_Get_Success(t *testing.T) {
	service, _ := setupSettingsServiceTest(t)

	// Default settings exist
	value, err := service.Get(context.Background(),"bulk_data_auto_update")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if value != "true" {
		t.Errorf("expected value 'true', got '%s'", value)
	}
}

func TestSettingsService_Get_NotFound(t *testing.T) {
	service, _ := setupSettingsServiceTest(t)

	value, err := service.Get(context.Background(),"nonexistent_key")
	if err == nil {
		t.Error("expected error for non-existent key")
	}

	if value != "" {
		t.Errorf("expected empty value, got '%s'", value)
	}
}

// Set tests

func TestSettingsService_Set_NewSetting(t *testing.T) {
	service, db := setupSettingsServiceTest(t)

	err := service.Set(context.Background(),"new_setting", "new_value")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify in database
	var setting models.Setting
	db.Where("key = ?", "new_setting").First(&setting)

	if setting.Value != "new_value" {
		t.Errorf("expected value 'new_value', got '%s'", setting.Value)
	}
}

func TestSettingsService_Set_UpdateExisting(t *testing.T) {
	service, db := setupSettingsServiceTest(t)

	// Update existing default
	err := service.Set(context.Background(),"bulk_data_auto_update", "false")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify in database
	var setting models.Setting
	db.Where("key = ?", "bulk_data_auto_update").First(&setting)

	if setting.Value != "false" {
		t.Errorf("expected value 'false', got '%s'", setting.Value)
	}

	// Verify no duplicate was created
	var count int64
	db.Model(&models.Setting{}).Where("key = ?", "bulk_data_auto_update").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 setting with key 'bulk_data_auto_update', got %d", count)
	}
}

// GetAll tests

func TestSettingsService_GetAll_Success(t *testing.T) {
	service, _ := setupSettingsServiceTest(t)

	settings, err := service.GetAll(context.Background())
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}

	if len(settings) < 7 {
		t.Errorf("expected at least 7 default settings, got %d", len(settings))
	}

	// Verify some defaults exist
	if settings["bulk_data_auto_update"] != "true" {
		t.Errorf("expected bulk_data_auto_update='true', got '%s'", settings["bulk_data_auto_update"])
	}
}

func TestSettingsService_GetAll_EmptyDatabase(t *testing.T) {
	t.Helper()

	// Create service without defaults initialization
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}

	if err := db.AutoMigrate(&models.Setting{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	service := &SettingsService{db: db}

	settings, err := service.GetAll(context.Background())
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}

	if len(settings) != 0 {
		t.Errorf("expected 0 settings, got %d", len(settings))
	}
}

// GetBool tests

func TestSettingsService_GetBool_True(t *testing.T) {
	service, _ := setupSettingsServiceTest(t)

	value := service.GetBool(context.Background(),"bulk_data_auto_update", false)
	if !value {
		t.Error("expected true, got false")
	}
}

func TestSettingsService_GetBool_False(t *testing.T) {
	service, _ := setupSettingsServiceTest(t)

	service.Set(context.Background(),"test_bool", "false")

	value := service.GetBool(context.Background(),"test_bool", true)
	if value {
		t.Error("expected false, got true")
	}
}

func TestSettingsService_GetBool_InvalidValue(t *testing.T) {
	service, _ := setupSettingsServiceTest(t)

	service.Set(context.Background(),"test_bool", "not_a_bool")

	value := service.GetBool(context.Background(),"test_bool", true)
	if !value {
		t.Error("expected default value true when parsing fails, got false")
	}
}

func TestSettingsService_GetBool_NotFound(t *testing.T) {
	service, _ := setupSettingsServiceTest(t)

	value := service.GetBool(context.Background(),"nonexistent", true)
	if !value {
		t.Error("expected default value true, got false")
	}

	value = service.GetBool(context.Background(),"nonexistent", false)
	if value {
		t.Error("expected default value false, got true")
	}
}

// GetInt tests

func TestSettingsService_GetInt_Success(t *testing.T) {
	service, _ := setupSettingsServiceTest(t)

	service.Set(context.Background(),"test_int", "42")

	value := service.GetInt(context.Background(),"test_int", 0)
	if value != 42 {
		t.Errorf("expected 42, got %d", value)
	}
}

func TestSettingsService_GetInt_InvalidValue(t *testing.T) {
	service, _ := setupSettingsServiceTest(t)

	service.Set(context.Background(),"test_int", "not_an_int")

	value := service.GetInt(context.Background(),"test_int", 99)
	if value != 99 {
		t.Errorf("expected default value 99 when parsing fails, got %d", value)
	}
}

func TestSettingsService_GetInt_NotFound(t *testing.T) {
	service, _ := setupSettingsServiceTest(t)

	value := service.GetInt(context.Background(),"nonexistent", 123)
	if value != 123 {
		t.Errorf("expected default value 123, got %d", value)
	}
}

func TestSettingsService_GetInt_NegativeValue(t *testing.T) {
	service, _ := setupSettingsServiceTest(t)

	service.Set(context.Background(),"test_int", "-50")

	value := service.GetInt(context.Background(),"test_int", 0)
	if value != -50 {
		t.Errorf("expected -50, got %d", value)
	}
}

// GetTime tests

func TestSettingsService_GetTime_Success(t *testing.T) {
	service, _ := setupSettingsServiceTest(t)

	now := time.Now()
	service.SetTime(context.Background(),"test_time", now)

	retrieved, err := service.GetTime(context.Background(),"test_time")
	if err != nil {
		t.Fatalf("GetTime failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("expected non-nil time")
	}

	// Compare times (allowing small difference due to formatting)
	diff := retrieved.Sub(now).Abs()
	if diff > time.Second {
		t.Errorf("time difference too large: %v", diff)
	}
}

func TestSettingsService_GetTime_EmptyValue(t *testing.T) {
	service, _ := setupSettingsServiceTest(t)

	// bulk_data_last_update is empty by default
	retrieved, err := service.GetTime(context.Background(),"bulk_data_last_update")
	if err != nil {
		t.Fatalf("GetTime failed: %v", err)
	}

	if retrieved != nil {
		t.Errorf("expected nil for empty value, got %v", retrieved)
	}
}

func TestSettingsService_GetTime_InvalidFormat(t *testing.T) {
	service, _ := setupSettingsServiceTest(t)

	service.Set(context.Background(),"test_time", "not_a_time")

	_, err := service.GetTime(context.Background(),"test_time")
	if err == nil {
		t.Error("expected error for invalid time format")
	}
}

func TestSettingsService_GetTime_NotFound(t *testing.T) {
	service, _ := setupSettingsServiceTest(t)

	_, err := service.GetTime(context.Background(),"nonexistent")
	if err == nil {
		t.Error("expected error for non-existent key")
	}
}

// SetTime tests

func TestSettingsService_SetTime_Success(t *testing.T) {
	service, db := setupSettingsServiceTest(t)

	testTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
	err := service.SetTime(context.Background(),"test_time", testTime)
	if err != nil {
		t.Fatalf("SetTime failed: %v", err)
	}

	// Verify in database
	var setting models.Setting
	db.Where("key = ?", "test_time").First(&setting)

	expectedValue := testTime.Format(time.RFC3339)
	if setting.Value != expectedValue {
		t.Errorf("expected value %s, got %s", expectedValue, setting.Value)
	}

	// Verify round-trip
	retrieved, _ := service.GetTime(context.Background(),"test_time")
	if !retrieved.Equal(testTime) {
		t.Errorf("expected %v, got %v", testTime, retrieved)
	}
}

func TestSettingsService_SetTime_UpdateExisting(t *testing.T) {
	service, db := setupSettingsServiceTest(t)

	time1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	time2 := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	service.SetTime(context.Background(),"test_time", time1)
	service.SetTime(context.Background(),"test_time", time2)

	// Verify only one setting exists
	var count int64
	db.Model(&models.Setting{}).Where("key = ?", "test_time").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 setting, got %d", count)
	}

	// Verify latest value
	retrieved, _ := service.GetTime(context.Background(),"test_time")
	if !retrieved.Equal(time2) {
		t.Errorf("expected %v, got %v", time2, retrieved)
	}
}
