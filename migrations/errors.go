package migrations

import (
	"errors"
	"fmt"
)

var (
	ErrNoMigrations        = errors.New("no migrations found")
	ErrNoPendingMigrations = errors.New("no pending migrations")
	ErrMigrationNotFound   = errors.New("migration not found")
	ErrInvalidMigrationFile = errors.New("invalid migration file format")
	ErrMigrationFailed     = errors.New("migration execution failed")
	ErrChecksumMismatch    = errors.New("migration checksum mismatch")
	ErrDirtyDatabase       = errors.New("database is in dirty state")
	ErrLockTimeout         = errors.New("could not acquire migration lock")
	ErrMigrationTimeout    = errors.New("migration execution timeout")
	ErrInvalidVersion      = errors.New("invalid migration version")
	ErrAlreadyApplied      = errors.New("migration already applied")
	ErrCircularDependency  = errors.New("circular dependency detected")
)

// MigrationError wraps migration errors with rich context
type MigrationError struct {
	Migration   Migration
	Err         error
	SQL         string
	Line        int
	DatabaseErr string
	Hint        string
}

func (e *MigrationError) Error() string {
	msg := fmt.Sprintf(
		"migration %s/%s failed: %v\n",
		e.Migration.Source, e.Migration.FullName(), e.Err,
	)

	if e.DatabaseErr != "" {
		msg += fmt.Sprintf("Database error: %s\n", e.DatabaseErr)
	}

	if e.SQL != "" {
		msg += fmt.Sprintf("SQL: %s\n", e.SQL)
	}

	if e.Hint != "" {
		msg += fmt.Sprintf("Hint: %s", e.Hint)
	}

	return msg
}

func (e *MigrationError) Unwrap() error {
	return e.Err
}

// ChecksumMismatchError indicates migration file was modified after being applied
type ChecksumMismatchError struct {
	Migration        Migration
	ExpectedChecksum string
	ActualChecksum   string
}

func (e *ChecksumMismatchError) Error() string {
	return fmt.Sprintf(
		"CRITICAL: Migration %s/%s has been modified after being applied!\n"+
			"Expected checksum: %s\n"+
			"Actual checksum:   %s\n"+
			"This indicates the migration file was changed after being run in production.\n"+
			"This can cause data inconsistency across environments.\n"+
			"DO NOT modify migrations that have been applied!",
		e.Migration.Source, e.Migration.FullName(),
		e.ExpectedChecksum, e.ActualChecksum,
	)
}

// DirtyDatabaseError indicates failed migrations exist that must be resolved
type DirtyDatabaseError struct {
	FailedMigrations []MigrationStatus
}

func (e *DirtyDatabaseError) Error() string {
	msg := fmt.Sprintf(
		"Database is in dirty state - %d failed migration(s) detected:\n",
		len(e.FailedMigrations),
	)

	for _, m := range e.FailedMigrations {
		msg += fmt.Sprintf(
			"  - [%s] %s: %s\n",
			m.Migration.Source, m.Migration.FullName(), m.Error,
		)
	}

	msg += "\nYou must:\n"
	msg += "  1. Fix the failing migration SQL and retry, OR\n"
	msg += "  2. Use Force() to mark as skipped (dangerous), OR\n"
	msg += "  3. Manually repair database and update schema_migrations status\n"

	return msg
}
