package query

import (
	"strings"
	"testing"

	"github.com/nicolasbonnici/gorest/database/mysql"
	"github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/database/sqlite"
)

// TestIdentifierInjection_PostgreSQL tests that double-quotes are properly escaped in PostgreSQL
func TestIdentifierInjection_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}

	tests := []struct {
		name            string
		maliciousInput  string
		wantContains    string
		wantNotContains string
	}{
		{
			name:            "double quote injection attempt",
			maliciousInput:  `email" FROM users WHERE admin=true; --`,
			wantContains:    `"email"" FROM users WHERE admin=true; --"`, // Escaped double-quotes
			wantNotContains: `"email" FROM users WHERE admin=true; --"`,  // Unescaped would allow injection
		},
		{
			name:            "column with embedded quote",
			maliciousInput:  `user"name`,
			wantContains:    `"user""name"`, // Escaped
			wantNotContains: `"user"name"`,  // Unescaped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test QuoteIdentifier directly
			quoted := dialect.QuoteIdentifier(tt.maliciousInput)
			if !strings.Contains(quoted, tt.wantContains) {
				t.Errorf("QuoteIdentifier() = %q, want to contain %q", quoted, tt.wantContains)
			}
			if strings.Contains(quoted, tt.wantNotContains) {
				t.Errorf("QuoteIdentifier() = %q, should NOT contain %q (vulnerable)", quoted, tt.wantNotContains)
			}

			// Test in actual query
			sql, _, _ := New(dialect).
				Select(tt.maliciousInput).
				From("users").
				Build()

			if !strings.Contains(sql, tt.wantContains) {
				t.Errorf("Query = %q, want to contain %q", sql, tt.wantContains)
			}
			if strings.Contains(sql, tt.wantNotContains) {
				t.Errorf("Query = %q, should NOT contain %q (vulnerable)", sql, tt.wantNotContains)
			}
		})
	}
}

// TestIdentifierInjection_MySQL tests that backticks are properly escaped in MySQL
func TestIdentifierInjection_MySQL(t *testing.T) {
	dialect := &mysql.MySQLDialect{}

	tests := []struct {
		name            string
		maliciousInput  string
		wantContains    string
		wantNotContains string
	}{
		{
			name:            "backtick injection attempt",
			maliciousInput:  "email` FROM users WHERE admin=1; --",
			wantContains:    "`email`` FROM users WHERE admin=1; --`", // Escaped backticks
			wantNotContains: "`email` FROM users WHERE admin=1; --`",  // Unescaped would allow injection
		},
		{
			name:            "column with embedded backtick",
			maliciousInput:  "user`name",
			wantContains:    "`user``name`", // Escaped
			wantNotContains: "`user`name`",  // Unescaped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test QuoteIdentifier directly
			quoted := dialect.QuoteIdentifier(tt.maliciousInput)
			if !strings.Contains(quoted, tt.wantContains) {
				t.Errorf("QuoteIdentifier() = %q, want to contain %q", quoted, tt.wantContains)
			}
			if strings.Contains(quoted, tt.wantNotContains) {
				t.Errorf("QuoteIdentifier() = %q, should NOT contain %q (vulnerable)", quoted, tt.wantNotContains)
			}

			// Test in actual query
			sql, _, _ := New(dialect).
				Select(tt.maliciousInput).
				From("users").
				Build()

			if !strings.Contains(sql, tt.wantContains) {
				t.Errorf("Query = %q, want to contain %q", sql, tt.wantContains)
			}
			if strings.Contains(sql, tt.wantNotContains) {
				t.Errorf("Query = %q, should NOT contain %q (vulnerable)", sql, tt.wantNotContains)
			}
		})
	}
}

