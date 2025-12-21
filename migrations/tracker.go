package migrations

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nicolasbonnici/gorest/database"
)

// MigrationTracker manages the schema_migrations table
type MigrationTracker struct {
	db database.Database
}

func NewMigrationTracker(db database.Database) *MigrationTracker {
	return &MigrationTracker{db: db}
}

func (t *MigrationTracker) CreateTrackingTable(ctx context.Context) error {
	driverName := t.db.DriverName()

	var createSQL string

	switch driverName {
	case "postgres":
		createSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(14) NOT NULL,
    source VARCHAR(100) NOT NULL DEFAULT 'app',
    name VARCHAR(255) NOT NULL,
    checksum CHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'applied',
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    execution_time_ms INTEGER,
    executed_by VARCHAR(100),
    hostname VARCHAR(255),
    error_message TEXT,
    PRIMARY KEY (version, source)
);

CREATE INDEX IF NOT EXISTS idx_migrations_status ON schema_migrations(status);
CREATE INDEX IF NOT EXISTS idx_migrations_applied_at ON schema_migrations(applied_at DESC);
CREATE INDEX IF NOT EXISTS idx_migrations_source ON schema_migrations(source);
`

	case "mysql":
		createSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(14) NOT NULL,
    source VARCHAR(100) NOT NULL DEFAULT 'app',
    name VARCHAR(255) NOT NULL,
    checksum CHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'applied',
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    execution_time_ms INTEGER,
    executed_by VARCHAR(100),
    hostname VARCHAR(255),
    error_message TEXT,
    PRIMARY KEY (version, source),
    INDEX idx_migrations_status (status),
    INDEX idx_migrations_applied_at (applied_at DESC),
    INDEX idx_migrations_source (source)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`

	case "sqlite":
		createSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'app',
    name TEXT NOT NULL,
    checksum TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'applied',
    applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    execution_time_ms INTEGER,
    executed_by TEXT,
    hostname TEXT,
    error_message TEXT,
    PRIMARY KEY (version, source)
);

CREATE INDEX IF NOT EXISTS idx_migrations_status ON schema_migrations(status);
CREATE INDEX IF NOT EXISTS idx_migrations_applied_at ON schema_migrations(applied_at DESC);
CREATE INDEX IF NOT EXISTS idx_migrations_source ON schema_migrations(source);
`

	default:
		return fmt.Errorf("unsupported database driver: %s", driverName)
	}

	_, err := t.db.Exec(ctx, createSQL)
	return err
}

func (t *MigrationTracker) GetAppliedMigrations(ctx context.Context) ([]MigrationStatus, error) {
	query := `
SELECT version, source, name, checksum, status, applied_at,
       execution_time_ms, executed_by, hostname, error_message
FROM schema_migrations
ORDER BY version ASC
`

	rows, err := t.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statuses []MigrationStatus

	for rows.Next() {
		var (
			version       string
			source        string
			name          string
			checksum      string
			status        string
			appliedAt     time.Time
			executionTime *int
			executedBy    *string
			hostname      *string
			errorMessage  *string
		)

		err := rows.Scan(
			&version, &source, &name, &checksum, &status, &appliedAt,
			&executionTime, &executedBy, &hostname, &errorMessage,
		)
		if err != nil {
			return nil, err
		}

		ms := MigrationStatus{
			Migration: Migration{
				Version: version,
				Name:    name,
				Source:  source,
			},
			Applied:    status == "applied",
			AppliedAt:  &appliedAt,
			Checksum:   checksum,
			Status:     status,
			ExecutedBy: getString(executedBy),
			Hostname:   getString(hostname),
			Error:      getString(errorMessage),
		}

		if executionTime != nil {
			ms.ExecutionTime = time.Duration(*executionTime) * time.Millisecond
		}

		statuses = append(statuses, ms)
	}

	return statuses, rows.Err()
}

func (t *MigrationTracker) RecordMigration(ctx context.Context, migration Migration, executionTime time.Duration) error {
	hostname, _ := os.Hostname()
	executedBy := os.Getenv("USER")

	query := `
INSERT INTO schema_migrations
    (version, source, name, checksum, status, execution_time_ms, executed_by, hostname)
VALUES (` + t.placeholders(1, 8) + `)
`

	_, err := t.db.Exec(ctx, query,
		migration.Version,
		migration.Source,
		migration.Name,
		migration.Checksum,
		"applied",
		int(executionTime.Milliseconds()),
		executedBy,
		hostname,
	)

	return err
}

func (t *MigrationTracker) RecordFailedMigration(ctx context.Context, migration Migration, errorMsg string) error {
	hostname, _ := os.Hostname()
	executedBy := os.Getenv("USER")

	query := `
INSERT INTO schema_migrations
    (version, source, name, checksum, status, executed_by, hostname, error_message)
