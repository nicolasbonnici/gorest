package database_test

import (
	"testing"

	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/database/mysql"
	"github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/database/sqlite"
)

func TestBaseDialect_LimitOffset(t *testing.T) {
	dialect := &database.BaseDialect{}

	tests := []struct {
		name     string
		limit    int
		offset   int
		expected string
	}{
		{
			name:     "both limit and offset",
			limit:    10,
			offset:   20,
			expected: "LIMIT 10 OFFSET 20",
		},
		{
			name:     "limit only",
			limit:    10,
			offset:   0,
			expected: "LIMIT 10",
		},
		{
			name:     "offset only",
			limit:    0,
			offset:   20,
			expected: "OFFSET 20",
		},
		{
			name:     "neither limit nor offset",
			limit:    0,
			offset:   0,
			expected: "",
		},
		{
			name:     "large limit and offset",
			limit:    1000,
			offset:   5000,
			expected: "LIMIT 1000 OFFSET 5000",
		},
		{
			name:     "limit 1",
			limit:    1,
			offset:   0,
			expected: "LIMIT 1",
		},
		{
			name:     "offset 1",
			limit:    0,
			offset:   1,
			expected: "OFFSET 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dialect.LimitOffset(tt.limit, tt.offset)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestDialect_SupportsFullJoin(t *testing.T) {
	tests := []struct {
		name     string
		dialect  database.Dialect
		expected bool
	}{
		{
			name:     "PostgreSQL supports full join",
			dialect:  &postgres.PostgresDialect{},
			expected: true,
		},
		{
			name:     "MySQL does not support full join",
			dialect:  &mysql.MySQLDialect{},
			expected: false,
		},
		{
			name:     "SQLite does not support full join",
			dialect:  &sqlite.SQLiteDialect{},
			expected: false,
		},
		{
			name:     "BaseDialect does not support full join",
			dialect:  &database.BaseDialect{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dialect.SupportsFullJoin()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDialect_SupportsWindowFunctions(t *testing.T) {
	tests := []struct {
		name     string
		dialect  database.Dialect
		expected bool
	}{
		{
			name:     "PostgreSQL supports window functions",
			dialect:  &postgres.PostgresDialect{},
			expected: true,
		},
		{
			name:     "MySQL supports window functions (8.0+)",
			dialect:  &mysql.MySQLDialect{},
			expected: true,
		},
		{
			name:     "SQLite supports window functions (3.25+)",
			dialect:  &sqlite.SQLiteDialect{},
			expected: true,
		},
		{
			name:     "BaseDialect does not support window functions",
			dialect:  &database.BaseDialect{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dialect.SupportsWindowFunctions()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDialect_SupportsCTE(t *testing.T) {
	tests := []struct {
		name     string
		dialect  database.Dialect
		expected bool
	}{
		{
			name:     "PostgreSQL supports CTE",
			dialect:  &postgres.PostgresDialect{},
			expected: true,
		},
		{
			name:     "MySQL supports CTE (8.0+)",
			dialect:  &mysql.MySQLDialect{},
			expected: true,
		},
		{
			name:     "SQLite supports CTE",
			dialect:  &sqlite.SQLiteDialect{},
			expected: true,
		},
		{
			name:     "BaseDialect does not support CTE",
			dialect:  &database.BaseDialect{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dialect.SupportsCTE()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDialect_SupportsArrays(t *testing.T) {
	tests := []struct {
		name     string
		dialect  database.Dialect
		expected bool
	}{
		{
			name:     "PostgreSQL supports arrays",
			dialect:  &postgres.PostgresDialect{},
			expected: true,
		},
		{
			name:     "MySQL does not support arrays",
			dialect:  &mysql.MySQLDialect{},
			expected: false,
		},
		{
			name:     "SQLite does not support arrays",
			dialect:  &sqlite.SQLiteDialect{},
			expected: false,
		},
		{
			name:     "BaseDialect does not support arrays",
			dialect:  &database.BaseDialect{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dialect.SupportsArrays()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDialect_UpsertSupport(t *testing.T) {
	tests := []struct {
		name     string
		dialect  database.Dialect
		expected bool
	}{
		{
			name:     "PostgreSQL supports upsert",
			dialect:  &postgres.PostgresDialect{},
			expected: true,
		},
		{
			name:     "MySQL supports upsert",
			dialect:  &mysql.MySQLDialect{},
			expected: true,
		},
		{
			name:     "SQLite supports upsert",
			dialect:  &sqlite.SQLiteDialect{},
			expected: true,
		},
		{
			name:     "BaseDialect does not support upsert",
			dialect:  &database.BaseDialect{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dialect.UpsertSupport()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestPostgresDialect_OnConflictClause(t *testing.T) {
	dialect := &postgres.PostgresDialect{}

	tests := []struct {
		name     string
		columns  []string
		action   string
		expected string
	}{
		{
			name:     "empty columns returns empty string",
			columns:  []string{},
			action:   "DO NOTHING",
			expected: "",
		},
		{
			name:     "single column with DO NOTHING",
			columns:  []string{"email"},
			action:   "DO NOTHING",
			expected: `ON CONFLICT ("email") DO NOTHING`,
		},
		{
			name:     "multiple columns with DO NOTHING",
			columns:  []string{"email", "username"},
			action:   "DO NOTHING",
			expected: `ON CONFLICT ("email", "username") DO NOTHING`,
		},
		{
			name:     "single column with DO UPDATE",
			columns:  []string{"id"},
			action:   "DO UPDATE SET name = EXCLUDED.name",
			expected: `ON CONFLICT ("id") DO UPDATE SET name = EXCLUDED.name`,
		},
		{
			name:     "columns without action",
			columns:  []string{"id"},
			action:   "",
			expected: `ON CONFLICT ("id")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dialect.OnConflictClause(tt.columns, tt.action)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestMySQLDialect_OnConflictClause(t *testing.T) {
	dialect := &mysql.MySQLDialect{}

	tests := []struct {
		name     string
		columns  []string
		action   string
		expected string
	}{
		{
			name:     "empty columns returns empty string",
			columns:  []string{},
			action:   "",
			expected: "",
		},
		{
			name:     "DO NOTHING returns empty string (not supported)",
			columns:  []string{"email"},
			action:   "DO NOTHING",
			expected: "",
		},
		{
			name:     "single column generates ON DUPLICATE KEY UPDATE",
			columns:  []string{"email"},
			action:   "",
			expected: "ON DUPLICATE KEY UPDATE `email` = VALUES(`email`)",
		},
		{
			name:     "multiple columns generates ON DUPLICATE KEY UPDATE",
			columns:  []string{"email", "username"},
			action:   "",
			expected: "ON DUPLICATE KEY UPDATE `email` = VALUES(`email`), `username` = VALUES(`username`)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dialect.OnConflictClause(tt.columns, tt.action)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestSQLiteDialect_OnConflictClause(t *testing.T) {
	dialect := &sqlite.SQLiteDialect{}

	tests := []struct {
		name     string
		columns  []string
		action   string
		expected string
	}{
		{
			name:     "empty columns returns empty string",
			columns:  []string{},
			action:   "DO NOTHING",
			expected: "",
		},
		{
			name:     "single column with DO NOTHING",
			columns:  []string{"email"},
			action:   "DO NOTHING",
			expected: `ON CONFLICT ("email") DO NOTHING`,
		},
		{
			name:     "multiple columns with DO NOTHING",
			columns:  []string{"email", "username"},
			action:   "DO NOTHING",
			expected: `ON CONFLICT ("email", "username") DO NOTHING`,
		},
		{
			name:     "single column with DO UPDATE",
			columns:  []string{"id"},
			action:   "DO UPDATE SET name = excluded.name",
			expected: `ON CONFLICT ("id") DO UPDATE SET name = excluded.name`,
		},
		{
			name:     "columns without action",
			columns:  []string{"id"},
			action:   "",
			expected: `ON CONFLICT ("id")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dialect.OnConflictClause(tt.columns, tt.action)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestBaseDialect_OnConflictClause(t *testing.T) {
	dialect := &database.BaseDialect{}

	result := dialect.OnConflictClause([]string{"id"}, "DO NOTHING")
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}
