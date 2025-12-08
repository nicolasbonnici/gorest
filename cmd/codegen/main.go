package main

import (
	"fmt"
	"os"

	"github.com/nicolasbonnici/gorest/config"
	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/mysql"
	_ "github.com/nicolasbonnici/gorest/database/postgres"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
	"github.com/nicolasbonnici/gorest/codegen"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.Load(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Connect to database
	db, err := database.Open("", cfg.Database.URL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Database connection failed: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	commandName := os.Args[1]

	// Route to command handler
	switch commandName {
	case "models":
		if err := modelsCommand(db); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "resources":
		if err := resourcesCommand(db); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "openapi":
		if err := openapiCommand(db); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "all":
		if err := allCommand(db); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", commandName)
		printUsage()
		os.Exit(1)
	}
}

func modelsCommand(db database.Database) error {
	progress("Loading database schema...")
	tables := codegen.LoadSchema(db)

	progress("Generating model structs...")
	codegen.GenerateStructs(tables)

	progress("Models generated successfully")
	fmt.Println("Model generation completed successfully")
	return nil
}

func resourcesCommand(db database.Database) error {
	progress("Loading configuration...")
	cfg, err := config.Load(".")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	progress("Building authentication configuration...")
	authCfg := codegen.GetAuthConfigFromConfig(cfg)

	progress("Generating API resources...")
	codegen.GenerateAPI(authCfg)

	progress("Resources generated successfully")
	fmt.Println("Resource generation completed successfully")
	return nil
}

func openapiCommand(db database.Database) error {
	progress("Generating OpenAPI schema...")
	tables := codegen.LoadSchema(db)
	codegen.GenerateOpenAPI(tables)

	progress("OpenAPI schema generated successfully")
	fmt.Println("OpenAPI generation completed successfully")
	return nil
}

func allCommand(db database.Database) error {
	progress("Running: models")
	if err := modelsCommand(db); err != nil {
		return err
	}

	progress("Running: resources")
	if err := resourcesCommand(db); err != nil {
		return err
	}

	progress("Running: openapi")
	if err := openapiCommand(db); err != nil {
		return err
	}

	fmt.Println("All code generation completed successfully")
	return nil
}

func progress(message string) {
	fmt.Printf("  → %s\n", message)
}

func printUsage() {
	fmt.Println("GoREST Code Generator")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  codegen <command>")
	fmt.Println()
	fmt.Println("Available Commands:")
	fmt.Println("  models      Generate model structs from database schema")
	fmt.Println("  resources   Generate REST API resources and DTOs from models")
	fmt.Println("  openapi     Generate OpenAPI schema file")
	fmt.Println("  all         Run all code generation steps")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  codegen models")
	fmt.Println("  codegen resources")
	fmt.Println("  codegen all")
}
