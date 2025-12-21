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

func (s *EmbeddedSource) Name() string {
	return s.name
}

// Migrations discovers and parses migration files from embedded FS.
// File pattern: {timestamp}_{name}.{up|down}[.{dialect}].sql
func (s *EmbeddedSource) Migrations() ([]Migration, error) {
	pattern := regexp.MustCompile(`^(\d{14})_([^.]+)\.(up|down)(?:\.([a-z]+))?\.sql$`)

	migrationPairs := make(map[string]*Migration)

	err := fs.WalkDir(s.fs, s.dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		relPath := strings.TrimPrefix(p, s.dir)
		relPath = strings.TrimPrefix(relPath, "/")

		matches := pattern.FindStringSubmatch(path.Base(relPath))
		if matches == nil {
			return nil
		}

		version := matches[1]
		name := matches[2]
		direction := matches[3]
		dialect := matches[4]

		if !s.shouldLoadFile(dialect) {
			return nil
		}

		content, err := fs.ReadFile(s.fs, p)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", p, err)
		}

		sql := string(content)

		key := version + "_" + name
		migration, exists := migrationPairs[key]
		if !exists {
			migration = &Migration{
				Version: version,
				Name:    name,
				Source:  s.name,
				Timeout: 30 * time.Second,
			}
			migrationPairs[key] = migration
		}

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

	var migrations []Migration
	for _, m := range migrationPairs {
		if m.UpSQL == "" {
			return nil, fmt.Errorf("%w: missing up migration for %s", ErrInvalidMigrationFile, m.FullName())
		}
		if m.DownSQL == "" {
			return nil, fmt.Errorf("%w: missing down migration for %s", ErrInvalidMigrationFile, m.FullName())
		}

		m.Checksum = m.CalculateChecksum()
		migrations = append(migrations, *m)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func (s *EmbeddedSource) shouldLoadFile(fileDialect string) bool {
	if fileDialect == "" {
		return true
	}

	if s.dialect == "" {
		return true
	}

	return fileDialect == s.dialect
}

// SetDialect updates the dialect for an existing source (for late binding)
func (s *EmbeddedSource) SetDialect(db database.Database) {
	if db != nil {
		s.db = db
		s.dialect = db.DriverName()
	}
}

func ValidateTimestamp(version string) error {
	if len(version) != 14 {
		return fmt.Errorf("%w: version must be 14 digits (YYYYMMDDHHMMSS), got: %s", ErrInvalidVersion, version)
	}

	_, err := time.Parse("20060102150405", version)
	if err != nil {
		return fmt.Errorf("%w: invalid timestamp format: %s", ErrInvalidVersion, version)
	}

	return nil
}
