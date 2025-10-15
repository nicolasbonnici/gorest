package main

import (
	"context"
	"log"
	"net/url"
	"os"
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

	if !strings.HasPrefix(dbURL, "postgres://") && !strings.HasPrefix(dbURL, "postgresql://") {
		log.Fatal("❌ DATABASE_URL must be a valid PostgreSQL connection string (postgres:// or postgresql://)")
	}

	if _, err := url.Parse(dbURL); err != nil {
		log.Fatalf("❌ Invalid DATABASE_URL format: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("❌ Invalid DATABASE_URL: %v", err)
	}

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

	log.Println("🔄 Generating models from database schema...")
	tables := internal.LoadSchema(db)
	internal.GenerateStructs(tables)
	log.Println("✅ Model generation completed successfully")
}
