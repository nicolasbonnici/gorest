package main

import (
	"fmt"
	"os"

	"github.com/nicolasbonnici/gorest/config"
	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/mysql"
	_ "github.com/nicolasbonnici/gorest/database/postgres"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
	"github.com/nicolasbonnici/gorest/migrations"
	"github.com/nicolasbonnici/gorest/migrations/cli"
)

func main() {
	// Parse global flags before the subcommand.
	// Flags can appear anywhere in the argument list.
	configDir := "."
	dsn := os.Getenv("DATABASE_URL")
	migrationDir := "."

	args := os.Args[1:]
	remaining := args[:0]

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--config":
			if i+1 < len(args) {
				i++
				configDir = args[i]
			}
		case "--dsn":
			if i+1 < len(args) {
				i++
				dsn = args[i]
			}
		case "--migration-dir":
			if i+1 < len(args) {
				i++
				migrationDir = args[i]
			}
		default:
			remaining = append(remaining, args[i])
		}
	}

	// Commands that don't need a DB connection.
	subCmd := ""
	if len(remaining) > 0 {
		subCmd = remaining[0]
	}
	if subCmd == "help" || subCmd == "--help" || subCmd == "-h" || subCmd == "" {
		runner := cli.New(nil, nil, migrationDir)
		_ = runner.Run(remaining)
		return
	}
	if subCmd == "new" {
		runner := cli.New(nil, nil, migrationDir)
		if err := runner.Run(remaining); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Resolve DSN: explicit --dsn beats gorest.yaml
	if dsn == "" {
		cfg, err := config.Load(configDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading config from %q: %v\n", configDir, err)
			os.Exit(1)
		}
		dsn = cfg.Database.URL
	}

	if dsn == "" {
		fmt.Fprintln(os.Stderr, "no database URL found — use --dsn or set DATABASE_URL, or point --config to a directory with gorest.yaml")
		os.Exit(1)
	}

	db, err := database.Open("", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	m := migrations.NewMigrator(db)
	runner := cli.New(m, db, migrationDir)

	if err := runner.Run(remaining); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
