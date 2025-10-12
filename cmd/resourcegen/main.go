package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicolasbonnici/gorest/internal"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("❌ DATABASE_URL environment variable is required")
	}

	// Validate URL format
	if !strings.HasPrefix(dbURL, "postgres://") && !strings.HasPrefix(dbURL, "postgresql://") {
		log.Fatal("❌ DATABASE_URL must be a valid PostgreSQL connection string (postgres:// or postgresql://)")
	}

	// Validate URL is well-formed
	if _, err := url.Parse(dbURL); err != nil {
		log.Fatalf("❌ Invalid DATABASE_URL format: %v", err)
	}

	// Validate that models exist
	projectRoot, err := internal.FindProjectRoot()
	if err != nil {
		log.Fatalf("❌ Failed to find project root: %v", err)
	}

	modelsDir := filepath.Join(projectRoot, "internal", "api", "models")
	if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
		log.Fatal("❌ Models directory not found. Run 'make modelgen' first to generate models.")
	}

	// Check if models directory has Go files
	files, err := os.ReadDir(modelsDir)
	if err != nil {
		log.Fatalf("❌ Failed to read models directory: %v", err)
	}

	hasModels := false
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".go") && file.Name() != "model.go" {
			hasModels = true
			break
		}
	}

	if !hasModels {
		log.Fatal("❌ No model files found. Run 'make modelgen' first to generate models.")
	}

	// Create context with timeout for connection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Parse config and set connection pool settings
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("❌ Invalid DATABASE_URL: %v", err)
	}

	// Configure for code generation workload
	config.MaxConns = 5
	config.MinConns = 1
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("❌ DB connection failed: %v", err)
	}
	defer func() {
		db.Close()
	}()

	log.Println("🔄 Generating API resources from models...")
	tables := internal.LoadSchema(db)
	internal.GenerateAPI(db, tables)
	log.Println("✅ Resource generation completed successfully")
}
