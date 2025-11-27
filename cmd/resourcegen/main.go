package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/nicolasbonnici/gorest/config"
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

	modelsDir, err := generator.GetModelsPath(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to get models path: %v", err)
	}
	if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
		log.Fatal("❌ Models directory not found. Run 'make modelgen' first to generate models.")
	}

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

	log.Println("🔄 Generating API resources from models...")
	log.Printf("   Auth enabled: %v", cfg.Generate.Auth.Enabled)
	if cfg.Generate.Auth.Enabled {
		log.Println("   Endpoints requiring auth:")
		if cfg.Generate.Auth.Endpoints.List {
			log.Println("     - List (GET)")
		}
		if cfg.Generate.Auth.Endpoints.Get {
			log.Println("     - Get (GET /:id)")
		}
		if cfg.Generate.Auth.Endpoints.Create {
			log.Println("     - Create (POST)")
		}
		if cfg.Generate.Auth.Endpoints.Update {
			log.Println("     - Update (PUT /:id)")
		}
		if cfg.Generate.Auth.Endpoints.Delete {
			log.Println("     - Delete (DELETE /:id)")
		}
	}

	tables := generator.LoadSchema(db)
	authCfg := createAuthConfigFromUnified(cfg, tables)
	generator.GenerateAPI(authCfg)
	log.Println("✅ Resource generation completed successfully")
}

func createAuthConfigFromUnified(cfg *config.Config, tables map[string]generator.TableSchema) *generator.AuthConfig {
	authCfg := &generator.AuthConfig{
		RequireAuth: make(map[string][]string),
	}

	if !cfg.Generate.Auth.Enabled {
		return authCfg
	}

	var methods []string
	if cfg.Generate.Auth.Endpoints.List || cfg.Generate.Auth.Endpoints.Get {
		methods = append(methods, "GET")
	}
	if cfg.Generate.Auth.Endpoints.Create {
		methods = append(methods, "POST")
	}
	if cfg.Generate.Auth.Endpoints.Update {
		methods = append(methods, "PUT")
	}
	if cfg.Generate.Auth.Endpoints.Delete {
		methods = append(methods, "DELETE")
	}

	for tableName := range tables {
		authCfg.RequireAuth[tableName] = methods
		plural := generator.Pluralize(tableName)
		if plural != tableName {
			authCfg.RequireAuth[plural] = methods
		}
	}

	return authCfg
}
