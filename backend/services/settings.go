package services

import (
	"backend/models"
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// SettingsService handles application settings
type SettingsService struct {
	db *gorm.DB
}

// NewSettingsService creates a new settings service
func NewSettingsService(db *gorm.DB) *SettingsService {
	service := &SettingsService{db: db}

	// Initialize default settings on first run
	service.initializeDefaults()

	return service
}

// initializeDefaults creates default settings if they don't exist
func (s *SettingsService) initializeDefaults() {
	defaults := map[string]string{
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

	for key, value := range defaults {
		var count int64
		if err := s.db.Model(&models.Setting{}).Where("key = ?", key).Count(&count).Error; err != nil {
			slog.Warn("failed to check setting existence", "key", key, "error", err)
			continue
		}

		if count == 0 {
			// Setting doesn't exist, create it
			setting := models.Setting{Key: key, Value: value}
			if err := s.db.Create(&setting).Error; err != nil {
				slog.Warn("failed to create default setting", "key", key, "error", err)
			}
		}
	}
}

// Get retrieves a setting value by key
func (s *SettingsService) Get(ctx context.Context, key string) (string, error) {
	var setting models.Setting
	if err := s.db.WithContext(ctx).Where("key = ?", key).First(&setting).Error; err != nil {
		return "", err
	}
	return setting.Value, nil
}

// Set creates or updates a setting
func (s *SettingsService) Set(ctx context.Context, key, value string) error {
	var setting models.Setting
	err := s.db.WithContext(ctx).Where("key = ?", key).First(&setting).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new setting
		setting = models.Setting{Key: key, Value: value}
		return s.db.WithContext(ctx).Create(&setting).Error
	} else if err != nil {
		return err
	}

	// Update existing setting
	setting.Value = value
	return s.db.WithContext(ctx).Save(&setting).Error
}

// GetAll retrieves all settings as a map
func (s *SettingsService) GetAll(ctx context.Context) (map[string]string, error) {
	var settings []models.Setting
	if err := s.db.WithContext(ctx).Find(&settings).Error; err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, setting := range settings {
		result[setting.Key] = setting.Value
	}
	return result, nil
}

// GetBool retrieves a setting as a boolean
func (s *SettingsService) GetBool(ctx context.Context, key string, defaultValue bool) bool {
	value, err := s.Get(ctx, key)
	if err != nil {
		return defaultValue
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// GetInt retrieves a setting as an integer
func (s *SettingsService) GetInt(ctx context.Context, key string, defaultValue int) int {
	value, err := s.Get(ctx, key)
	if err != nil {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// GetTime retrieves a setting as a time.Time
// Expects value in RFC3339 format
func (s *SettingsService) GetTime(ctx context.Context, key string) (*time.Time, error) {
	value, err := s.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// SetTime stores a time.Time as a setting
func (s *SettingsService) SetTime(ctx context.Context, key string, value time.Time) error {
	return s.Set(ctx, key, value.Format(time.RFC3339))
}

// ValidSettingKeys returns the set of valid setting keys
func ValidSettingKeys() map[string]bool {
	return map[string]bool{
		"bulk_data_auto_update":           true,
		"bulk_data_update_time":           true,
		"bulk_data_url":                   true,
		"bulk_data_last_update":           true,
		"bulk_data_last_update_status":    true,
		"set_data_auto_update":            true,
		"set_data_update_time":            true,
		"set_data_last_update":            true,
		"set_data_last_update_status":     true,
		"scryfall_default_search":         true,
		"scryfall_unique_mode":            true,
		"job_cleanup_last_run":            true,
		"scheduler_catchup_enabled":       true,
		"scheduler_catchup_delay_seconds": true,
	}
}

// SetBulk updates multiple settings in a single transaction
func (s *SettingsService) SetBulk(ctx context.Context, settings map[string]string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for key, value := range settings {
			var setting models.Setting
			err := tx.Where("key = ?", key).First(&setting).Error

			if errors.Is(err, gorm.ErrRecordNotFound) {
				setting = models.Setting{Key: key, Value: value}
				if err := tx.Create(&setting).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else {
				setting.Value = value
				if err := tx.Save(&setting).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
