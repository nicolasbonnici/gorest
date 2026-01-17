package query

import (
	"strings"
	"testing"

	"github.com/nicolasbonnici/gorest/database/mysql"
	"github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/database/sqlite"
)

func TestInsertBuilder_SingleRow(t *testing.T) {
	tests := []struct {
		name           string
		dialect        string
		builder        func() *InsertBuilder
		expectedSQL    string
		expectedArgs   []any
		expectedParams int
	}{
		{
			name:    "PostgreSQL single row",
			dialect: "postgres",
			builder: func() *InsertBuilder {
				return New(&postgres.PostgresDialect{}).
					Insert("users").
					Columns("name", "email", "age").
					Values("John Doe", "john@example.com", 30)
			},
			expectedSQL:    `INSERT INTO "users" ("name", "email", "age") VALUES ($1, $2, $3)`,
			expectedArgs:   []any{"John Doe", "john@example.com", 30},
			expectedParams: 3,
		},
		{
			name:    "MySQL single row",
			dialect: "mysql",
			builder: func() *InsertBuilder {
				return New(&mysql.MySQLDialect{}).
					Insert("users").
					Columns("name", "email", "age").
					Values("Jane Doe", "jane@example.com", 25)
			},
			expectedSQL:    "INSERT INTO `users` (`name`, `email`, `age`) VALUES (?, ?, ?)",
			expectedArgs:   []any{"Jane Doe", "jane@example.com", 25},
			expectedParams: 3,
		},
		{
			name:    "SQLite single row",
			dialect: "sqlite",
			builder: func() *InsertBuilder {
				return New(&sqlite.SQLiteDialect{}).
					Insert("users").
					Columns("name", "email").
					Values("Bob Smith", "bob@example.com")
			},
			expectedSQL:    `INSERT INTO "users" ("name", "email") VALUES (?, ?)`,
			expectedArgs:   []any{"Bob Smith", "bob@example.com"},
			expectedParams: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args := tt.builder().Build()

			if sql != tt.expectedSQL {
				t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", tt.expectedSQL, sql)
			}

			if len(args) != tt.expectedParams {
				t.Errorf("Parameter count mismatch: expected %d, got %d", tt.expectedParams, len(args))
			}

			for i, arg := range args {
				if arg != tt.expectedArgs[i] {
					t.Errorf("Arg[%d] mismatch: expected %v, got %v", i, tt.expectedArgs[i], arg)
				}
			}
		})
	}
}

func TestInsertBuilder_BatchInsert(t *testing.T) {
	tests := []struct {
		name           string
		dialect        string
		builder        func() *InsertBuilder
		expectedSQL    string
		expectedArgs   []any
		expectedParams int
	}{
		{
			name:    "PostgreSQL batch insert",
			dialect: "postgres",
			builder: func() *InsertBuilder {
				return New(&postgres.PostgresDialect{}).
					Insert("users").
					Columns("name", "email", "age").
					Values("John Doe", "john@example.com", 30).
					Values("Jane Doe", "jane@example.com", 25).
					Values("Bob Smith", "bob@example.com", 35)
			},
			expectedSQL:    `INSERT INTO "users" ("name", "email", "age") VALUES ($1, $2, $3), ($4, $5, $6), ($7, $8, $9)`,
			expectedArgs:   []any{"John Doe", "john@example.com", 30, "Jane Doe", "jane@example.com", 25, "Bob Smith", "bob@example.com", 35},
			expectedParams: 9,
		},
		{
			name:    "MySQL batch insert",
			dialect: "mysql",
			builder: func() *InsertBuilder {
				return New(&mysql.MySQLDialect{}).
					Insert("products").
					Columns("name", "price").
					Values("Product A", 19.99).
					Values("Product B", 29.99)
			},
			expectedSQL:    "INSERT INTO `products` (`name`, `price`) VALUES (?, ?), (?, ?)",
			expectedArgs:   []any{"Product A", 19.99, "Product B", 29.99},
			expectedParams: 4,
		},
		{
			name:    "SQLite batch insert",
			dialect: "sqlite",
			builder: func() *InsertBuilder {
				return New(&sqlite.SQLiteDialect{}).
					Insert("tags").
					Columns("name").
					Values("tag1").
					Values("tag2").
					Values("tag3").
					Values("tag4")
			},
			expectedSQL:    `INSERT INTO "tags" ("name") VALUES (?), (?), (?), (?)`,
			expectedArgs:   []any{"tag1", "tag2", "tag3", "tag4"},
			expectedParams: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args := tt.builder().Build()

			if sql != tt.expectedSQL {
				t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", tt.expectedSQL, sql)
			}

			if len(args) != tt.expectedParams {
				t.Errorf("Parameter count mismatch: expected %d, got %d", tt.expectedParams, len(args))
			}

			for i, arg := range args {
				if arg != tt.expectedArgs[i] {
					t.Errorf("Arg[%d] mismatch: expected %v, got %v", i, tt.expectedArgs[i], arg)
				}
			}
		})
	}
}

