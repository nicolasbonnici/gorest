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

			// Test in actual query - validation should REJECT malicious identifiers
			_, _, err := New(dialect).
				Select(tt.maliciousInput).
				From("users").
				Build()

			if err == nil {
				t.Errorf("Expected validation error for malicious input %q, but got none", tt.maliciousInput)
			}
			if err != nil && !strings.Contains(err.Error(), "invalid characters") {
				t.Errorf("Expected 'invalid characters' validation error, got: %v", err)
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

			// Test in actual query - validation should REJECT malicious identifiers
			_, _, err := New(dialect).
				Select(tt.maliciousInput).
				From("users").
				Build()

			if err == nil {
				t.Errorf("Expected validation error for malicious input %q, but got none", tt.maliciousInput)
			}
			if err != nil && !strings.Contains(err.Error(), "invalid characters") {
				t.Errorf("Expected 'invalid characters' validation error, got: %v", err)
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

			// Test in actual query - validation should REJECT malicious identifiers
			_, _, err := New(dialect).
				Select(tt.maliciousInput).
				From("users").
				Build()

			if err == nil {
				t.Errorf("Expected validation error for malicious input %q, but got none", tt.maliciousInput)
			}
			if err != nil && !strings.Contains(err.Error(), "invalid characters") {
				t.Errorf("Expected 'invalid characters' validation error, got: %v", err)
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

		if err == nil {
			t.Error("Expected error for malicious table name, got nil")
		}
		if !strings.Contains(err.Error(), "invalid characters") {
			t.Errorf("Expected 'invalid characters' error, got: %v", err)
		}
	})
}

// TestColumnNameInjection_PostgreSQL tests that reserved words and SQL injection attempts
// in column names are rejected
func TestColumnNameInjection_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}

	tests := []struct {
		name       string
		columnName string
		errorMsg   string
	}{
		{"reserved word SELECT", "SELECT", "reserved word"},
		{"reserved word DROP", "DROP", "reserved word"},
		{"reserved word DELETE", "DELETE", "reserved word"},
		{"SQL injection attempt", `id"; DROP TABLE users--`, "invalid characters"},
		{"special characters semicolon", "col;name", "invalid characters"},
		{"special characters quotes", `col"name`, "invalid characters"},
		{"SQL comment injection", "id--comment", "invalid characters"},
		{"space in column", "col name", "invalid characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := New(dialect).
				Select(tt.columnName).
				From("users").
				Build()

			if err == nil {
				t.Errorf("Expected error for malicious column name %q", tt.columnName)
			}
			if !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
			}
		})
	}
}

// TestColumnNameInjection_MySQL tests column name validation for MySQL
func TestColumnNameInjection_MySQL(t *testing.T) {
	dialect := &mysql.MySQLDialect{}

	tests := []struct {
		name       string
		columnName string
		errorMsg   string
	}{
		{"reserved word SELECT", "SELECT", "reserved word"},
		{"reserved word DROP", "DROP", "reserved word"},
		{"SQL injection with backtick", "id`; DROP TABLE users--", "invalid characters"},
		{"special characters", "col$name", "invalid characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := New(dialect).
				Select(tt.columnName).
				From("users").
				Build()

			if err == nil {
				t.Errorf("Expected error for malicious column name %q", tt.columnName)
			}
			if !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
			}
		})
	}
}

// TestColumnNameInjection_SQLite tests column name validation for SQLite
func TestColumnNameInjection_SQLite(t *testing.T) {
	dialect := &sqlite.SQLiteDialect{}

	tests := []struct {
		name       string
		columnName string
		errorMsg   string
	}{
		{"reserved word UPDATE", "UPDATE", "reserved word"},
		{"reserved word WHERE", "WHERE", "reserved word"},
		{"SQL injection", `id" OR 1=1--`, "invalid characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := New(dialect).
				Select(tt.columnName).
				From("users").
				Build()

			if err == nil {
				t.Errorf("Expected error for malicious column name %q", tt.columnName)
			}
			if !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
			}
		})
	}
}