VALUES (` + t.placeholders(1, 8) + `)
ON CONFLICT (version, source) DO UPDATE SET
    status = EXCLUDED.status,
    error_message = EXCLUDED.error_message,
    applied_at = CURRENT_TIMESTAMP
`

	if t.db.DriverName() == "mysql" {
		query = `
INSERT INTO schema_migrations
    (version, source, name, checksum, status, executed_by, hostname, error_message)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    status = VALUES(status),
    error_message = VALUES(error_message),
    applied_at = CURRENT_TIMESTAMP
`
	} else if t.db.DriverName() == "sqlite" {
		query = `
INSERT INTO schema_migrations
    (version, source, name, checksum, status, executed_by, hostname, error_message)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(version, source) DO UPDATE SET
    status = excluded.status,
    error_message = excluded.error_message,
    applied_at = CURRENT_TIMESTAMP
`
	}

	_, err := t.db.Exec(ctx, query,
		migration.Version,
		migration.Source,
		migration.Name,
		migration.Checksum,
		"failed",
		executedBy,
		hostname,
		errorMsg,
	)

	return err
}

func (t *MigrationTracker) RemoveMigration(ctx context.Context, version, source string) error {
	query := `DELETE FROM schema_migrations WHERE version = ` + t.db.Dialect().Placeholder(1) + ` AND source = ` + t.db.Dialect().Placeholder(2)

	_, err := t.db.Exec(ctx, query, version, source)
	return err
}

func (t *MigrationTracker) CheckForDirtyDatabase(ctx context.Context) error {
	query := `
SELECT version, source, name, error_message
FROM schema_migrations
WHERE status = 'failed'
ORDER BY version ASC
`

	rows, err := t.db.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	var failedMigrations []MigrationStatus

	for rows.Next() {
		var (
			version      string
			source       string
			name         string
			errorMessage *string
		)

		err := rows.Scan(&version, &source, &name, &errorMessage)
		if err != nil {
			return err
		}

		failedMigrations = append(failedMigrations, MigrationStatus{
			Migration: Migration{
				Version: version,
				Name:    name,
				Source:  source,
			},
			Status: "failed",
			Error:  getString(errorMessage),
		})
	}

	if err := rows.Err(); err != nil {
		return err
	}

	if len(failedMigrations) > 0 {
		return &DirtyDatabaseError{FailedMigrations: failedMigrations}
	}

	return nil
}

func (t *MigrationTracker) VerifyChecksums(ctx context.Context, migrations []Migration) error {
	applied, err := t.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	appliedMap := make(map[string]MigrationStatus)
	for _, m := range applied {
		key := m.Migration.Version + ":" + m.Migration.Source
		appliedMap[key] = m
	}

	for _, migration := range migrations {
		key := migration.Version + ":" + migration.Source

		if appliedStatus, exists := appliedMap[key]; exists && appliedStatus.Status == "applied" {
			if appliedStatus.Checksum != migration.Checksum {
				return &ChecksumMismatchError{
					Migration:        migration,
					ExpectedChecksum: appliedStatus.Checksum,
					ActualChecksum:   migration.Checksum,
				}
			}
		}
	}

	return nil
}

func (t *MigrationTracker) ForceMigration(ctx context.Context, migration Migration) error {
	hostname, _ := os.Hostname()
	executedBy := os.Getenv("USER")

	query := `
INSERT INTO schema_migrations
    (version, source, name, checksum, status, executed_by, hostname, execution_time_ms)
VALUES (` + t.placeholders(1, 8) + `)
ON CONFLICT (version, source) DO UPDATE SET
    status = 'applied',
    error_message = NULL,
    applied_at = CURRENT_TIMESTAMP
`

	if t.db.DriverName() == "mysql" {
		query = `
INSERT INTO schema_migrations
    (version, source, name, checksum, status, executed_by, hostname, execution_time_ms)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    status = 'applied',
    error_message = NULL,
    applied_at = CURRENT_TIMESTAMP
`
	} else if t.db.DriverName() == "sqlite" {
		query = `
INSERT INTO schema_migrations
    (version, source, name, checksum, status, executed_by, hostname, execution_time_ms)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(version, source) DO UPDATE SET
    status = 'applied',
    error_message = NULL,
    applied_at = CURRENT_TIMESTAMP
`
	}

	_, err := t.db.Exec(ctx, query,
		migration.Version,
		migration.Source,
		migration.Name,
		migration.Checksum,
		"applied",
		executedBy,
		hostname,
		0, // execution_time_ms
	)

	return err
}

func (t *MigrationTracker) placeholders(start, count int) string {
	if t.db.DriverName() == "postgres" {
		placeholders := ""
		for i := start; i < start+count; i++ {
			if i > start {
				placeholders += ", "
			}
			placeholders += fmt.Sprintf("$%d", i)
		}
		return placeholders
	}

	placeholders := ""
	for i := 0; i < count; i++ {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "?"
	}
	return placeholders
}

func getString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