// TestIdentifierInjection_SQLite tests that double-quotes are properly escaped in SQLite
func TestIdentifierInjection_SQLite(t *testing.T) {
	dialect := &sqlite.SQLiteDialect{}

	tests := []struct {
		name            string
		maliciousInput  string
		wantContains    string
		wantNotContains string
	}{
		{
			name:            "double quote injection attempt",
			maliciousInput:  `email" FROM users WHERE admin=1; --`,
			wantContains:    `"email"" FROM users WHERE admin=1; --"`, // Escaped double-quotes
			wantNotContains: `"email" FROM users WHERE admin=1; --"`,  // Unescaped would allow injection
		},
		{
			name:            "column with embedded quote",
			maliciousInput:  `user"name`,
			wantContains:    `"user""name"`, // Escaped
			wantNotContains: `"user"name"`,  // Unescaped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test QuoteIdentifier directly
			quoted := dialect.QuoteIdentifier(tt.maliciousInput)
			if !strings.Contains(quoted, tt.wantContains) {
				t.Errorf("QuoteIdentifier() = %q, want to contain %q", quoted, tt.wantContains)
			}
			if strings.Contains(quoted, tt.wantNotContains) {
				t.Errorf("QuoteIdentifier() = %q, should NOT contain %q (vulnerable)", quoted, tt.wantNotContains)
			}

			// Test in actual query
			sql, _, _ := New(dialect).
				Select(tt.maliciousInput).
				From("users").
				Build()

			if !strings.Contains(sql, tt.wantContains) {
				t.Errorf("Query = %q, want to contain %q", sql, tt.wantContains)
			}
			if strings.Contains(sql, tt.wantNotContains) {
				t.Errorf("Query = %q, should NOT contain %q (vulnerable)", sql, tt.wantNotContains)
			}
		})
	}
}

// TestReturningClauseEscaping tests that RETURNING clause properly escapes all columns
func TestReturningClauseEscaping(t *testing.T) {
	t.Run("PostgreSQL RETURNING with malicious column names", func(t *testing.T) {
		dialect := &postgres.PostgresDialect{}
		sql, _, _ := New(dialect).
			Insert("users").
			Columns("name").
			Values("test").
			Returning(`id", "password`).
			Build()

		// Should escape the embedded quote
		if !strings.Contains(sql, `"id"", ""password"`) {
			t.Errorf("RETURNING clause not properly escaped: %q", sql)
		}
	})

	t.Run("SQLite RETURNING with malicious column names", func(t *testing.T) {
		dialect := &sqlite.SQLiteDialect{}
		sql, _, _ := New(dialect).
			Insert("users").
			Columns("name").
			Values("test").
			Returning(`id", "password`).
			Build()

		// Should escape the embedded quote
		if !strings.Contains(sql, `"id"", ""password"`) {
			t.Errorf("RETURNING clause not properly escaped: %q", sql)
		}
	})
}

// TestOnConflictEscaping tests that ON CONFLICT clause properly escapes columns
func TestOnConflictEscaping(t *testing.T) {
	t.Run("PostgreSQL ON CONFLICT with malicious column", func(t *testing.T) {
		dialect := &postgres.PostgresDialect{}
		result := dialect.OnConflictClause([]string{`email", "admin`}, "DO NOTHING")

		// Should escape the embedded quote
		if !strings.Contains(result, `"email"", ""admin"`) {
			t.Errorf("ON CONFLICT not properly escaped: %q", result)
		}
	})

	t.Run("MySQL ON DUPLICATE KEY with malicious column", func(t *testing.T) {
		dialect := &mysql.MySQLDialect{}
		result := dialect.OnConflictClause([]string{"email` , `admin"}, "")

		// Should escape the embedded backtick
		if !strings.Contains(result, "`email`` , ``admin`") {
			t.Errorf("ON DUPLICATE KEY not properly escaped: %q", result)
		}
	})

	t.Run("SQLite ON CONFLICT with malicious column", func(t *testing.T) {
		dialect := &sqlite.SQLiteDialect{}
		result := dialect.OnConflictClause([]string{`email", "admin`}, "DO NOTHING")

		// Should escape the embedded quote
		if !strings.Contains(result, `"email"", ""admin"`) {
			t.Errorf("ON CONFLICT not properly escaped: %q", result)
		}
	})
}

// TestTableNameEscaping tests that malicious table names are rejected
func TestTableNameEscaping(t *testing.T) {
	t.Run("PostgreSQL table injection", func(t *testing.T) {
		dialect := &postgres.PostgresDialect{}
		_, _, err := New(dialect).
			Select("id").
			From(`users" WHERE 1=1; --`).
			Build()

		// Should reject the malicious table name
		if err == nil {
			t.Error("Expected error for malicious table name, got nil")
		}
		if !strings.Contains(err.Error(), "invalid characters") {
			t.Errorf("Expected 'invalid characters' error, got: %v", err)
		}
	})

	t.Run("MySQL table injection", func(t *testing.T) {
		dialect := &mysql.MySQLDialect{}
		_, _, err := New(dialect).
			Select("id").
			From("users` WHERE 1=1; --").
			Build()

		// Should reject the malicious table name
		if err == nil {
			t.Error("Expected error for malicious table name, got nil")
		}
		if !strings.Contains(err.Error(), "invalid characters") {
			t.Errorf("Expected 'invalid characters' error, got: %v", err)
		}
	})
}
