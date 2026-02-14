package models

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupStorageTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	if err := db.AutoMigrate(&StorageLocation{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return db
}

func TestStorageType_IsValid(t *testing.T) {
	tests := []struct {
		name        string
		storageType StorageType
		expected    bool
	}{
		{"Box", Box, true},
		{"Binder", Binder, true},
		{"Invalid", StorageType("InvalidType"), false},
		{"Empty", StorageType(""), false},
		{"Lowercase", StorageType("box"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.storageType.IsValid()
			if result != tt.expected {
				t.Errorf("IsValid() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestStorageLocation_ValidateStorageLocation(t *testing.T) {
	db := setupStorageTestDB(t)

	tests := []struct {
		name        string
		storage     *StorageLocation
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid Box",
			storage:     &StorageLocation{Name: "Main Box", StorageType: Box},
			expectError: false,
		},
		{
			name:        "Valid Binder",
			storage:     &StorageLocation{Name: "Binder 1", StorageType: Binder},
			expectError: false,
		},
		{
			name:        "Empty Name",
			storage:     &StorageLocation{Name: "", StorageType: Box},
			expectError: true,
			errorMsg:    "name cannot be empty",
		},
		{
			name:        "Invalid StorageType",
			storage:     &StorageLocation{Name: "Test", StorageType: StorageType("Invalid")},
			expectError: true,
			errorMsg:    "invalid storage type",
		},
		{
			name:        "Both Invalid",
			storage:     &StorageLocation{Name: "", StorageType: StorageType("Invalid")},
			expectError: true,
			errorMsg:    "name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.storage.ValidateStorageLocation(db)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestStorageLocation_BeforeCreate(t *testing.T) {
	db := setupStorageTestDB(t)

	tests := []struct {
		name        string
		storage     *StorageLocation
		expectError bool
	}{
		{
			name:        "Valid Create",
			storage:     &StorageLocation{Name: "Test Box", StorageType: Box},
			expectError: false,
		},
		{
			name:        "Invalid Create - Empty Name",
			storage:     &StorageLocation{Name: "", StorageType: Box},
			expectError: true,
		},
		{
			name:        "Invalid Create - Invalid Type",
			storage:     &StorageLocation{Name: "Test", StorageType: StorageType("Invalid")},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Create(tt.storage).Error
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

func TestStorageLocation_BeforeUpdate(t *testing.T) {
	db := setupStorageTestDB(t)

	tests := []struct {
		name        string
		updateFunc  func(*StorageLocation)
		expectError bool
	}{
		{
			name: "Valid Update - Name",
			updateFunc: func(s *StorageLocation) {
				s.Name = "Updated Name"
			},
			expectError: false,
		},
		{
			name: "Valid Update - Type",
			updateFunc: func(s *StorageLocation) {
				s.StorageType = Binder
			},
			expectError: false,
		},
		{
			name: "Invalid Update - Empty Name",
			updateFunc: func(s *StorageLocation) {
				s.Name = ""
			},
			expectError: true,
		},
		{
			name: "Invalid Update - Invalid Type",
			updateFunc: func(s *StorageLocation) {
				s.StorageType = StorageType("Invalid")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh storage location for each test
			testStorage := &StorageLocation{Name: "Original Name", StorageType: Box}
			if err := db.Create(testStorage).Error; err != nil {
				t.Fatalf("failed to create storage: %v", err)
			}

			// Apply update
			tt.updateFunc(testStorage)

			// Try to save (triggers BeforeUpdate)
			err := db.Save(testStorage).Error
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

func TestStorageLocation_CreateSuccess(t *testing.T) {
	db := setupStorageTestDB(t)

	storage := &StorageLocation{
		Name:        "Test Box",
		StorageType: Box,
	}

	if err := db.Create(storage).Error; err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Verify storage was created
	if storage.ID == 0 {
		t.Error("expected ID to be set after create")
	}
	if storage.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set after create")
	}
	if storage.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set after create")
	}

	// Verify we can retrieve it
	var retrieved StorageLocation
	if err := db.First(&retrieved, storage.ID).Error; err != nil {
		t.Fatalf("failed to retrieve storage: %v", err)
	}

	if retrieved.Name != "Test Box" {
		t.Errorf("expected name 'Test Box', got '%s'", retrieved.Name)
	}
	if retrieved.StorageType != Box {
		t.Errorf("expected storage type Box, got %v", retrieved.StorageType)
	}
}

func TestStorageLocation_UpdateSuccess(t *testing.T) {
	db := setupStorageTestDB(t)

	// Create initial storage
	storage := &StorageLocation{
		Name:        "Original",
		StorageType: Box,
	}
	if err := db.Create(storage).Error; err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	originalUpdatedAt := storage.UpdatedAt

	// Update the storage
	storage.Name = "Updated"
	storage.StorageType = Binder
	if err := db.Save(storage).Error; err != nil {
		t.Fatalf("failed to update storage: %v", err)
	}

	// Verify updates
	if storage.Name != "Updated" {
		t.Errorf("expected name 'Updated', got '%s'", storage.Name)
	}
	if storage.StorageType != Binder {
		t.Errorf("expected storage type Binder, got %v", storage.StorageType)
	}
	if !storage.UpdatedAt.After(originalUpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestStorageLocation_DeleteSuccess(t *testing.T) {
	db := setupStorageTestDB(t)

	storage := &StorageLocation{
		Name:        "Test",
		StorageType: Box,
	}
	if err := db.Create(storage).Error; err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Delete the storage
	if err := db.Delete(storage).Error; err != nil {
		t.Fatalf("failed to delete storage: %v", err)
	}

	// Verify it was deleted
	var count int64
	db.Model(&StorageLocation{}).Where("id = ?", storage.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected storage to be deleted, found %d records", count)
	}
}

func TestStorageLocation_UniqueNames(t *testing.T) {
	db := setupStorageTestDB(t)

	// Create first storage
	storage1 := &StorageLocation{
		Name:        "Duplicate Name",
		StorageType: Box,
	}
	if err := db.Create(storage1).Error; err != nil {
		t.Fatalf("failed to create first storage: %v", err)
	}

	// Create second storage with same name (should succeed - no unique constraint)
	storage2 := &StorageLocation{
		Name:        "Duplicate Name",
		StorageType: Binder,
	}
	if err := db.Create(storage2).Error; err != nil {
		t.Fatalf("failed to create second storage: %v", err)
	}

	// Verify both exist
	var count int64
	db.Model(&StorageLocation{}).Where("name = ?", "Duplicate Name").Count(&count)
	if count != 2 {
		t.Errorf("expected 2 storage locations with duplicate name, found %d", count)
	}
}
