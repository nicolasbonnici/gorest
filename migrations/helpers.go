package migrations

import (
	"context"
	"fmt"

	"github.com/nicolasbonnici/gorest/database"
)

// GoMigration is a helper for creating Go-based migrations using the DAL
type GoMigration struct {
	version     string
	description string
	upFunc      func(ctx context.Context, db database.Database) error
	downFunc    func(ctx context.Context, db database.Database) error
}

// NewGoMigration creates a new Go-based migration with timestamp
func NewGoMigration(version, description string) *GoMigration {
	return &GoMigration{
		version:     version,
		description: description,
	}
}

// Up sets the up migration function
func (m *GoMigration) Up(fn func(ctx context.Context, db database.Database) error) *GoMigration {
	m.upFunc = fn
	return m
}

// Down sets the down migration function
func (m *GoMigration) Down(fn func(ctx context.Context, db database.Database) error) *GoMigration {
	m.downFunc = fn
	return m
}

// Build creates a Migration from the GoMigration
func (m *GoMigration) Build() Migration {
	executor := &GoMigrationExecutor{
		upFunc:   m.upFunc,
		downFunc: m.downFunc,
	}

	return Migration{
		Version:  m.version,
		Name:     m.description,
		Executor: executor,
		Checksum: executor.Checksum(),
	}
}

// GoMigrationExecutor executes Go-based migration functions
type GoMigrationExecutor struct {
	upFunc   func(ctx context.Context, db database.Database) error
	downFunc func(ctx context.Context, db database.Database) error
}

func (e *GoMigrationExecutor) Up(ctx context.Context, db database.Database) error {
	if e.upFunc == nil {
		return fmt.Errorf("up function not defined")
	}
	return e.upFunc(ctx, db)
}

func (e *GoMigrationExecutor) Down(ctx context.Context, db database.Database) error {
	if e.downFunc == nil {
		return fmt.Errorf("down function not defined")
	}
	return e.downFunc(ctx, db)
}

func (e *GoMigrationExecutor) Checksum() string {
	// For Go-based migrations, we use a simple hash based on function pointers
	// This is stable within a build but allows code changes between builds
	return fmt.Sprintf("%p_%p", e.upFunc, e.downFunc)
}

// MigrationBuilder provides a fluent interface for building migrations
type MigrationBuilder struct {
	migrations []Migration
	sourceName string
}

// NewMigrationBuilder creates a new migration builder
func NewMigrationBuilder(sourceName string) *MigrationBuilder {
	return &MigrationBuilder{
		sourceName: sourceName,
		migrations: make([]Migration, 0),
	}
}

// Add adds a migration to the builder
func (b *MigrationBuilder) Add(version, description string, up, down func(ctx context.Context, db database.Database) error) *MigrationBuilder {
	migration := NewGoMigration(version, description).
		Up(up).
		Down(down).
		Build()

	migration.Source = b.sourceName
	b.migrations = append(b.migrations, migration)
	return b
}

// AddSQL adds a SQL-based migration to the builder
func (b *MigrationBuilder) AddSQL(version, description, upSQL, downSQL string) *MigrationBuilder {
	migration := Migration{
		Version:  version,
		Name:     description,
		Source:   b.sourceName,
		Executor: NewSQLMigrationExecutor(upSQL, downSQL),
	}
	migration.Checksum = migration.CalculateChecksum()
	b.migrations = append(b.migrations, migration)
	return b
}

// Build creates a GoMigrationSource from the builder
func (b *MigrationBuilder) Build() *GoMigrationSource {
	return NewGoMigrationSource(b.sourceName, b.migrations)
}

// Common migration helpers using DAL

// CreateTableIfNotExists creates a table with dialect-aware SQL
func CreateTableIfNotExists(ctx context.Context, db database.Database, tableName, columns string) error {
	var sql string

	switch db.DriverName() {
	case "postgres":
		sql = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tableName, columns)
	case "mysql":
		sql = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4", tableName, columns)
	case "sqlite":
		sql = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tableName, columns)
	default:
		return fmt.Errorf("unsupported database driver: %s", db.DriverName())
	}

	_, err := db.Exec(ctx, sql)
	return err
}

// DropTableIfExists drops a table if it exists
func DropTableIfExists(ctx context.Context, db database.Database, tableName string) error {
	cascade := ""
	if db.DriverName() == "postgres" {
		cascade = " CASCADE"
	}

	sql := fmt.Sprintf("DROP TABLE IF EXISTS %s%s", tableName, cascade)
	_, err := db.Exec(ctx, sql)
	return err
}

// AddColumn adds a column to an existing table
func AddColumn(ctx context.Context, db database.Database, tableName, columnDef string) error {
	sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tableName, columnDef)
	_, err := db.Exec(ctx, sql)
	return err
}

// DropColumn drops a column from a table
func DropColumn(ctx context.Context, db database.Database, tableName, columnName string) error {
	sql := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tableName, columnName)
	_, err := db.Exec(ctx, sql)
	return err
}

// CreateIndex creates an index on a table
func CreateIndex(ctx context.Context, db database.Database, indexName, tableName, columns string) error {
	sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)", indexName, tableName, columns)

	// MySQL doesn't support IF NOT EXISTS for indexes in older versions
	if db.DriverName() == "mysql" {
		sql = fmt.Sprintf("CREATE INDEX %s ON %s (%s)", indexName, tableName, columns)
	}

	_, err := db.Exec(ctx, sql)
	return err
}

// DropIndex drops an index
func DropIndex(ctx context.Context, db database.Database, indexName, tableName string) error {
	var sql string

	switch db.DriverName() {
	case "postgres", "sqlite":
		sql = fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName)
	case "mysql":
		sql = fmt.Sprintf("DROP INDEX %s ON %s", indexName, tableName)
	default:
		return fmt.Errorf("unsupported database driver: %s", db.DriverName())
	}

	_, err := db.Exec(ctx, sql)
	return err
}

// DialectSQL holds SQL statements for different database dialects.
// All fields are optional - only provide SQL for the databases you support.
type DialectSQL struct {
	Postgres string
	MySQL    string
	SQLite   string
}

// SQL executes different SQL based on database dialect.
// You can specify SQL for one, two, or all three databases.
func SQL(ctx context.Context, db database.Database, sql DialectSQL) error {
	var query string

	switch db.DriverName() {
	case "postgres":
		query = sql.Postgres
	case "mysql":
		query = sql.MySQL
	case "sqlite":
		query = sql.SQLite
	default:
		return fmt.Errorf("unsupported database driver: %s", db.DriverName())
	}

	if query == "" {
		return fmt.Errorf("no SQL provided for %s dialect", db.DriverName())
	}

	_, err := db.Exec(ctx, query)
	return err
}
