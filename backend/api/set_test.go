package api

import (
	"backend/models"
	"backend/scryfall"
	"backend/services"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSetTestApp(t *testing.T) (*fiber.App, *gorm.DB, string) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := db.AutoMigrate(&models.Set{}, &models.Job{}, &models.Setting{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	dataDir := t.TempDir()

	// Create set-icons directory with a test icon
	iconsDir := filepath.Join(dataDir, "set-icons")
	if err := os.MkdirAll(iconsDir, 0755); err != nil {
		t.Fatalf("failed to create icons dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(iconsDir, "tst.svg"), []byte("<svg></svg>"), 0644); err != nil {
		t.Fatalf("failed to write test icon: %v", err)
	}

	jobService := services.NewJobService(db)
	settingsService := services.NewSettingsService(db)
	scryfallClient, err := scryfall.NewClient()
	if err != nil {
		t.Fatalf("failed to create scryfall client: %v", err)
	}
	setDataService := services.NewSetDataService(db, jobService, settingsService, scryfallClient, dataDir)
	handler := NewSetHandler(db, setDataService, dataDir)

	app := fiber.New()
	sets := app.Group("/sets")
	sets.Get("/", handler.List)
	sets.Get("/id/:id", handler.GetByID)
	sets.Get("/code/:code", handler.GetByCode)
	sets.Get("/code/:code/icon", handler.GetIcon)

	return app, db, dataDir
}

func TestSetList_Empty(t *testing.T) {
	app, _, _ := setupSetTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/sets/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestSetList_WithData(t *testing.T) {
	app, db, _ := setupSetTestApp(t)

	db.Create(&models.Set{
		ScryfallID: "set-1",
		Code:       "tst",
		Name:       "Test Set",
		SetType:    "expansion",
		CardCount:  100,
	})

	req := httptest.NewRequest(http.MethodGet, "/sets/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestSetGetByID_Found(t *testing.T) {
	app, db, _ := setupSetTestApp(t)

	db.Create(&models.Set{
		ScryfallID: "abc-123",
		Code:       "tst",
		Name:       "Test Set",
	})

	req := httptest.NewRequest(http.MethodGet, "/sets/id/abc-123", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var set models.Set
	if err := json.Unmarshal(body, &set); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if set.Name != "Test Set" {
		t.Errorf("expected name %q, got %q", "Test Set", set.Name)
	}
}

func TestSetGetByID_NotFound(t *testing.T) {
	app, _, _ := setupSetTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/sets/id/nonexistent", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

func TestSetGetByCode_Found(t *testing.T) {
	app, db, _ := setupSetTestApp(t)

	db.Create(&models.Set{
		ScryfallID: "abc-123",
		Code:       "mkm",
		Name:       "Murders at Karlov Manor",
	})

	req := httptest.NewRequest(http.MethodGet, "/sets/code/mkm", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestSetGetByCode_NotFound(t *testing.T) {
	app, _, _ := setupSetTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/sets/code/zzz", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

func TestSetGetIcon_ValidCode(t *testing.T) {
	app, _, _ := setupSetTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/sets/code/tst/icon", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "image/svg+xml" {
		t.Errorf("expected content type %q, got %q", "image/svg+xml", contentType)
	}
}

func TestSetGetIcon_NotFound(t *testing.T) {
	app, _, _ := setupSetTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/sets/code/nonexistent/icon", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

func TestSetGetIcon_PathTraversal(t *testing.T) {
	app, _, _ := setupSetTestApp(t)

	// Attempt path traversal — should NOT return 200
	req := httptest.NewRequest(http.MethodGet, "/sets/code/..%2F..%2F..%2Fetc%2Fpasswd/icon", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should be rejected — either 400 (containment check) or 404 (cleaned path not found)
	if resp.StatusCode == http.StatusOK {
		t.Errorf("path traversal attempt should not succeed with 200, got %d", resp.StatusCode)
	}
}

func TestSetGetIcon_PathTraversalDotDot(t *testing.T) {
	app, _, _ := setupSetTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/sets/code/../icon", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should be either 400 (blocked) or 404 (not found), but NOT 200
	if resp.StatusCode == http.StatusOK {
		t.Error("path traversal attempt should not succeed with 200")
	}
}
