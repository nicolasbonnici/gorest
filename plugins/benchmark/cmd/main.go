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
	"github.com/nicolasbonnici/gorest/pluginloader"
)

func main() {
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

	// Load benchmark plugin via auto-discovery
	plugins, err := pluginloader.LoadAllCommandPlugins(db, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load plugins: %v\n", err)
		os.Exit(1)
	}

	// Find benchmark command
	var benchmarkCmd plugin.Command
	for _, p := range plugins {
		if p.Name() == "benchmark" {
			commandProvider, ok := p.(plugin.CommandProvider)
			if ok {
				commands := commandProvider.Commands()
				if len(commands) > 0 {
					benchmarkCmd = commands[0]
					break
				}
			}
		}
	}

	if benchmarkCmd == nil {
		fmt.Fprintf(os.Stderr, "Benchmark plugin not found\n")
		os.Exit(1)
	}

	// Execute benchmark command
	ctx := &plugin.CommandContext{
		Args: os.Args[1:],
		ProgressCallback: func(message string) {
			fmt.Printf("  → %s\n", message)
		},
	}

	result := benchmarkCmd.Run(ctx)
	if !result.Success {
		fmt.Fprintf(os.Stderr, "Error: %v\n", result.Error)
		os.Exit(1)
	}

	fmt.Println(result.Message)
}