func TestInsertBuilder_ValuesMap(t *testing.T) {
	t.Run("single ValuesMap", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Insert("users").
			Columns("name", "email", "age").
			ValuesMap(map[string]any{
				"name":  "John Doe",
				"email": "john@example.com",
				"age":   30,
			})

		sql, args := builder.Build()

		expectedSQL := `INSERT INTO "users" ("name", "email", "age") VALUES ($1, $2, $3)`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(args))
		}
	})

	t.Run("ValuesMap with auto column extraction", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Insert("users").
			ValuesMap(map[string]any{
				"name":  "John Doe",
				"email": "john@example.com",
			})

		sql, args := builder.Build()

		if !strings.Contains(sql, `INSERT INTO "users"`) {
			t.Errorf("Expected INSERT INTO users, got: %s", sql)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})

	t.Run("multiple ValuesMap calls", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Insert("users").
			Columns("name", "email").
			ValuesMap(map[string]any{"name": "John", "email": "john@example.com"}).
			ValuesMap(map[string]any{"name": "Jane", "email": "jane@example.com"})

		sql, args := builder.Build()

		expectedSQL := `INSERT INTO "users" ("name", "email") VALUES ($1, $2), ($3, $4)`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 4 {
			t.Errorf("Expected 4 args, got %d", len(args))
		}
	})
}

func TestInsertBuilder_Returning(t *testing.T) {
	t.Run("PostgreSQL with RETURNING", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Insert("users").
			Columns("name", "email").
			Values("John Doe", "john@example.com").
			Returning("id", "created_at")

		sql, args := builder.Build()

		expectedSQL := `INSERT INTO "users" ("name", "email") VALUES ($1, $2) RETURNING id, created_at`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})

	t.Run("SQLite with RETURNING", func(t *testing.T) {
		builder := New(&sqlite.SQLiteDialect{}).
			Insert("users").
			Columns("name", "email").
			Values("Jane Doe", "jane@example.com").
			Returning("id")

		sql, args := builder.Build()

		expectedSQL := `INSERT INTO "users" ("name", "email") VALUES (?, ?) RETURNING "id"`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})

	t.Run("MySQL without RETURNING support", func(t *testing.T) {
		builder := New(&mysql.MySQLDialect{}).
			Insert("users").
			Columns("name", "email").
			Values("Bob Smith", "bob@example.com").
			Returning("id")

		sql, args := builder.Build()

		// MySQL doesn't support RETURNING, so it should be omitted
		expectedSQL := "INSERT INTO `users` (`name`, `email`) VALUES (?, ?)"
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})
}

func TestInsertBuilder_ParameterNumbering(t *testing.T) {
	t.Run("PostgreSQL parameter numbering", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Insert("users").
			Columns("a", "b", "c").
			Values(1, 2, 3).
			Values(4, 5, 6)

		sql, _ := builder.Build()

		expected := `INSERT INTO "users" ("a", "b", "c") VALUES ($1, $2, $3), ($4, $5, $6)`
		if sql != expected {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expected, sql)
		}
	})

	t.Run("MySQL parameter numbering", func(t *testing.T) {
		builder := New(&mysql.MySQLDialect{}).
			Insert("users").
			Columns("a", "b").
			Values(1, 2).
			Values(3, 4).
			Values(5, 6)

		sql, _ := builder.Build()

		expected := "INSERT INTO `users` (`a`, `b`) VALUES (?, ?), (?, ?), (?, ?)"
		if sql != expected {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expected, sql)
		}
	})
}

func TestInsertBuilder_FluentInterface(t *testing.T) {
	builder := New(&postgres.PostgresDialect{}).
		Insert("users").
		Columns("name", "email", "age").
		Values("John Doe", "john@example.com", 30).
		Values("Jane Doe", "jane@example.com", 25).
		Returning("id")

	sql, args := builder.Build()

	if !strings.Contains(sql, "INSERT INTO") {
		t.Error("Expected INSERT INTO in SQL")
	}

	if len(args) != 6 {
		t.Errorf("Expected 6 args, got %d", len(args))
	}
}

func TestInsertBuilder_EdgeCases(t *testing.T) {
	t.Run("single column insert", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Insert("tags").
			Columns("name").
			Values("tag1")

		sql, args := builder.Build()

		expectedSQL := `INSERT INTO "tags" ("name") VALUES ($1)`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got %d", len(args))
		}
	})

	t.Run("many columns", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Insert("records").
			Columns("a", "b", "c", "d", "e", "f", "g", "h", "i", "j").
			Values(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)

		sql, args := builder.Build()

		if !strings.Contains(sql, "$10") {
			t.Error("Expected $10 placeholder in SQL")
		}

		if len(args) != 10 {
			t.Errorf("Expected 10 args, got %d", len(args))
		}
	})

	t.Run("many rows", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Insert("items").
			Columns("id", "name")

		for i := 1; i <= 100; i++ {
			builder.Values(i, "item")
		}

		sql, args := builder.Build()

		if len(args) != 200 {
			t.Errorf("Expected 200 args, got %d", len(args))
		}

		if !strings.Contains(sql, "$200") {
			t.Error("Expected $200 placeholder in SQL")
		}
	})
}
