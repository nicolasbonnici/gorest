package main

import (
	"context"
	"log"
	"net/url"

	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/mysql"
	_ "github.com/nicolasbonnici/gorest/database/postgres"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
	"github.com/nicolasbonnici/gorest/generator"
)

func main() {
	cfg, err := generator.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	if _, err := url.Parse(cfg.Database.URL); err != nil {
		log.Fatalf("❌ Invalid DATABASE_URL format: %v", err)
	}

	db, err := database.Open("", cfg.Database.URL)
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
	tables := generator.LoadSchema(db)
	generator.GenerateOpenAPI(tables)
	log.Println("✅ OpenAPI generation completed successfully")
}
