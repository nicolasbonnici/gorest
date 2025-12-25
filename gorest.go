package gorest

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/nicolasbonnici/gorest/config"
	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/mysql"
	_ "github.com/nicolasbonnici/gorest/database/postgres"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
	"github.com/nicolasbonnici/gorest/logger"
	"github.com/nicolasbonnici/gorest/middleware"
	"github.com/nicolasbonnici/gorest/plugin"
	"github.com/nicolasbonnici/gorest/pluginloader"
	"github.com/nicolasbonnici/gorest/response"
)

var Version = "dev"

type Config struct {
	ConfigPath     string
	RegisterRoutes func(app *fiber.App, db database.Database, paginationLimit, paginationMaxLimit int, pluginRegistry *plugin.PluginRegistry)
}

func Start(cfg Config) {
	if cfg.ConfigPath == "" {
		cfg.ConfigPath = "."
	}

	appConfig, err := config.Load(cfg.ConfigPath)
	if err != nil {
		logger.Log.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	if err := appConfig.Validate(); err != nil {
		logger.Log.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}

	// Initialize response package with version
	response.Initialize(Version)

	db, err := database.Open("", appConfig.Database.URL)
	if err != nil {
		logger.Log.Error("DB connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	app := fiber.New(fiber.Config{
		BodyLimit:    4 * 1024 * 1024,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	app.Use(middleware.Security())
	app.Use(middleware.CORS(appConfig.Server.CORSOrigins))

	enrichedConfigs := pluginloader.InjectSharedConfig(appConfig.Plugins, db, appConfig)

	// Load plugins (plugins are NOT automatically applied - user must use app.Use() or fiber groups)
	pluginRegistry, err := pluginloader.LoadPlugins(enrichedConfigs, Version)
	if err != nil {
		logger.Log.Error("Failed to load plugins", "error", err)
		os.Exit(1)
	}

	// Apply global middleware before setting up any endpoints (including plugin endpoints)
	// This ensures all endpoints, including /health and /login, are protected by security middleware
	pluginloader.ApplyGlobalMiddleware(pluginRegistry, app)

	// Setup endpoints for any plugins that implement EndpointSetup interface (e.g. auth /login, health /health)
	if err := pluginloader.SetupPluginEndpoints(pluginRegistry, app); err != nil {
		logger.Log.Error("Failed to setup plugin endpoints", "error", err)
		os.Exit(1)
	}

	if cfg.RegisterRoutes != nil {
		cfg.RegisterRoutes(app, db, appConfig.Pagination.DefaultLimit, appConfig.Pagination.MaxLimit, pluginRegistry)
	} else {
		logger.Log.Warn("No routes registered. Set Config.RegisterRoutes to register your API endpoints.")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		port := fmt.Sprintf("%d", appConfig.Server.Port)
		logger.Log.Info("REST API running", "port", port, "url", "http://localhost:"+port, "version", Version)
		logger.Log.Info("Health check available", "url", "http://localhost:"+port+"/health")
		logger.Log.Info("Environment", "env", appConfig.Server.Environment)
		if err := app.Listen(":" + port); err != nil {
			logger.Log.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	logger.Log.Info("Shutting down server gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Log.Warn("Server forced to shutdown", "error", err)
	}

	db.Close()
	logger.Log.Info("Server shutdown complete")
}

func FindProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}
