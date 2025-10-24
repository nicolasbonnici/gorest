package gorest

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/nicolasbonnici/gorest/internal"
	"github.com/nicolasbonnici/gorest/internal/api"
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
	Port      string
}

func Start(cfg Config) {
	projectRoot, err := internal.FindProjectRoot()
	if err != nil {
		log.Fatalf("❌ Failed to find project root: %v", err)
	}

	modelsDir := filepath.Join(projectRoot, "internal", "api", "models")
	if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
		log.Fatal("❌ Models not found. Run 'make modelgen' first to generate models from database schema.")
	}

	resourcesDir := filepath.Join(projectRoot, "internal", "api", "resources")
	if _, err := os.Stat(resourcesDir); os.IsNotExist(err) {
		log.Fatal("❌ Resources not found. Run 'make resourcegen' first to generate API resources.")
	}

	openapiDir := filepath.Join(projectRoot, "internal", "api", "openapi")
	if _, err := os.Stat(openapiDir); os.IsNotExist(err) {
		log.Fatal("❌ OpenAPI schema not found. Run 'make openapigen' first to generate OpenAPI schema.")
	}

	db, err := database.Open(cfg.DBDriver, cfg.DBUrl)
	if err != nil {
		log.Fatalf("❌ DB connection failed: %v", err)
	}
	defer db.Close()

	schemaSlice, err := db.Introspector().LoadSchema(context.Background())
	if err != nil {
		log.Fatalf("❌ Failed to load schema: %v", err)
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

	app.Use(middleware.HTTPLogger())

	internal.SetupHealthCheck(app, db)
	internal.SetupAuth(app, db, cfg.JWTSecret)
	api.RegisterGeneratedRoutes(app, db, tables, cfg.JWTSecret)

	internal.SetupOpenAPI(app, tables)

	// Channel to listen for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Printf("🚀 REST API running at http://localhost:%s", cfg.Port)
		log.Printf("📊 Health check available at http://localhost:%s/health", cfg.Port)
		if err := app.Listen(":" + cfg.Port); err != nil {
			log.Fatalf("❌ Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	log.Println("🛑 Shutting down server gracefully...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("⚠️  Server forced to shutdown: %v", err)
	}

	// Close database connection
	db.Close()
	log.Println("✅ Server shutdown complete")
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
