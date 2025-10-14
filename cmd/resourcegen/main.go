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

	// Collect existing resources and models to be generated
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

	// Identify which resources will be generated from models
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".go") && file.Name() != "model.go" {
			resourceName := strings.TrimSuffix(file.Name(), ".go")
			resourcesToGenerate = append(resourcesToGenerate, resourceName)
		}
	}

	// Prompt for each existing resource and authentication requirements
	resourcesToSkip := make(map[string]bool)
	resourceAuthConfig := make(map[string]bool) // Track which resources need auth

	if *autoYes {
		log.Println("🤖 Non-interactive mode enabled (-y flag)")
		// In auto-yes mode, override all and enable auth for all resources
		for _, resourceName := range resourcesToGenerate {
			resourceFile := resourceName + ".go"
			if existingResources[resourceFile] {
				log.Printf("✅ Auto-overriding: %s", resourceFile)
			}
			resourceAuthConfig[resourceName] = true
			log.Printf("✅ Auto-enabling authentication for: %s", resourceName)
		}
	} else {
		// Interactive mode
		for _, resourceName := range resourcesToGenerate {
			resourceFile := resourceName + ".go"

			// Check if resource exists and prompt for override
			if existingResources[resourceFile] {
				if !promptYesNo(fmt.Sprintf("⚠️  Resource '%s' already exists. Override?", resourceFile), true) {
					resourcesToSkip[resourceName] = true
					log.Printf("⏭️  Skipping %s", resourceFile)
					continue
				}
			}

			// Prompt for authentication for this specific resource
			requireAuth := promptYesNo(fmt.Sprintf("🔐 Require authentication for '%s' endpoints (GET/POST/PUT/DELETE)?", resourceName), true)
			resourceAuthConfig[resourceName] = requireAuth
		}
	}

	// If all resources are skipped, exit
	if len(resourcesToSkip) == len(resourcesToGenerate) && len(resourcesToGenerate) > 0 {
		log.Println("❌ All resources skipped. Nothing to generate.")
		return
	}

	// Create auth config based on per-resource choices
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

	// Build a map of table names to their pluralized resource names for matching
	tableToResourceMap := make(map[string]string)
	for tableName := range tables {
		// Singularize and lowercase to match resource names
		singularName := internal.SingularizeExported(tableName)
		resourceName := strings.ToLower(singularName)
		tableToResourceMap[tableName] = resourceName
	}

	// Apply auth config based on per-resource choices
	for _, resourceName := range tableToResourceMap {
		if requireAuth, exists := resourceAuthConfig[resourceName]; exists && requireAuth {
			// Include all HTTP methods: GET, POST, PUT, DELETE
			// Use the pluralized resource name for the auth config (e.g., "todos" not "todo")
			pluralResourceName := pluralizeResourceName(resourceName)
			authCfg.SetResourceAuth(pluralResourceName, []string{"GET", "POST", "PUT", "DELETE"})
			log.Printf("🔒 Authentication enabled for resource: %s", resourceName)
		} else {
			log.Printf("🔓 No authentication for resource: %s", resourceName)
		}
	}

	// Save auth config
	if err := internal.SaveAuthConfig(authCfg, authConfigPath); err != nil {
		log.Printf("⚠️  Warning: Failed to save auth config: %v", err)
	} else {
		log.Printf("💾 Auth configuration saved to: %s", authConfigPath)
	}

	internal.GenerateAPIWithSkip(db, tables, authCfg, resourcesToSkip)
	log.Println("✅ Resource generation completed successfully")
}

// pluralizeResourceName converts a singular resource name to plural
func pluralizeResourceName(word string) string {
	// Handle common singular patterns
	if strings.HasSuffix(word, "y") && len(word) > 1 && !isVowel(word[len(word)-2]) {
		// category -> categories, story -> stories
		return word[:len(word)-1] + "ies"
	}
	if strings.HasSuffix(word, "fe") {
		// knife -> knives, wolf -> wolves
		return word[:len(word)-2] + "ves"
	}
	if strings.HasSuffix(word, "f") {
		// shelf -> shelves
		return word[:len(word)-1] + "ves"
	}
	if strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") ||
		strings.HasSuffix(word, "z") || strings.HasSuffix(word, "ch") ||
		strings.HasSuffix(word, "sh") {
		// class -> classes, box -> boxes, church -> churches, dish -> dishes
		return word + "es"
	}
	// Default: add 's'
	return word + "s"
}

func isVowel(c byte) bool {
	return c == 'a' || c == 'e' || c == 'i' || c == 'o' || c == 'u'
}

// promptYesNo prompts the user for a yes/no answer with a default value
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
