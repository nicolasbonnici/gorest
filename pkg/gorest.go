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
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nicolasbonnici/gorest/internal"
	"github.com/nicolasbonnici/gorest/internal/api"
	"github.com/nicolasbonnici/gorest/internal/middleware"
)

type Config struct {
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

	db, err := pgxpool.New(context.Background(), cfg.DBUrl)
	if err != nil {
		log.Fatalf("❌ DB connection failed: %v", err)
	}
	defer db.Close()

	tables := internal.LoadSchema(db)

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
