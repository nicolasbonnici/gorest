package main

import (
	"context"
	"log"
	"net/url"
	"os"

	"github.com/joho/godotenv"
	"github.com/nicolasbonnici/gorest/internal"
	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/mysql"
	_ "github.com/nicolasbonnici/gorest/database/postgres"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("❌ DATABASE_URL environment variable is required")
	}

	if _, err := url.Parse(dbURL); err != nil {
		log.Fatalf("❌ Invalid DATABASE_URL format: %v", err)
	}

	db, err := database.Open("", dbURL)
	if err != nil {
		log.Fatalf("❌ DB connection failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("❌ DB ping failed: %v", err)
	}
	log.Printf("✅ Database connection verified (%s)", db.DriverName())

	log.Println("🔄 Generating OpenAPI schema...")
	tables := internal.LoadSchema(db)
	internal.GenerateOpenAPI(tables)
	log.Println("✅ OpenAPI generation completed successfully")
}
