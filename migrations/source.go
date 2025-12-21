package migrations

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/nicolasbonnici/gorest/database"
)

// EmbeddedSource wraps go:embed FS for migrations
type EmbeddedSource struct {
	name    string
	fs      fs.FS
	dir     string
	db      database.Database // Needed for dialect detection
	dialect string            // Cached dialect name
}

// NewEmbeddedSource creates a source from embedded files
func NewEmbeddedSource(name string, filesys fs.FS, dir string, db database.Database) *EmbeddedSource {
	var dialect string
	if db != nil {
		dialect = db.DriverName()
	}

	return &EmbeddedSource{
		name:    name,
		fs:      filesys,
		dir:     dir,
		db:      db,
		dialect: dialect,
	}
}

// Name returns source identifier
func (s *EmbeddedSource) Name() string {
	return s.name
}

// Migrations discovers and parses migration files from embedded FS
func (s *EmbeddedSource) Migrations() ([]Migration, error) {
	// Pattern: {timestamp}_{name}.{up|down}[.{dialect}].sql
	// Examples:
	//   20250120143022_create_users.up.sql
	//   20250120143022_create_users.down.sql
	//   20250120143022_create_users.up.postgres.sql
	//   20250120143022_create_users.down.mysql.sql

	pattern := regexp.MustCompile(`^(\d{14})_([^.]+)\.(up|down)(?:\.([a-z]+))?\.sql$`)

	// Map to group up/down migrations
	migrationPairs := make(map[string]*Migration)

	err := fs.WalkDir(s.fs, s.dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Get relative path from dir
		relPath := strings.TrimPrefix(p, s.dir)
		relPath = strings.TrimPrefix(relPath, "/")

		matches := pattern.FindStringSubmatch(path.Base(relPath))
		if matches == nil {
			// Not a migration file, skip
			return nil
		}

		version := matches[1]
		name := matches[2]
		direction := matches[3]
		dialect := matches[4] // May be empty

		// Check if this file is for current dialect
		if !s.shouldLoadFile(dialect) {
			return nil
		}

		// Read file content
		content, err := fs.ReadFile(s.fs, p)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", p, err)
		}

		sql := string(content)

		// Create or update migration
		key := version + "_" + name
		migration, exists := migrationPairs[key]
		if !exists {
			migration = &Migration{
				Version: version,
				Name:    name,
				Source:  s.name,
				Timeout: 30 * time.Second, // Default timeout
			}
			migrationPairs[key] = migration
		}

		// Set SQL based on direction
		if direction == "up" {
			migration.UpSQL = sql
		} else {
			migration.DownSQL = sql
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert map to slice and calculate checksums
	var migrations []Migration
	for _, m := range migrationPairs {
		// Validate migration has both up and down
		if m.UpSQL == "" {
			return nil, fmt.Errorf("%w: missing up migration for %s", ErrInvalidMigrationFile, m.FullName())
		}
		if m.DownSQL == "" {
			return nil, fmt.Errorf("%w: missing down migration for %s", ErrInvalidMigrationFile, m.FullName())
		}

		// Calculate checksum
		m.Checksum = m.CalculateChecksum()

		migrations = append(migrations, *m)
	}

	// Sort by version (timestamp)
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// shouldLoadFile determines if a migration file should be loaded based on dialect
func (s *EmbeddedSource) shouldLoadFile(fileDialect string) bool {
	// If file has no dialect suffix, it's generic - always load
	if fileDialect == "" {
		return true
	}

	// If we don't have a database connection yet, load all dialects
	// (migrations will be filtered later)
	if s.dialect == "" {
		return true
	}

	// Only load if file dialect matches current dialect
	return fileDialect == s.dialect
}

// SetDialect updates the dialect for an existing source (for late binding)
func (s *EmbeddedSource) SetDialect(db database.Database) {
	if db != nil {
		s.db = db
		s.dialect = db.DriverName()
	}
}

// ValidateTimestamp checks if a version string is a valid timestamp
func ValidateTimestamp(version string) error {
	if len(version) != 14 {
		return fmt.Errorf("%w: version must be 14 digits (YYYYMMDDHHMMSS), got: %s", ErrInvalidVersion, version)
	}

	// Try to parse as timestamp
	_, err := time.Parse("20060102150405", version)
	if err != nil {
		return fmt.Errorf("%w: invalid timestamp format: %s", ErrInvalidVersion, version)
	}

	return nil
}
