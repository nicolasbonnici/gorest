package gorest

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nicolasbonnici/gorest/internal"
	"github.com/nicolasbonnici/gorest/internal/api"
)

type Config struct {
	DBUrl     string
	JWTSecret string
	Port      string
}

func Start(cfg Config) {
	// Validate that generated files exist before starting
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

	db, err := pgxpool.New(context.Background(), cfg.DBUrl)
	if err != nil {
		log.Fatalf("❌ DB connection failed: %v", err)
	}
	defer db.Close()

	tables := internal.LoadSchema(db)

	app := fiber.New()

	internal.SetupAuth(app, cfg.JWTSecret)

	// Register routes dynamically
	api.RegisterGeneratedRoutes(app, db, tables)

	internal.SetupOpenAPI(app, tables)

	log.Printf("🚀 REST API running at http://localhost:%s", cfg.Port)
	app.Listen(":" + cfg.Port)
}
