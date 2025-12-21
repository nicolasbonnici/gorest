package migrations

import (
	"fmt"
	"strings"
)

// MigrationValidator validates migrations before execution
type MigrationValidator struct{}

func NewMigrationValidator() *MigrationValidator {
	return &MigrationValidator{}
}

func (v *MigrationValidator) Validate(migration Migration) error {
	if err := ValidateTimestamp(migration.Version); err != nil {
		return err
	}

	if strings.TrimSpace(migration.Name) == "" {
		return fmt.Errorf("%w: migration name cannot be empty", ErrInvalidMigrationFile)
	}

	if strings.TrimSpace(migration.Source) == "" {
		return fmt.Errorf("%w: migration source cannot be empty", ErrInvalidMigrationFile)
	}

	if strings.TrimSpace(migration.UpSQL) == "" {
		return fmt.Errorf("%w: up migration SQL cannot be empty for %s", ErrInvalidMigrationFile, migration.FullName())
	}

	if strings.TrimSpace(migration.DownSQL) == "" {
		return fmt.Errorf("%w: down migration SQL cannot be empty for %s", ErrInvalidMigrationFile, migration.FullName())
	}

	if err := v.checkDangerousOperations(migration.UpSQL, migration); err != nil {
		return err
	}

	if migration.Checksum == "" {
		return fmt.Errorf("%w: migration checksum is missing for %s", ErrInvalidMigrationFile, migration.FullName())
	}

	return nil
}

func (v *MigrationValidator) ValidateBatch(migrations []Migration) error {
	seen := make(map[string]bool)

	for _, migration := range migrations {
		if err := v.Validate(migration); err != nil {
			return err
		}

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

func (v *MigrationValidator) checkDangerousOperations(sql string, migration Migration) error {
	sqlLower := strings.ToLower(sql)

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

func (v *MigrationValidator) ValidateNoDuplicates(allMigrations []Migration) error {
	versionMap := make(map[string][]string)

	for _, m := range allMigrations {
		versionMap[m.Version] = append(versionMap[m.Version], m.Source)
	}

	return nil
}
