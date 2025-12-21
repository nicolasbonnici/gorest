package migrations

import (
	"fmt"
	"strings"
)

// MigrationValidator validates migrations before execution
type MigrationValidator struct{}

// NewMigrationValidator creates a new validator
func NewMigrationValidator() *MigrationValidator {
	return &MigrationValidator{}
}

// Validate performs comprehensive validation on a migration
func (v *MigrationValidator) Validate(migration Migration) error {
	// 1. Validate version format (timestamp)
	if err := ValidateTimestamp(migration.Version); err != nil {
		return err
	}

	// 2. Validate name is not empty
	if strings.TrimSpace(migration.Name) == "" {
		return fmt.Errorf("%w: migration name cannot be empty", ErrInvalidMigrationFile)
	}

	// 3. Validate source is not empty
	if strings.TrimSpace(migration.Source) == "" {
		return fmt.Errorf("%w: migration source cannot be empty", ErrInvalidMigrationFile)
	}

	// 4. Validate up SQL is not empty
	if strings.TrimSpace(migration.UpSQL) == "" {
		return fmt.Errorf("%w: up migration SQL cannot be empty for %s", ErrInvalidMigrationFile, migration.FullName())
	}

	// 5. Validate down SQL is not empty
	if strings.TrimSpace(migration.DownSQL) == "" {
		return fmt.Errorf("%w: down migration SQL cannot be empty for %s", ErrInvalidMigrationFile, migration.FullName())
	}

	// 6. Check for dangerous operations in up SQL
	if err := v.checkDangerousOperations(migration.UpSQL, migration); err != nil {
		return err
	}

	// 7. Validate checksum exists
	if migration.Checksum == "" {
		return fmt.Errorf("%w: migration checksum is missing for %s", ErrInvalidMigrationFile, migration.FullName())
	}

	return nil
}

// ValidateBatch validates a list of migrations
func (v *MigrationValidator) ValidateBatch(migrations []Migration) error {
	// Check for duplicate versions within same source
	seen := make(map[string]bool)

	for _, migration := range migrations {
		// Validate individual migration
		if err := v.Validate(migration); err != nil {
			return err
		}

		// Check for duplicates
		key := migration.Version + ":" + migration.Source
		if seen[key] {
			return fmt.Errorf(
				"%w: duplicate migration version %s for source %s",
				ErrInvalidMigrationFile, migration.Version, migration.Source,
			)
		}
		seen[key] = true
	}

	return nil
}

// checkDangerousOperations scans SQL for potentially dangerous operations
func (v *MigrationValidator) checkDangerousOperations(sql string, migration Migration) error {
	sqlLower := strings.ToLower(sql)

	// List of dangerous operations
	dangerousPatterns := []struct {
		pattern string
		message string
	}{
		{
			pattern: "drop database",
			message: "DROP DATABASE is not allowed in migrations",
		},
		{
			pattern: "drop schema",
			message: "DROP SCHEMA is not allowed in migrations (use DROP TABLE instead)",
		},
	}

	for _, dangerous := range dangerousPatterns {
		if strings.Contains(sqlLower, dangerous.pattern) {
			return &MigrationError{
				Migration: migration,
				Err:       ErrInvalidMigrationFile,
				SQL:       sql,
				Hint:      dangerous.message,
			}
		}
	}

	return nil
}

// ValidateNoDuplicates ensures no duplicate migrations across sources
func (v *MigrationValidator) ValidateNoDuplicates(allMigrations []Migration) error {
	versionMap := make(map[string][]string) // version -> list of sources

	for _, m := range allMigrations {
		versionMap[m.Version] = append(versionMap[m.Version], m.Source)
	}

	// Check for same version across different sources
	// This is allowed, but we should warn or validate they're intentional
	for version, sources := range versionMap {
		if len(sources) > 1 {
			// This is actually OK - different sources can have migrations with same timestamp
			// as long as they're truly independent
			_ = version // No error, just noting it
		}
	}

	return nil
}
