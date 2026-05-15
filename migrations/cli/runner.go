package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/nicolasbonnici/gorest/database"
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
			return nil
		}).
		Down(func(ctx context.Context, db database.Database) error {
			// TODO: Implement down migration
			return nil
		}).
		Build()
}
`

// Runner dispatches migrate CLI subcommands.
type Runner struct {
	migrator     migrations.Migrator
	db           database.Database
	migrationDir string
}

// New returns a Runner. db is used directly for status/repair so it works
// even when the migrator has zero sources.
func New(m migrations.Migrator, db database.Database, migrationDir string) *Runner {
	return &Runner{migrator: m, db: db, migrationDir: migrationDir}
}

// Run dispatches args to the appropriate subcommand.
func (r *Runner) Run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	ctx := context.Background()

	switch args[0] {
	case "run":
		return r.cmdRun(ctx, args[1:])
	case "new":
		return r.cmdNew(args[1:])
	case "status":
		return r.cmdStatus(ctx)
	case "pending":
		return r.cmdPending(ctx)
	case "dry-run":
		return r.cmdDryRun(ctx)
	case "repair":
		return r.cmdRepair(ctx, args[1:])
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q — run 'migrate help' for usage", args[0])
	}
}

// cmdRun handles: migrate run [<version> <source> [--down]]
func (r *Runner) cmdRun(ctx context.Context, args []string) error {
	if len(args) == 0 {
		fmt.Println("Running all pending migrations...")
		return r.migrator.Up(ctx)
	}

	if len(args) < 2 {
		return fmt.Errorf("usage: migrate run <version> <source> [--down]")
	}

	version, source := args[0], args[1]
	direction := migrations.DirectionUp
	for _, a := range args[2:] {
		if a == "--down" {
			direction = migrations.DirectionDown
		}
	}

	dir := "up"
	if direction == migrations.DirectionDown {
		dir = "down"
	}
	fmt.Printf("Running migration %s [%s] %s...\n", source, version, dir)
	return r.migrator.RunOne(ctx, version, source, direction)
}

// cmdNew handles: migrate new <name> [--dir <output-dir>]
func (r *Runner) cmdNew(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: migrate new <name> [--dir <output-dir>]")
	}

	name := args[0]
	outDir := r.migrationDir

	for i := 1; i < len(args)-1; i++ {
		if args[i] == "--dir" {
			outDir = args[i+1]
		}
	}

	timestamp := migrations.GenerateTimestamp()
	cleanName := strings.ToLower(name)
	cleanName = strings.ReplaceAll(cleanName, " ", "_")
	cleanName = strings.ReplaceAll(cleanName, "-", "_")

	parts := strings.Split(cleanName, "_")
	var sb strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			runes := []rune(part)
			runes[0] = unicode.ToUpper(runes[0])
			sb.WriteString(string(runes))
		}
	}
	safeName := sb.String()

	data := struct {
		Version, Name, SafeName, Description string
	}{timestamp, cleanName, safeName, name}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	filename := fmt.Sprintf("%s_%s.go", timestamp, cleanName)
	path := filepath.Join(outDir, filename)

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}

	tmpl, err := template.New("m").Parse(goMigrationTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return err
	}

	fmt.Printf("Created migration: %s\n", path)
	fmt.Printf("  Version: %s\n  Name:    %s\n", timestamp, cleanName)
	return nil
}

// cmdStatus prints all migration records from the DB.
func (r *Runner) cmdStatus(ctx context.Context) error {
	tracker := migrations.NewMigrationTracker(r.db)

	if err := tracker.CreateTrackingTable(ctx); err != nil {
		return err
	}

	records, err := tracker.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	if len(records) == 0 {
		fmt.Println("No migration records found.")
		return nil
	}

	fmt.Printf("%-20s %-25s %-35s %-10s %s\n", "VERSION", "SOURCE", "NAME", "STATUS", "ERROR")
	fmt.Println(strings.Repeat("-", 110))

	for _, r := range records {
		errStr := ""
		if r.Error != "" {
			if len(r.Error) > 60 {
				errStr = r.Error[:57] + "..."
			} else {
				errStr = r.Error
			}
		}
		appliedAt := ""
		if r.AppliedAt != nil {
			appliedAt = r.AppliedAt.Format(time.DateTime)
		}
		status := r.Status
		fmt.Printf("%-20s %-25s %-35s %-10s %s %s\n",
			r.Migration.Version,
			r.Migration.Source,
			r.Migration.Name,
			status,
			appliedAt,
			errStr,
		)
	}

	return nil
}

// cmdPending lists pending migrations (requires sources).
func (r *Runner) cmdPending(ctx context.Context) error {
	pending, err := r.migrator.Pending(ctx)
	if err != nil {
		return err
	}

	if len(pending) == 0 {
		fmt.Println("No pending migrations.")
		return nil
	}

	fmt.Printf("%-20s %-25s %s\n", "VERSION", "SOURCE", "NAME")
	fmt.Println(strings.Repeat("-", 80))
	for _, m := range pending {
		fmt.Printf("%-20s %-25s %s\n", m.Version, m.Source, m.Name)
	}
	return nil
}

// cmdDryRun prints migrations that would run without executing them.
func (r *Runner) cmdDryRun(ctx context.Context) error {
	pending, err := r.migrator.DryRun(ctx)
	if err != nil {
		return err
	}

	if len(pending) == 0 {
		fmt.Println("Nothing to run.")
		return nil
	}

	fmt.Println("Would run:")
	for _, m := range pending {
		fmt.Printf("  [%s] %s\n", m.Source, m.FullName())
	}
	return nil
}

// cmdRepair handles: migrate repair <force|retry> <version> <source>
func (r *Runner) cmdRepair(ctx context.Context, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: migrate repair <force|retry> <version> <source>")
	}

	subCmd, version, source := args[0], args[1], args[2]

	switch subCmd {
	case "force":
		fmt.Printf("Forcing migration %s [%s] as applied (no SQL executed)...\n", source, version)
		if err := r.migrator.Force(ctx, version, source); err != nil {
			return err
		}
		fmt.Println("Done.")
	case "retry":
		fmt.Printf("Removing failed record for %s [%s] — will re-run on next startup...\n", source, version)
		if err := r.migrator.Retry(ctx, version, source); err != nil {
			return err
		}
		fmt.Println("Done.")
	default:
		return fmt.Errorf("unknown repair subcommand %q — use 'force' or 'retry'", subCmd)
	}

	return nil
}

func printUsage() {
	fmt.Println(`GoREST Migration CLI

Usage:
  migrate <command> [arguments]

Commands:
  run                              Run all pending migrations
  run <version> <source> [--down]  Run one migration up (or down)
  new <name> [--dir <dir>]         Generate a new migration file
  status                           Show all recorded migration states
  pending                          List pending migrations
  dry-run                          Show what would run without executing
  repair force <version> <source>  Mark a failed migration as applied
  repair retry <version> <source>  Remove failed record so it re-runs

Connection flags (set before the command or via environment):
  --config <dir>    Directory containing gorest.yaml (default: .)
  --dsn <url>       Raw database URL (overrides config)

Examples:
  migrate status --config /var/apps/myapi
  migrate run --dsn postgres://user:pass@localhost/mydb
  migrate repair retry 20250121000001000 gorest-core-auth --config .
  migrate new add_role_column --dir ./migrations`)
}
