// Package database provides database connection and migration management for SQLite.
// It handles database lifecycle, custom migrations, and GORM configuration.
package database

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"backend/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Client holds the database connection
type Client struct {
	DB *gorm.DB
}

// NewClient creates a new database client
func NewClient(dbPath string) (*Client, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Connect to database with silent logger (only logs errors)
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// SQLite only supports one writer at a time
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Run migrations
	if err := migrate(db); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Info("connected to database", "path", dbPath)
	return &Client{DB: db}, nil
}

// Close closes the database connection
func (c *Client) Close() error {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	return sqlDB.Close()
}

// migrate runs database migrations
func migrate(db *gorm.DB) error {
	// Run auto-migrations for all models
	if err := db.AutoMigrate(
		&models.StorageLocation{},
		&models.SortingRule{},
		&models.Inventory{},
		&models.List{},
		&models.ListItem{},
		&models.Setting{},
		&models.Job{},
		&models.Card{},
		&models.Set{},
	); err != nil {
		return err
	}

	// Run custom migrations for features not supported by AutoMigrate
	if err := customMigrations(db); err != nil {
		return err
	}

	return nil
}

// tableColumns returns a set of column names for the given table.
// The table parameter must be a trusted constant — PRAGMA does not support parameterised queries.
func tableColumns(db *gorm.DB, table string) (map[string]bool, error) {
	type columnInfo struct {
		Name string `gorm:"column:name"`
	}
	var cols []columnInfo
	if err := db.Raw("PRAGMA table_xinfo(" + table + ")").Scan(&cols).Error; err != nil {
		return nil, fmt.Errorf("failed to read columns for table %s: %w", table, err)
	}
	m := make(map[string]bool, len(cols))
	for _, c := range cols {
		m[c.Name] = true
	}
	return m, nil
}

// customMigrations handles database-specific features like generated columns
func customMigrations(db *gorm.DB) error {
	// Drop legacy bulk_cards table if it exists
	if err := db.Exec("DROP TABLE IF EXISTS bulk_cards").Error; err != nil {
		return fmt.Errorf("failed to drop legacy bulk_cards table: %w", err)
	}

	// Check if cards table exists first
	var tableExists int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='cards'").Scan(&tableExists)
	if tableExists == 0 {
		// Table doesn't exist yet, skip (will be created by AutoMigrate)
		return nil
	}

	// Add generated columns if they don't already exist
	existingCols, err := tableColumns(db, "cards")
	if err != nil {
		return err
	}

	if !existingCols["name"] {
		if err := db.Exec(`
			ALTER TABLE cards ADD COLUMN name TEXT
			GENERATED ALWAYS AS (json_extract(raw_json, '$.name')) STORED
		`).Error; err != nil {
			return fmt.Errorf("failed to add name column: %w", err)
		}
	}

	if !existingCols["set_code"] {
		if err := db.Exec(`
			ALTER TABLE cards ADD COLUMN set_code TEXT
			GENERATED ALWAYS AS (json_extract(raw_json, '$.set')) STORED
		`).Error; err != nil {
			return fmt.Errorf("failed to add set_code column: %w", err)
		}
	}

	// Create indexes (IF NOT EXISTS is natively supported)
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_cards_name ON cards(name)").Error; err != nil {
		return fmt.Errorf("failed to create name index: %w", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_cards_set_code ON cards(set_code)").Error; err != nil {
		return fmt.Errorf("failed to create set_code index: %w", err)
	}

	return nil
}

