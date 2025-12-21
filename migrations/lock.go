package migrations

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/nicolasbonnici/gorest/database"
)

// MigrationLock handles database-specific advisory locking
type MigrationLock struct {
	db       database.Database
	acquired bool
	lockKey  string
}

// NewMigrationLock creates a new migration lock
func NewMigrationLock(db database.Database) *MigrationLock {
	return &MigrationLock{
		db:      db,
		lockKey: "gorest_migrations",
	}
}

// Acquire obtains an advisory lock to prevent concurrent migrations
func (l *MigrationLock) Acquire(ctx context.Context) error {
	if l.acquired {
		return nil
	}

	driverName := l.db.DriverName()

	switch driverName {
	case "postgres":
		return l.acquirePostgres(ctx)
	case "mysql":
		return l.acquireMySQL(ctx)
	case "sqlite":
		return l.acquireSQLite(ctx)
	default:
		return fmt.Errorf("unsupported database driver for locking: %s", driverName)
	}
}

// Release releases the advisory lock
func (l *MigrationLock) Release(ctx context.Context) error {
	if !l.acquired {
		return nil
	}

	driverName := l.db.DriverName()

	switch driverName {
	case "postgres":
		return l.releasePostgres(ctx)
	case "mysql":
		return l.releaseMySQL(ctx)
	case "sqlite":
		return l.releaseSQLite(ctx)
	default:
		return nil
	}
}

// acquirePostgres uses PostgreSQL advisory locks
func (l *MigrationLock) acquirePostgres(ctx context.Context) error {
	// Convert lock key to integer using hash
	lockID := l.hashLockKey()

	query := "SELECT pg_advisory_lock($1)"
	_, err := l.db.Exec(ctx, query, lockID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrLockTimeout, err)
	}

	l.acquired = true
	return nil
}

// releasePostgres releases PostgreSQL advisory lock
func (l *MigrationLock) releasePostgres(ctx context.Context) error {
	lockID := l.hashLockKey()

	query := "SELECT pg_advisory_unlock($1)"
	_, err := l.db.Exec(ctx, query, lockID)
	if err != nil {
		return err
	}

	l.acquired = false
	return nil
}

// acquireMySQL uses MySQL GET_LOCK
func (l *MigrationLock) acquireMySQL(ctx context.Context) error {
	query := "SELECT GET_LOCK(?, 60)"

	var result int
	err := l.db.QueryRow(ctx, query, l.lockKey).Scan(&result)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrLockTimeout, err)
	}

	if result != 1 {
		return fmt.Errorf("%w: failed to acquire lock (result: %d)", ErrLockTimeout, result)
	}

	l.acquired = true
	return nil
}

// releaseMySQL releases MySQL lock
func (l *MigrationLock) releaseMySQL(ctx context.Context) error {
	query := "SELECT RELEASE_LOCK(?)"

	var result int
	err := l.db.QueryRow(ctx, query, l.lockKey).Scan(&result)
	if err != nil {
		return err
	}

	l.acquired = false
	return nil
}

// acquireSQLite uses EXCLUSIVE transaction for SQLite
// Note: The transaction must be managed by the caller for SQLite
func (l *MigrationLock) acquireSQLite(ctx context.Context) error {
	// SQLite doesn't have advisory locks
	// We use BEGIN EXCLUSIVE which will be handled at transaction level
	// For now, just mark as acquired
	l.acquired = true
	return nil
}

// releaseSQLite releases SQLite lock (handled by transaction commit/rollback)
func (l *MigrationLock) releaseSQLite(ctx context.Context) error {
	l.acquired = false
	return nil
}

// hashLockKey converts the lock key string to an integer for PostgreSQL
func (l *MigrationLock) hashLockKey() int64 {
	h := fnv.New64a()
	h.Write([]byte(l.lockKey))
	// PostgreSQL advisory lock uses 64-bit integer, but we'll cast to int64
	return int64(h.Sum64())
}
