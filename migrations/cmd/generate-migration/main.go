package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/nicolasbonnici/gorest/migrations"
)

const goMigrationTemplate = `package migrations

import (
	"context"

	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/migrations"
)

func init() {
	// Register migration with your migration source
	// Example: yourMigrations = append(yourMigrations, Migration_{{.Version}}_{{.SafeName}}())
}

// Migration_{{.Version}}_{{.SafeName}} creates the {{.Description}} migration
func Migration_{{.Version}}_{{.SafeName}}() migrations.Migration {
	return migrations.NewGoMigration("{{.Version}}", "{{.Name}}").
		Up(func(ctx context.Context, db database.Database) error {
			// TODO: Implement up migration
			// Example using helpers:
			// return migrations.CreateTableIfNotExists(ctx, db, "table_name", "id TEXT PRIMARY KEY")

			// Example using SQL with dialect support:
			// return migrations.SQL(ctx, db, migrations.DialectSQL{
			//     Postgres: "CREATE TABLE...",
			//     MySQL:    "CREATE TABLE...",
			//     SQLite:   "CREATE TABLE...",
			// })

			return nil
		}).
		Down(func(ctx context.Context, db database.Database) error {
			// TODO: Implement down migration
			// Example:
			// return migrations.DropTableIfExists(ctx, db, "table_name")

			return nil
		}).
		Build()
}
`

type MigrationData struct {
	Version     string
	Name        string
	SafeName    string
	Description string
}

func main() {
	var (
		name        string
		outputDir   string
		showHelp    bool
		listHelpers bool
	)

	flag.StringVar(&name, "name", "", "Migration name (e.g., create_users_table)")
	flag.StringVar(&outputDir, "dir", ".", "Output directory for migration file")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.BoolVar(&listHelpers, "helpers", false, "List available migration helpers")
	flag.Parse()

	if showHelp {
		printHelp()
		return
	}

	if listHelpers {
		printHelpers()
		return
	}

	if name == "" {
		fmt.Println("Error: migration name is required")
		fmt.Println()
		printHelp()
		os.Exit(1)
	}

	// Generate timestamp with milliseconds
	timestamp := migrations.GenerateTimestamp()

	// Clean the name
	cleanName := strings.ToLower(name)
	cleanName = strings.ReplaceAll(cleanName, " ", "_")
	cleanName = strings.ReplaceAll(cleanName, "-", "_")

	// Create safe name for function (remove underscores for camelCase)
	parts := strings.Split(cleanName, "_")
	safeName := ""
	for _, part := range parts {
		if len(part) > 0 {
			safeName += strings.Title(part)
		}
	}

	data := MigrationData{
		Version:     timestamp,
		Name:        cleanName,
		SafeName:    safeName,
		Description: name,
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Generate filename
	filename := fmt.Sprintf("%s_%s.go", timestamp, cleanName)
	filepath := filepath.Join(outputDir, filename)

	// Check if file already exists
	if _, err := os.Stat(filepath); err == nil {
		fmt.Printf("Error: file already exists: %s\n", filepath)
		os.Exit(1)
	}

	// Parse template
	tmpl, err := template.New("migration").Parse(goMigrationTemplate)
	if err != nil {
		fmt.Printf("Error parsing template: %v\n", err)
		os.Exit(1)
	}

	// Create file
	file, err := os.Create(filepath)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, data); err != nil {
		fmt.Printf("Error writing template: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Created migration: %s\n", filepath)
	fmt.Printf("  Version: %s\n", timestamp)
	fmt.Printf("  Name: %s\n", cleanName)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit the generated file and implement Up/Down functions")
	fmt.Println("  2. Register the migration with your migration source")
	fmt.Println("  3. Run your migrations")
}

func printHelp() {
	fmt.Println("GoREST Migration Generator")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  generate-migration -name=<migration_name> [-dir=<output_dir>]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -name string")
	fmt.Println("        Migration name (required)")
	fmt.Println("        Example: create_users_table, add_email_index")
	fmt.Println()
	fmt.Println("  -dir string")
	fmt.Println("        Output directory (default: current directory)")
	fmt.Println()
	fmt.Println("  -helpers")
	fmt.Println("        List available migration helper functions")
	fmt.Println()
	fmt.Println("  -help")
	fmt.Println("        Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  generate-migration -name=create_users_table")
	fmt.Println("  generate-migration -name=add_email_index -dir=migrations")
	fmt.Println()
	fmt.Println("The generator creates a timestamped migration file using the format:")
	fmt.Println("  YYYYMMDDHHMMSSmmm_description.go")
	fmt.Println()
	fmt.Println("Example output:")
	fmt.Println("  20251222204530123_create_users_table.go")
}

func printHelpers() {
	fmt.Println("Available Migration Helper Functions")
	fmt.Println("=====================================")
	fmt.Println()
	fmt.Println("Table Operations:")
	fmt.Println("  migrations.CreateTableIfNotExists(ctx, db, tableName, columns)")
	fmt.Println("  migrations.DropTableIfExists(ctx, db, tableName)")
	fmt.Println()
	fmt.Println("Column Operations:")
	fmt.Println("  migrations.AddColumn(ctx, db, tableName, columnDef)")
	fmt.Println("  migrations.DropColumn(ctx, db, tableName, columnName)")
	fmt.Println()
	fmt.Println("Index Operations:")
	fmt.Println("  migrations.CreateIndex(ctx, db, indexName, tableName, columns)")
	fmt.Println("  migrations.DropIndex(ctx, db, indexName, tableName)")
	fmt.Println()
	fmt.Println("Dialect-Specific SQL:")
	fmt.Println("  migrations.SQL(ctx, db, migrations.DialectSQL{")
	fmt.Println("    Postgres: \"CREATE TABLE...\",")
	fmt.Println("    MySQL:    \"CREATE TABLE...\",")
	fmt.Println("    SQLite:   \"CREATE TABLE...\",")
	fmt.Println("  })")
	fmt.Println()
	fmt.Println("Migration Builder:")
	fmt.Println("  builder := migrations.NewMigrationBuilder(\"source_name\")")
	fmt.Println("  builder.Add(version, description, upFunc, downFunc)")
	fmt.Println("  builder.AddSQL(version, description, upSQL, downSQL)")
	fmt.Println("  source := builder.Build()")
	fmt.Println()
	fmt.Println("Example Usage:")
	fmt.Println()
	fmt.Println("  Up: func(ctx context.Context, db database.Database) error {")
	fmt.Println("    return migrations.CreateTableIfNotExists(ctx, db, \"users\",")
	fmt.Println("      \"id TEXT PRIMARY KEY, email TEXT NOT NULL\")")
	fmt.Println("  }")
	fmt.Println()
	fmt.Println("  Down: func(ctx context.Context, db database.Database) error {")
	fmt.Println("    return migrations.DropTableIfExists(ctx, db, \"users\")")
	fmt.Println("  }")
	fmt.Println()
	fmt.Println("For more examples, see migrations/examples.go")
}
