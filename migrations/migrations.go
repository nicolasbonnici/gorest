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
	UpSQL    string
	DownSQL  string
	Checksum string // Prevents drift when migration files are modified
	Timeout  time.Duration
}

// FullName returns version_name (e.g., "20250120143022_create_users")
func (m Migration) FullName() string {
	return m.Version + "_" + m.Name
}

func (m Migration) CalculateChecksum() string {
	h := sha256.New()
	h.Write([]byte(m.UpSQL))
	h.Write([]byte(m.DownSQL))
	return hex.EncodeToString(h.Sum(nil))
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
