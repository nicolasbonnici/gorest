package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicolasbonnici/gorest/internal"
)

var (
	autoYes = flag.Bool("y", false, "Automatic yes to prompts (non-interactive mode)")
)

func main() {
	flag.Parse()
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

	projectRoot, err := internal.FindProjectRoot()
	if err != nil {
		log.Fatalf("❌ Failed to find project root: %v", err)
	}

	modelsDir := filepath.Join(projectRoot, "internal", "api", "models")
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

	resourcesDir := filepath.Join(projectRoot, "internal", "api", "resources")
	existingResources := make(map[string]bool)
	resourcesToGenerate := []string{}

	if _, err := os.Stat(resourcesDir); err == nil {
		existingFiles, _ := os.ReadDir(resourcesDir)
		for _, file := range existingFiles {
			if strings.HasSuffix(file.Name(), ".go") {
				existingResources[file.Name()] = true
			}
		}
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".go") && file.Name() != "model.go" {
			resourceName := strings.TrimSuffix(file.Name(), ".go")
			resourcesToGenerate = append(resourcesToGenerate, resourceName)
		}
	}

	resourcesToSkip := make(map[string]bool)
	resourceAuthConfig := make(map[string]bool)

	if *autoYes {
		log.Println("🤖 Non-interactive mode enabled (-y flag)")
		for _, resourceName := range resourcesToGenerate {
			resourceFile := resourceName + ".go"
			if existingResources[resourceFile] {
				log.Printf("✅ Auto-overriding: %s", resourceFile)
			}
			resourceAuthConfig[resourceName] = true
			log.Printf("✅ Auto-enabling authentication for: %s", resourceName)
		}
	} else {
		for _, resourceName := range resourcesToGenerate {
			resourceFile := resourceName + ".go"

			if existingResources[resourceFile] {
				if !promptYesNo(fmt.Sprintf("⚠️  Resource '%s' already exists. Override?", resourceFile), true) {
					resourcesToSkip[resourceName] = true
					log.Printf("⏭️  Skipping %s", resourceFile)
					continue
				}
			}

			requireAuth := promptYesNo(fmt.Sprintf("🔐 Require authentication for '%s' endpoints (GET/POST/PUT/DELETE)?", resourceName), true)
			resourceAuthConfig[resourceName] = requireAuth
		}
	}

	if len(resourcesToSkip) == len(resourcesToGenerate) && len(resourcesToGenerate) > 0 {
		log.Println("❌ All resources skipped. Nothing to generate.")
		return
	}

	authConfigPath := filepath.Join(projectRoot, "config", "auth.json")
	authCfg := internal.NoAuthConfig()

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

	log.Println("🔄 Generating API resources from models...")
	tables := internal.LoadSchema(db)

	tableToResourceMap := make(map[string]string)
	for tableName := range tables {
		singularName := internal.SingularizeExported(tableName)
		resourceName := strings.ToLower(singularName)
		tableToResourceMap[tableName] = resourceName
	}

	for _, resourceName := range tableToResourceMap {
		if requireAuth, exists := resourceAuthConfig[resourceName]; exists && requireAuth {
			pluralResourceName := pluralizeResourceName(resourceName)
			authCfg.SetResourceAuth(pluralResourceName, []string{"GET", "POST", "PUT", "DELETE"})
			log.Printf("🔒 Authentication enabled for resource: %s", resourceName)
		} else {
			log.Printf("🔓 No authentication for resource: %s", resourceName)
		}
	}

	if err := internal.SaveAuthConfig(authCfg, authConfigPath); err != nil {
		log.Printf("⚠️  Warning: Failed to save auth config: %v", err)
	} else {
		log.Printf("💾 Auth configuration saved to: %s", authConfigPath)
	}

	internal.GenerateAPIWithSkip(db, tables, authCfg, resourcesToSkip)
	log.Println("✅ Resource generation completed successfully")
}

func pluralizeResourceName(word string) string {
	if strings.HasSuffix(word, "y") && len(word) > 1 && !isVowel(word[len(word)-2]) {
		return word[:len(word)-1] + "ies"
	}
	if strings.HasSuffix(word, "fe") {
		return word[:len(word)-2] + "ves"
	}
	if strings.HasSuffix(word, "f") {
		return word[:len(word)-1] + "ves"
	}
	if strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") ||
		strings.HasSuffix(word, "z") || strings.HasSuffix(word, "ch") ||
		strings.HasSuffix(word, "sh") {
		return word + "es"
	}
	return word + "s"
}

func isVowel(c byte) bool {
	return c == 'a' || c == 'e' || c == 'i' || c == 'o' || c == 'u'
}

func promptYesNo(prompt string, defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)

	defaultText := "Y/n"
	if !defaultYes {
		defaultText = "y/N"
	}

	fmt.Printf("%s [%s]: ", prompt, defaultText)
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("❌ Failed to read input: %v", err)
	}

	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		return defaultYes
	}

	return input == "y" || input == "yes"
}
