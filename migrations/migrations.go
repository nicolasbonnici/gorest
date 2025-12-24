package migrations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/nicolasbonnici/gorest/database"
)

// MigrationExecutor defines how a migration executes its up/down logic
type MigrationExecutor interface {
	Up(ctx context.Context, db database.Database) error
	Down(ctx context.Context, db database.Database) error
	Checksum() string
}

// SQLMigrationExecutor executes migrations using raw SQL strings
type SQLMigrationExecutor struct {
	upSQL   string
	downSQL string
}

func NewSQLMigrationExecutor(upSQL, downSQL string) *SQLMigrationExecutor {
	return &SQLMigrationExecutor{
		upSQL:   upSQL,
		downSQL: downSQL,
	}
}

func (e *SQLMigrationExecutor) Up(ctx context.Context, db database.Database) error {
	_, err := db.Exec(ctx, e.upSQL)
	return err
}

func (e *SQLMigrationExecutor) Down(ctx context.Context, db database.Database) error {
	_, err := db.Exec(ctx, e.downSQL)
	return err
}

func (e *SQLMigrationExecutor) Checksum() string {
	h := sha256.New()
	h.Write([]byte(e.upSQL))
	h.Write([]byte(e.downSQL))
	return hex.EncodeToString(h.Sum(nil))
}

// Migration represents a single database migration
type Migration struct {
	Version  string // Timestamp: "20250120143022123" (17 digits with milliseconds) or "20250120143022" (14 digits legacy)
	Name     string // Descriptive name: "create_users"
	Source   string // Source identifier: "app", "auth", etc.
	Executor MigrationExecutor
	Checksum string // Prevents drift when migration files are modified
	Timeout  time.Duration

	// Deprecated: Use Executor instead
	UpSQL   string
	DownSQL string
}

// FullName returns version_name (e.g., "20250120143022123_create_users")
func (m Migration) FullName() string {
	return m.Version + "_" + m.Name
}

func (m Migration) CalculateChecksum() string {
	if m.Executor != nil {
		return m.Executor.Checksum()
	}
	// Legacy SQL-based checksum
	h := sha256.New()
	h.Write([]byte(m.UpSQL))
	h.Write([]byte(m.DownSQL))
	return hex.EncodeToString(h.Sum(nil))
}

// ExecuteUp executes the migration's up logic
func (m Migration) ExecuteUp(ctx context.Context, db database.Database) error {
	if m.Executor != nil {
		return m.Executor.Up(ctx, db)
	}
	// Legacy SQL execution
	if m.UpSQL == "" {
		return fmt.Errorf("migration has no up SQL or executor")
	}
	_, err := db.Exec(ctx, m.UpSQL)
	return err
}

// ExecuteDown executes the migration's down logic
func (m Migration) ExecuteDown(ctx context.Context, db database.Database) error {
	if m.Executor != nil {
		return m.Executor.Down(ctx, db)
	}
	// Legacy SQL execution
	if m.DownSQL == "" {
		return fmt.Errorf("migration has no down SQL or executor")
	}
	_, err := db.Exec(ctx, m.DownSQL)
	return err
}

// MigrationSource represents a source of migrations (app or plugin)
type MigrationSource interface {
	Name() string
	Migrations() ([]Migration, error)
}

// MigrationStatus represents the state of a migration
type MigrationStatus struct {
	Migration     Migration
	Applied       bool
	AppliedAt     *time.Time
	ExecutionTime time.Duration
	Checksum      string
	Status        string // "pending", "applied", "failed", "rolled_back"
	Error         string
	ExecutedBy    string
	Hostname      string
}

// MigrationOptions configures migration execution
type MigrationOptions struct {
	Transactional bool // All-or-nothing: all succeed or all rollback
	DryRun        bool
	StopOnError   bool
}

// Migrator executes migrations against a database
type Migrator interface {
	Up(ctx context.Context) error
	UpWithOptions(ctx context.Context, opts MigrationOptions) error
	UpOne(ctx context.Context) error
	UpTo(ctx context.Context, version string) error
	Down(ctx context.Context) error
	DownTo(ctx context.Context, version string) error
	Status(ctx context.Context) ([]MigrationStatus, error)
	Pending(ctx context.Context) ([]Migration, error)
	Validate(ctx context.Context) error
	DryRun(ctx context.Context) ([]Migration, error)
	Force(ctx context.Context, version, source string) error
	UpSource(ctx context.Context, sourceName string) error
	DownSource(ctx context.Context, sourceName string) error
	SetSourceDependencies(source string, dependencies []string)
}
