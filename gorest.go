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
	"github.com/nicolasbonnici/gorest/generator"
	"github.com/nicolasbonnici/gorest/logger"
	"github.com/nicolasbonnici/gorest/plugin"
	"github.com/nicolasbonnici/gorest/pluginloader"
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

	checkGeneratedCode(appConfig)

	db, err := database.Open("", appConfig.Database.URL)
	if err != nil {
		logger.Log.Error("DB connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	schemaSlice, err := db.Introspector().LoadSchema(context.Background())
	if err != nil {
		logger.Log.Error("Failed to load schema", "error", err)
		os.Exit(1)
	}

	tables := make(map[string]generator.TableSchema)
	for _, t := range schemaSlice {
		tables[t.TableName] = generator.TableSchema{
			TableName: t.TableName,
			Columns:   convertColumns(t.Columns),
			Relations: convertRelations(t.Relations),
		}
	}

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

	enrichedConfigs := pluginloader.InjectSharedConfig(appConfig.Plugins, db)

	// Load plugins (plugins are NOT automatically applied - user must use app.Use() or fiber groups)
	pluginRegistry, err := pluginloader.LoadPlugins(enrichedConfigs, Version)
	if err != nil {
		logger.Log.Error("Failed to load plugins", "error", err)
		os.Exit(1)
	}

	SetupOpenAPIUI(app)

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

	generator.SetupOpenAPI(app, tables, appConfig.Pagination.DefaultLimit, appConfig.Pagination.MaxLimit)

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

func checkGeneratedCode(cfg *config.Config) {
	projectRoot, err := FindProjectRoot()
	if err != nil {
		logger.Log.Error("Failed to find project root", "error", err)
		os.Exit(1)
	}

	modelsDir := filepath.Join(projectRoot, cfg.Generate.Output.Models)
	if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
		logger.Log.Error("Models not found. Run model generation first to generate models from database schema",
			"expected_path", modelsDir)
		os.Exit(1)
	}

	resourcesDir := filepath.Join(projectRoot, cfg.Generate.Output.Resources)
	if _, err := os.Stat(resourcesDir); os.IsNotExist(err) {
		logger.Log.Error("Resources not found. Run resource generation first to generate API resources",
			"expected_path", resourcesDir)
		os.Exit(1)
	}

	openapiDir := filepath.Join(projectRoot, cfg.Generate.Output.OpenAPI)
	if _, err := os.Stat(openapiDir); os.IsNotExist(err) {
		logger.Log.Error("OpenAPI schema not found. Run OpenAPI generation first to generate OpenAPI schema",
			"expected_path", openapiDir)
		os.Exit(1)
	}
}

func convertColumns(dbCols []database.Column) []generator.Column {
	cols := make([]generator.Column, len(dbCols))
	for i, c := range dbCols {
		cols[i] = generator.Column{
			Name:       c.Name,
			Type:       c.Type,
			IsNullable: c.IsNullable,
		}
	}
	return cols
}

func convertRelations(dbRels []database.Relation) []generator.Relation {
	rels := make([]generator.Relation, len(dbRels))
	for i, r := range dbRels {
		rels[i] = generator.Relation{
			ChildTable:   r.ChildTable,
			ChildColumn:  r.ChildColumn,
			ParentTable:  r.ParentTable,
			ParentColumn: r.ParentColumn,
		}
	}
	return rels
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

func SetupOpenAPIUI(app *fiber.App) {
	app.Get("/openapi", func(c *fiber.Ctx) error {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>GoREST API Documentation</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body {
            margin: 0;
            padding: 0;
        }
    </style>
</head>
<body>
    <script id="api-reference" data-url="/openapi.json"></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`
		c.Set("Content-Type", "text/html")
		return c.SendString(html)
	})
}
