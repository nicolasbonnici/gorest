package query

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// MaxIdentifierLength is the maximum allowed length for identifiers (PostgreSQL limit)
	MaxIdentifierLength = 63

	// MaxLimitValue is the maximum allowed value for LIMIT clause
	MaxLimitValue = 10000

	// MaxOffsetValue is the maximum allowed value for OFFSET clause
	MaxOffsetValue = 1000000

	// MaxSubqueryDepth is the maximum allowed nesting depth for subqueries
	MaxSubqueryDepth = 3
)

// SQL reserved words (common across PostgreSQL, MySQL, SQLite)
var sqlReservedWords = map[string]bool{
	"SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true, "FROM": true,
	"WHERE": true, "JOIN": true, "LEFT": true, "RIGHT": true, "INNER": true,
	"OUTER": true, "ON": true, "AND": true, "OR": true, "NOT": true, "IN": true,
	"EXISTS": true, "BETWEEN": true, "LIKE": true, "IS": true, "NULL": true,
	"TRUE": true, "FALSE": true, "AS": true, "ORDER": true, "BY": true,
	"GROUP": true, "HAVING": true, "LIMIT": true, "OFFSET": true, "UNION": true,
	"ALL": true, "DISTINCT": true, "CASE": true, "WHEN": true, "THEN": true,
	"ELSE": true, "END": true, "CREATE": true, "DROP": true, "ALTER": true,
	"TABLE": true, "INDEX": true, "VIEW": true, "DATABASE": true, "SCHEMA": true,
	"PRIMARY": true, "KEY": true, "FOREIGN": true, "REFERENCES": true, "UNIQUE": true,
	"CHECK": true, "DEFAULT": true, "CASCADE": true, "RESTRICT": true, "SET": true,
	"VALUES": true, "INTO": true, "RETURNING": true, "WITH": true, "RECURSIVE": true,
	"OVER": true, "PARTITION": true, "ROWS": true, "RANGE": true, "UNBOUNDED": true,
	"PRECEDING": true, "FOLLOWING": true, "CURRENT": true, "ROW": true,
}

// identifierPattern matches valid SQL identifiers: must start with letter or underscore,
// followed by letters, digits, or underscores
var identifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// ValidateIdentifier validates that a string is a safe SQL identifier
func ValidateIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("identifier cannot be empty")
	}

	if len(name) > MaxIdentifierLength {
		return fmt.Errorf("identifier %q exceeds maximum length of %d characters", name, MaxIdentifierLength)
	}

	if !identifierPattern.MatchString(name) {
		return fmt.Errorf("identifier %q contains invalid characters (must match [a-zA-Z_][a-zA-Z0-9_]*)", name)
	}

	// Check for SQL reserved words
	upper := strings.ToUpper(name)
	if sqlReservedWords[upper] {
		return fmt.Errorf("identifier %q is a SQL reserved word", name)
	}

	return nil
}

// ValidateLimit validates LIMIT value
func ValidateLimit(limit int) error {
	if limit < 0 {
		return fmt.Errorf("LIMIT cannot be negative: %d", limit)
	}
	if limit > MaxLimitValue {
		return fmt.Errorf("LIMIT %d exceeds maximum allowed value of %d", limit, MaxLimitValue)
	}
	return nil
}

// ValidateOffset validates OFFSET value
func ValidateOffset(offset int) error {
	if offset < 0 {
		return fmt.Errorf("OFFSET cannot be negative: %d", offset)
	}
	if offset > MaxOffsetValue {
		return fmt.Errorf("OFFSET %d exceeds maximum allowed value of %d", offset, MaxOffsetValue)
	}
	return nil
}

// ValidateWindowFrame validates window frame specification syntax
func ValidateWindowFrame(frame string) error {
	if frame == "" {
		return nil
	}

	frame = strings.ToUpper(strings.TrimSpace(frame))

	// Must start with ROWS or RANGE
	if !strings.HasPrefix(frame, "ROWS ") && !strings.HasPrefix(frame, "RANGE ") {
		return fmt.Errorf("window frame must start with ROWS or RANGE")
	}

	// Common valid patterns
	validPatterns := []string{
		"ROWS UNBOUNDED PRECEDING",
		"ROWS CURRENT ROW",
		"ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW",
		"ROWS BETWEEN CURRENT ROW AND UNBOUNDED FOLLOWING",
		"ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING",
		"RANGE UNBOUNDED PRECEDING",
		"RANGE CURRENT ROW",
		"RANGE BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW",
		"RANGE BETWEEN CURRENT ROW AND UNBOUNDED FOLLOWING",
		"RANGE BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING",
	}

	// Check if frame matches any valid pattern or contains numeric offset
	for _, pattern := range validPatterns {
		if frame == pattern {
			return nil
		}
	}

	// Allow numeric offsets (e.g., "ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING")
	// Basic validation: must contain BETWEEN and AND
	if strings.Contains(frame, "BETWEEN") && strings.Contains(frame, " AND ") {
		// This is a valid BETWEEN clause structure
		return nil
	}

	// Allow simple numeric offset (e.g., "ROWS 5 PRECEDING")
	if regexp.MustCompile(`^(ROWS|RANGE) \d+ (PRECEDING|FOLLOWING)$`).MatchString(frame) {
		return nil
	}

	return fmt.Errorf("invalid window frame specification: %q", frame)
}