// TestJoinTableInjection tests that malicious table names in JOIN clauses are rejected
func TestJoinTableInjection(t *testing.T) {
	dialect := &postgres.PostgresDialect{}

	tests := []struct {
		name      string
		tableName string
		errorMsg  string
	}{
		{"reserved word SELECT", "SELECT", "reserved word"},
		{"SQL injection", `posts" WHERE 1=1--`, "invalid characters"},
		{"special characters", "table$name", "invalid characters"},
		{"semicolon injection", "posts; DROP TABLE users--", "invalid characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := New(dialect).
				Select("id").
				From("users").
				InnerJoin(tt.tableName, Eq("id", 1)).
				Build()

			if err == nil {
				t.Errorf("Expected error for malicious join table %q", tt.tableName)
			}
			if !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
			}
		})
	}
}

// TestAliasInjection tests that malicious aliases are rejected
func TestAliasInjection(t *testing.T) {
	t.Run("Table alias with reserved word", func(t *testing.T) {
		dialect := &postgres.PostgresDialect{}
		_, _, err := New(dialect).
			Select("id").
			From("users").
			As("DROP").
			Build()

		if err == nil {
			t.Error("Expected error for reserved word as alias")
		}
		if !strings.Contains(err.Error(), "reserved word") {
			t.Errorf("Expected 'reserved word' error, got: %v", err)
		}
	})

	t.Run("Table alias with special characters", func(t *testing.T) {
		dialect := &postgres.PostgresDialect{}
		_, _, err := New(dialect).
			Select("id").
			From("users").
			As("u; DROP TABLE users--").
			Build()

		if err == nil {
			t.Error("Expected error for malicious alias")
		}
		if !strings.Contains(err.Error(), "invalid characters") {
			t.Errorf("Expected 'invalid characters' error, got: %v", err)
		}
	})

	t.Run("Column alias with reserved word", func(t *testing.T) {
		dialect := &postgres.PostgresDialect{}
		_, _, err := New(dialect).
			Select().
			SelectExpr(As(Col("id"), "SELECT")).
			From("users").
			Build()

		if err == nil {
			t.Error("Expected error for reserved word as column alias")
		}
		if !strings.Contains(err.Error(), "reserved word") {
			t.Errorf("Expected 'reserved word' error, got: %v", err)
		}
	})

	t.Run("Column alias with SQL injection", func(t *testing.T) {
		dialect := &mysql.MySQLDialect{}
		_, _, err := New(dialect).
			Select().
			SelectExpr(As(Col("name"), "n` FROM users WHERE admin=1--")).
			From("users").
			Build()

		if err == nil {
			t.Error("Expected error for malicious column alias")
		}
		if !strings.Contains(err.Error(), "invalid characters") {
			t.Errorf("Expected 'invalid characters' error, got: %v", err)
		}
	})
}

// TestWhereConditionInvalidColumn tests that condition functions validate column names
func TestWhereConditionInvalidColumn(t *testing.T) {
	dialect := &postgres.PostgresDialect{}

	tests := []struct {
		name      string
		condition func() Condition
		errorMsg  string
	}{
		{
			name:      "Eq with reserved word",
			condition: func() Condition { return Eq("SELECT", "value") },
			errorMsg:  "reserved word",
		},
		{
			name:      "Ne with SQL injection",
			condition: func() Condition { return Ne(`id"; DROP TABLE users--`, "value") },
			errorMsg:  "invalid characters",
		},
		{
			name:      "Gt with special characters",
			condition: func() Condition { return Gt("col$name", 10) },
			errorMsg:  "invalid characters",
		},
		{
			name:      "Lt with spaces",
			condition: func() Condition { return Lt("col name", 5) },
			errorMsg:  "invalid characters",
		},
		{
			name:      "Like with reserved word",
			condition: func() Condition { return Like("WHERE", "%test%") },
			errorMsg:  "reserved word",
		},
		{
			name:      "IsNull with injection attempt",
			condition: func() Condition { return IsNull("id; DROP TABLE users--") },
			errorMsg:  "invalid characters",
		},
		{
			name:      "In with reserved word",
			condition: func() Condition { return In("DELETE", 1, 2, 3) },
			errorMsg:  "reserved word",
		},
		{
			name:      "Between with malicious column",
			condition: func() Condition { return Between(`id" OR 1=1--`, 1, 10) },
			errorMsg:  "invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := New(dialect).
				Select("id").
				From("users").
				Where(tt.condition()).
				Build()

			if err == nil {
				t.Error("Expected error for invalid column name in condition")
			}
			if !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
			}
		})
	}
}

