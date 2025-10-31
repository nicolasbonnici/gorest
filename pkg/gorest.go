package gorest

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/nicolasbonnici/gorest/internal"
	"github.com/nicolasbonnici/gorest/internal/api"
	"github.com/nicolasbonnici/gorest/internal/logger"
	"github.com/nicolasbonnici/gorest/internal/middleware"
	"github.com/nicolasbonnici/gorest/pkg/database"
	_ "github.com/nicolasbonnici/gorest/pkg/database/mysql"
	_ "github.com/nicolasbonnici/gorest/pkg/database/postgres"
	_ "github.com/nicolasbonnici/gorest/pkg/database/sqlite"
)

type Config struct {
	DBDriver  string
	DBUrl     string
	JWTSecret string
	JWTTTL    int
	Port      string
}

func validateConfig(cfg Config) {
	if cfg.DBUrl == "" {
		logger.Log.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	if cfg.JWTSecret == "" {
		logger.Log.Error("JWT_SECRET is required")
		os.Exit(1)
	}

	if len(cfg.JWTSecret) < 32 {
		logger.Log.Error("JWT_SECRET must be at least 32 characters long for security", "current_length", len(cfg.JWTSecret))
		os.Exit(1)
	}

	if cfg.JWTTTL <= 0 {
		logger.Log.Error("JWT_TTL must be a positive integer (seconds)", "current_value", cfg.JWTTTL)
		os.Exit(1)
	}

	if cfg.Port == "" {
		logger.Log.Error("PORT is required")
		os.Exit(1)
	}
}

func Start(cfg Config) {
	validateConfig(cfg)
	projectRoot, err := internal.FindProjectRoot()
	if err != nil {
		logger.Log.Error("Failed to find project root", "error", err)
		os.Exit(1)
	}

	modelsDir := filepath.Join(projectRoot, "internal", "models")
	if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
		logger.Log.Error("Models not found. Run 'make modelgen' first to generate models from database schema")
		os.Exit(1)
	}

	resourcesDir := filepath.Join(projectRoot, "internal", "api", "resources")
	if _, err := os.Stat(resourcesDir); os.IsNotExist(err) {
		logger.Log.Error("Resources not found. Run 'make resourcegen' first to generate API resources")
		os.Exit(1)
	}

	openapiDir := filepath.Join(projectRoot, "internal", "openapi")
	if _, err := os.Stat(openapiDir); os.IsNotExist(err) {
		logger.Log.Error("OpenAPI schema not found. Run 'make openapigen' first to generate OpenAPI schema")
		os.Exit(1)
	}

	db, err := database.Open(cfg.DBDriver, cfg.DBUrl)
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

	tables := make(map[string]internal.TableSchema)
	for _, t := range schemaSlice {
		tables[t.TableName] = internal.TableSchema{
			TableName: t.TableName,
			Columns:   convertColumns(t.Columns),
			Relations: convertRelations(t.Relations),
		}
	}

	app := fiber.New()

	app.Use(requestid.New())
	app.Use(middleware.HTTPLogger())

	internal.SetupOpenAPIUI(app)
	internal.SetupHealthCheck(app, db)
	internal.SetupAuth(app, db, cfg.JWTSecret, cfg.JWTTTL)
	api.RegisterGeneratedRoutes(app, db, tables, cfg.JWTSecret)

	internal.SetupOpenAPI(app, tables)

	// Channel to listen for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		logger.Log.Info("REST API running", "port", cfg.Port, "url", "http://localhost:"+cfg.Port)
		logger.Log.Info("Health check available", "url", "http://localhost:"+cfg.Port+"/health")
		if err := app.Listen(":" + cfg.Port); err != nil {
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

func convertColumns(dbCols []database.Column) []internal.Column {
	cols := make([]internal.Column, len(dbCols))
	for i, c := range dbCols {
		cols[i] = internal.Column{
			Name:       c.Name,
			Type:       c.Type,
			IsNullable: c.IsNullable,
		}
	}
	return cols
}

func convertRelations(dbRels []database.Relation) []internal.Relation {
	rels := make([]internal.Relation, len(dbRels))
	for i, r := range dbRels {
		rels[i] = internal.Relation{
			ChildTable:   r.ChildTable,
			ChildColumn:  r.ChildColumn,
			ParentTable:  r.ParentTable,
			ParentColumn: r.ParentColumn,
		}
	}
	return rels
}
