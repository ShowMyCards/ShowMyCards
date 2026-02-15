package server

import (
	"backend/database"
	"backend/scryfall"
	"backend/services"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

// Server holds the main application components
type Server struct {
	app             *fiber.App
	db              *database.Client
	scryfall        *scryfall.Client
	settingsService *services.SettingsService
	jobService      *services.JobService
	bulkDataService *services.BulkDataService
	setDataService  *services.SetDataService
	dataDir         string
	appCtx          context.Context
}

// NewServer creates a new server instance
func NewServer(appCtx context.Context, dbClient *database.Client, scryfallClient *scryfall.Client, settingsService *services.SettingsService, jobService *services.JobService, bulkDataService *services.BulkDataService, setDataService *services.SetDataService, dataDir string) *Server {
	app := fiber.New(fiber.Config{
		BodyLimit:    4 * 1024 * 1024, // 4MB
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			slog.Error("request failed", "method", c.Method(), "path", c.Path(), "error", err)
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Middleware
	app.Use(logger.New())
	// Default allows the standard SvelteKit dev server origin.
	// Override via ALLOWED_ORIGINS env var (comma-separated) for production deployments.
	allowedOrigins := []string{"http://localhost:5173"}
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		allowedOrigins = strings.Split(origins, ",")
	}
	for _, origin := range allowedOrigins {
		if strings.TrimSpace(origin) == "*" {
			slog.Warn("ALLOWED_ORIGINS contains wildcard '*', all origins will be accepted", "component", "server")
			break
		}
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Content-Type"},
	}))

	return &Server{
		app:             app,
		db:              dbClient,
		scryfall:        scryfallClient,
		settingsService: settingsService,
		jobService:      jobService,
		bulkDataService: bulkDataService,
		setDataService:  setDataService,
		dataDir:         dataDir,
		appCtx:          appCtx,
	}
}

// Start initializes and starts the server
func (s *Server) Start() error {
	// Setup routes
	s.setupRoutes()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("invalid PORT value %q: must be a numeric port number", port)
	}

	slog.Info("starting server", "port", port)
	return s.app.Listen(":" + port)
}

// Close shuts down the server gracefully
func (s *Server) Close() error {
	return s.app.Shutdown()
}

func (s *Server) setupRoutes() {
	HealthRoutes(s.app, s.db.DB)
	DashboardRoutes(s.app, s.db.DB)
	StorageRoutes(s.app, s.db.DB)
	SortingRulesRoutes(s.app, s.db.DB)
	InventoryRoutes(s.app, s.db.DB)
	ListRoutes(s.app, s.db.DB)
	SearchRoutes(s.app, s.scryfall, s.db.DB, s.settingsService)
	SettingsRoutes(s.app, s.settingsService)
	JobsRoutes(s.app, s.jobService)
	BulkDataRoutes(s.app, s.bulkDataService, s.appCtx)
	SetRoutes(s.app, s.db.DB, s.setDataService, s.dataDir, s.appCtx)
	s.RegisterSchedulerRoutes(s.app)
}
