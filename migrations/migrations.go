package migrations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// Migration represents a single database migration with up/down SQL
type Migration struct {
	Version  string        // Timestamp: "20250120143022"
	Name     string        // Descriptive name: "create_users"
	Source   string        // Source identifier: "app", "auth", etc.
	UpSQL    string        // SQL for applying migration
	DownSQL  string        // SQL for reverting migration
	Checksum string        // SHA256 hash of UpSQL+DownSQL (prevents drift)
	Timeout  time.Duration // Per-migration timeout (default: 30s)
}

// FullName returns version_name (e.g., "20250120143022_create_users")
func (m Migration) FullName() string {
	return m.Version + "_" + m.Name
}

// CalculateChecksum computes SHA256 hash of migration content
func (m Migration) CalculateChecksum() string {
	h := sha256.New()
	h.Write([]byte(m.UpSQL))
	h.Write([]byte(m.DownSQL))
	return hex.EncodeToString(h.Sum(nil))
}

// MigrationSource represents a source of migrations (app or plugin)
type MigrationSource interface {
	// Name returns the source identifier (e.g., "app", "auth", "logger")
	Name() string

	// Migrations returns all migration files from this source
	Migrations() ([]Migration, error)
}

// MigrationStatus represents the state of a migration
type MigrationStatus struct {
	Migration     Migration
	Applied       bool
	AppliedAt     *time.Time
	ExecutionTime time.Duration // How long it took to execute
	Checksum      string         // Stored checksum
	Status        string         // "pending", "applied", "failed", "rolled_back"
	Error         string         // Error message if status='failed'
	ExecutedBy    string         // Who ran the migration
	Hostname      string         // Where it ran
}

// MigrationOptions configures migration execution
type MigrationOptions struct {
	Transactional bool // Wrap all migrations in single transaction (all-or-nothing)
	DryRun        bool // Show what would execute without executing
	StopOnError   bool // Stop on first error (default: true)
}

// Migrator executes migrations against a database
type Migrator interface {
	// Up applies all pending migrations from all sources
	Up(ctx context.Context) error

	// UpWithOptions applies migrations with custom options
	UpWithOptions(ctx context.Context, opts MigrationOptions) error

	// UpOne applies the next pending migration (any source, ordered by version)
	UpOne(ctx context.Context) error

	// UpTo applies migrations up to specific version
	UpTo(ctx context.Context, version string) error

	// Down reverts the most recently applied migration (any source)
	Down(ctx context.Context) error

	// DownTo reverts migrations down to specific version
	DownTo(ctx context.Context, version string) error

	// Status returns migration status for all sources
	Status(ctx context.Context) ([]MigrationStatus, error)

	// Pending returns list of pending migrations
	Pending(ctx context.Context) ([]Migration, error)

	// Validate validates all migrations without executing
	Validate(ctx context.Context) error

	// DryRun shows what would be executed
	DryRun(ctx context.Context) ([]Migration, error)

	// Force marks a migration as applied without executing (repair tool - use with caution)
	Force(ctx context.Context, version, source string) error

	// UpSource applies all pending migrations for a specific source
	UpSource(ctx context.Context, sourceName string) error

	// DownSource reverts the most recent migration for a specific source
	DownSource(ctx context.Context, sourceName string) error
}
