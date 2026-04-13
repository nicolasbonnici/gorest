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
	"github.com/nicolasbonnici/gorest/migrations"
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
	app.Use(middleware.RequestID())
	app.Use(middleware.Logger())
	app.Use(middleware.ContentNegotiation())

	if appConfig.Server.CompressionEnabled {
		app.Use(middleware.Compress(appConfig.Server.CompressionLevel))
	}

	if appConfig.Server.RateLimitEnabled {
		app.Use(middleware.RateLimit(appConfig.Server.RateLimitRPS, appConfig.Server.RateLimitBurst))
	}

	enrichedConfigs := pluginloader.InjectSharedConfig(appConfig.Plugins, db, appConfig, nil)

	pluginRegistry, err := pluginloader.LoadPlugins(enrichedConfigs, Version)
	if err != nil {
		logger.Log.Error("Failed to load plugins", "error", err)
		os.Exit(1)
	}

	if err := runPluginMigrations(context.Background(), db, pluginRegistry, appConfig.Plugins); err != nil {
		if err != migrations.ErrNoPendingMigrations {
			logger.Log.Error("Failed to run migrations", "error", err)
			os.Exit(1)
		}
	}

	if openAPIPlugin, ok := pluginRegistry.Get("openapi"); ok {
		enrichedConfigs = pluginloader.InjectSharedConfig(appConfig.Plugins, db, appConfig, pluginRegistry)
		for _, cfg := range enrichedConfigs {
			if cfg.Name == "openapi" {
				if err := openAPIPlugin.Initialize(cfg.Config); err != nil {
					logger.Log.Error("Failed to reinitialize openapi plugin", "error", err)
					os.Exit(1)
				}
				break
			}
		}
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

		url := appConfig.Server.Scheme + "://" + appConfig.Server.Host
		if (appConfig.Server.Scheme == "http" && appConfig.Server.Port != 80) ||
			(appConfig.Server.Scheme == "https" && appConfig.Server.Port != 443) {
			url += ":" + port
		}

		logger.Log.Info("REST API running", "port", port, "url", url, "version", Version)
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

func runPluginMigrations(ctx context.Context, db database.Database, pluginRegistry *plugin.PluginRegistry, pluginConfigs []config.PluginConfig) error {
	configMap := make(map[string]config.PluginConfig)
	for _, cfg := range pluginConfigs {
		configMap[cfg.Name] = cfg
	}

	var sources []migrations.MigrationSource
	pluginToSourceName := make(map[string]string)

	for _, p := range pluginRegistry.GetAll() {
		migProvider, hasMigrations := p.(plugin.MigrationProvider)
		if !hasMigrations {
			continue
		}

		pluginCfg, exists := configMap[p.Name()]
		if !exists {
			continue
		}

		migrationsConfig, hasMigConfig := pluginCfg.Config["migrations"].(map[string]interface{})
		if !hasMigConfig {
			continue
		}

		enabled, ok := migrationsConfig["enabled"].(bool)
		if !ok || !enabled {
			continue
		}

		source := migProvider.MigrationSource()
		if migSource, ok := source.(migrations.MigrationSource); ok {
			sources = append(sources, migSource)
			sourceName := migSource.Name()
			pluginToSourceName[p.Name()] = sourceName

			logger.Log.Info("Registered migrations", "plugin", p.Name(), "source", sourceName, "dependencies", migProvider.MigrationDependencies())
		}
	}

	if len(sources) == 0 {
		logger.Log.Info("No plugin migrations to run")
		return nil
	}

	migrator := migrations.NewMigrator(db, sources...)

	for _, p := range pluginRegistry.GetAll() {
		migProvider, ok := p.(plugin.MigrationProvider)
		if !ok {
			continue
		}

		sourceName, exists := pluginToSourceName[p.Name()]
		if !exists {
			continue
		}

		deps := migProvider.MigrationDependencies()
		if len(deps) == 0 {
			continue
		}

		mappedDeps := make([]string, 0, len(deps))
		for _, depPluginName := range deps {
			if depSourceName, ok := pluginToSourceName[depPluginName]; ok {
				mappedDeps = append(mappedDeps, depSourceName)
			}
		}

		if len(mappedDeps) > 0 {
			migrator.SetSourceDependencies(sourceName, mappedDeps)
		}
	}

	logger.Log.Info("Running migrations", "sources", len(sources))

	if err := migrator.Up(ctx); err != nil {
		return err
	}

	logger.Log.Info("Migrations completed successfully")
	return nil
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