// TestColumnComparisonValidation tests that column comparison conditions validate both columns
func TestColumnComparisonValidation(t *testing.T) {
	dialect := &postgres.PostgresDialect{}

	tests := []struct {
		name      string
		condition func() Condition
		errorMsg  string
	}{
		{
			name:      "ColEq with reserved word in col1",
			condition: func() Condition { return ColEq("SELECT", "other_col") },
			errorMsg:  "reserved word",
		},
		{
			name:      "ColEq with reserved word in col2",
			condition: func() Condition { return ColEq("valid_col", "DROP") },
			errorMsg:  "reserved word",
		},
		{
			name:      "ColNe with SQL injection in col1",
			condition: func() Condition { return ColNe(`id"; DROP TABLE users--`, "other") },
			errorMsg:  "invalid characters",
		},
		{
			name:      "ColGt with special characters",
			condition: func() Condition { return ColGt("col1", "col$2") },
			errorMsg:  "invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := New(dialect).
				Select("id").
				From("users").
				Where(tt.condition()).
				Build()

			if err == nil {
				t.Error("Expected error for invalid column name in column comparison")
			}
			if !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
			}
		})
	}
}

// TestOrderByInjection tests that ORDER BY clauses validate column names
func TestOrderByInjection(t *testing.T) {
	dialect := &postgres.PostgresDialect{}

	tests := []struct {
		name       string
		columnName string
		errorMsg   string
	}{
		{"reserved word", "SELECT", "reserved word"},
		{"SQL injection", "id; DROP TABLE users--", "invalid characters"},
		{"special characters", "col@name", "invalid characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := New(dialect).
				Select("id").
				From("users").
				OrderBy(tt.columnName, ASC).
				Build()

			if err == nil {
				t.Errorf("Expected error for malicious ORDER BY column %q", tt.columnName)
			}
			if !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
			}
		})
	}
}

// TestGroupByInjection tests that GROUP BY clauses validate column names
func TestGroupByInjection(t *testing.T) {
	dialect := &postgres.PostgresDialect{}

	tests := []struct {
		name       string
		columnName string
		errorMsg   string
	}{
		{"reserved word", "UPDATE", "reserved word"},
		{"SQL injection", `status" OR 1=1--`, "invalid characters"},
		{"semicolon", "status; DROP TABLE users--", "invalid characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := New(dialect).
				Select("id", "COUNT(*)").
				From("users").
				GroupBy(tt.columnName).
				Build()

			if err == nil {
				t.Errorf("Expected error for malicious GROUP BY column %q", tt.columnName)
			}
			if !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
			}
		})
	}
}

// TestInsertColumnInjection tests that INSERT column names are validated
func TestInsertColumnInjection(t *testing.T) {
	dialect := &postgres.PostgresDialect{}

	tests := []struct {
		name       string
		columnName string
		errorMsg   string
	}{
		{"reserved word", "FROM", "reserved word"},
		{"SQL injection", `name", "admin"=true--`, "invalid characters"},
		{"special characters", "col;name", "invalid characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := New(dialect).
				Insert("users").
				Columns(tt.columnName).
				Values("test").
				Build()

			if err == nil {
				t.Errorf("Expected error for malicious column name %q in INSERT", tt.columnName)
			}
			if !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
			}
		})
	}
}

// TestUpdateColumnInjection tests that UPDATE SET clauses validate column names
func TestUpdateColumnInjection(t *testing.T) {
	dialect := &postgres.PostgresDialect{}

	tests := []struct {
		name       string
		columnName string
		errorMsg   string
	}{
		{"reserved word", "JOIN", "reserved word"},
		{"SQL injection", `name" WHERE admin=true--`, "invalid characters"},
		{"special characters", "col@name", "invalid characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := New(dialect).
				Update("users").
				Set(tt.columnName, "value").
				Build()

			if err == nil {
				t.Errorf("Expected error for malicious column name %q in UPDATE", tt.columnName)
			}
			if !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
			}
		})
	}
}

// TestDeleteTableValidation tests that DELETE validates table names
func TestDeleteTableValidation(t *testing.T) {
	dialect := &postgres.PostgresDialect{}

	tests := []struct {
		name      string
		tableName string
		errorMsg  string
	}{
		{"reserved word", "CREATE", "reserved word"},
		{"SQL injection", "users; DROP TABLE sessions--", "invalid characters"},
		{"special characters", "table@name", "invalid characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := New(dialect).
				Delete(tt.tableName).
				Build()

			if err == nil {
				t.Errorf("Expected error for malicious table name %q in DELETE", tt.tableName)
			}
			if !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
			}
		})
	}
}
