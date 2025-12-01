package main

import (
	"fmt"
	"os"

	"github.com/nicolasbonnici/gorest/config"
	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/mysql"
	_ "github.com/nicolasbonnici/gorest/database/postgres"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
	"github.com/nicolasbonnici/gorest/plugin"
	_ "github.com/nicolasbonnici/gorest/plugins/benchmark"
	_ "github.com/nicolasbonnici/gorest/plugins/codegen"
	"github.com/nicolasbonnici/gorest/pluginloader"
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

	// Load all registered command plugins via auto-discovery
	plugins, err := pluginloader.LoadAllCommandPlugins(db, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load plugins: %v\n", err)
		os.Exit(1)
	}

	// Aggregate all commands from all plugins
	var commands []plugin.Command
	for _, p := range plugins {
		commandProvider, ok := p.(plugin.CommandProvider)
		if ok {
			commands = append(commands, commandProvider.Commands()...)
		}
	}

	commandName := os.Args[1]

	// Find and execute the requested command
	for _, cmd := range commands {
		if cmd.Name() == commandName {
			result := executeCommand(cmd, os.Args[2:])
			if !result.Success {
				fmt.Fprintf(os.Stderr, "Error: %v\n", result.Error)
				os.Exit(1)
			}
			fmt.Println(result.Message)
			os.Exit(0)
		}
	}

	// Command not found
	fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", commandName)
	printUsage()
	os.Exit(1)
}

func executeCommand(cmd plugin.Command, args []string) *plugin.CommandResult {
	ctx := &plugin.CommandContext{
		Args: args,
		ProgressCallback: func(message string) {
			fmt.Printf("  → %s\n", message)
		},
	}

	return cmd.Run(ctx)
}

func printUsage() {
	fmt.Println("GoREST Code Generator")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gorest-codegen <command>")
	fmt.Println()
	fmt.Println("Available Commands:")
	fmt.Println("  models      Generate model structs from database schema")
	fmt.Println("  resources   Generate REST API resources and DTOs from models")
	fmt.Println("  openapi     Generate OpenAPI schema file")
	fmt.Println("  all         Run all code generation steps")
	fmt.Println("  benchmark   Run API performance benchmarks")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  gorest-codegen models")
	fmt.Println("  gorest-codegen resources")
	fmt.Println("  gorest-codegen all")
	fmt.Println("  gorest-codegen benchmark")
}
