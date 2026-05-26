package api

import (
	"backend/models"
	"backend/services"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupBannerTestApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	if err := db.AutoMigrate(&models.Job{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	service := services.NewBannerService(services.NewJobService(db))
	handler := NewBannerHandler(service)

	app := fiber.New()
	app.Get("/api/banners", handler.GetAll)

	return app, db
}

func TestBannersGetAll_Empty(t *testing.T) {
	app, _ := setupBannerTestApp(t)

	req := httptest.NewRequest("GET", "/api/banners", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	// An empty result must serialise as [], not null, so the frontend can
	// safely iterate it.
	if string(body) != "[]" {
		t.Errorf("expected empty array body, got %s", string(body))
	}
}

func TestBannersGetAll_WithBanner(t *testing.T) {
	app, db := setupBannerTestApp(t)

	db.Create(&models.Job{
		Type:     models.JobTypeBulkDataImport,
		Status:   models.JobStatusInProgress,
		Metadata: "{}",
	})

	req := httptest.NewRequest("GET", "/api/banners", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var banners []models.Banner
	if err := json.Unmarshal(body, &banners); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(banners) != 1 {
		t.Fatalf("expected 1 banner, got %d", len(banners))
	}
	if banners[0].Severity != models.BannerSeverityInfo {
		t.Errorf("expected info severity, got %s", banners[0].Severity)
	}
}
