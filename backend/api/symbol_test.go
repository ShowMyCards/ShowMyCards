package api

import (
	"backend/models"
	"backend/scryfall"
	"backend/services"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSymbolTestApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := db.AutoMigrate(&models.Symbol{}, &models.Job{}, &models.Setting{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	jobService := services.NewJobService(db)
	settingsService := services.NewSettingsService(db)
	scryfallClient, err := scryfall.NewClient()
	if err != nil {
		t.Fatalf("failed to create scryfall client: %v", err)
	}
	symbolDataService := services.NewSymbolDataService(db, jobService, settingsService, scryfallClient)
	handler := NewSymbolHandler(db, symbolDataService)

	app := fiber.New()
	symbols := app.Group("/api/symbols")
	symbols.Get("/:symbol", handler.GetSVG)

	return app, db
}

func TestSymbolGetSVG_Found(t *testing.T) {
	app, db := setupSymbolTestApp(t)

	db.Create(&models.Symbol{
		Code:    "T",
		Symbol:  "{T}",
		English: "tap this permanent",
		SVG:     "<svg id=\"tap\"></svg>",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/symbols/T", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "image/svg+xml" {
		t.Errorf("expected content type %q, got %q", "image/svg+xml", ct)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "<svg id=\"tap\"></svg>" {
		t.Errorf("unexpected svg body: %q", string(body))
	}
}

func TestSymbolGetSVG_NormalizesBraces(t *testing.T) {
	app, db := setupSymbolTestApp(t)

	db.Create(&models.Symbol{
		Code:   "W",
		Symbol: "{W}",
		SVG:    "<svg id=\"white\"></svg>",
	})

	// Lowercase lookup should still resolve to the uppercase-stored code.
	req := httptest.NewRequest(http.MethodGet, "/api/symbols/w", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestSymbolGetSVG_NotFound(t *testing.T) {
	app, _ := setupSymbolTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/api/symbols/ZZZ", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

func TestSymbolGetSVG_EmptySVGTreatedAsNotFound(t *testing.T) {
	app, db := setupSymbolTestApp(t)

	db.Create(&models.Symbol{
		Code:   "2",
		Symbol: "{2}",
		SVG:    "",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/symbols/2", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}
