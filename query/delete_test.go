package query

import (
	"strings"
	"testing"

	"github.com/nicolasbonnici/gorest/database/mysql"
	"github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/database/sqlite"
)

func TestDeleteBuilder_SingleCondition(t *testing.T) {
	tests := []struct {
		name           string
		dialect        string
		builder        func() *DeleteBuilder
		expectedSQL    string
		expectedArgs   []any
		expectedParams int
	}{
		{
			name:    "PostgreSQL single condition",
			dialect: "postgres",
			builder: func() *DeleteBuilder {
				return New(&postgres.PostgresDialect{}).
					Delete("users").
					Where(Eq("id", 123))
			},
			expectedSQL:    `DELETE FROM "users" WHERE "id" = $1`,
			expectedArgs:   []any{123},
			expectedParams: 1,
		},
		{
			name:    "MySQL single condition",
			dialect: "mysql",
			builder: func() *DeleteBuilder {
				return New(&mysql.MySQLDialect{}).
					Delete("products").
					Where(Eq("id", 456))
			},
			expectedSQL:    "DELETE FROM `products` WHERE `id` = ?",
			expectedArgs:   []any{456},
			expectedParams: 1,
		},
		{
			name:    "SQLite single condition",
			dialect: "sqlite",
			builder: func() *DeleteBuilder {
				return New(&sqlite.SQLiteDialect{}).
					Delete("posts").
					Where(Eq("id", 789))
			},
			expectedSQL:    `DELETE FROM "posts" WHERE "id" = ?`,
			expectedArgs:   []any{789},
			expectedParams: 1,
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

func TestDeleteBuilder_MultipleConditions(t *testing.T) {
	t.Run("PostgreSQL multiple conditions with And", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Delete("users").
			Where(Eq("id", 123)).
			And(Eq("tenant_id", 456))

		sql, args := builder.Build()

		expectedSQL := `DELETE FROM "users" WHERE ("id" = $1 AND "tenant_id" = $2)`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}

		expectedArgs := []any{123, 456}
		for i, arg := range args {
			if arg != expectedArgs[i] {
				t.Errorf("Arg[%d] mismatch: expected %v, got %v", i, expectedArgs[i], arg)
			}
		}
	})

	t.Run("MySQL multiple conditions", func(t *testing.T) {
		builder := New(&mysql.MySQLDialect{}).
			Delete("products").
			Where(Eq("status", "inactive")).
			And(Lt("price", 10.0))

		sql, args := builder.Build()

		expectedSQL := "DELETE FROM `products` WHERE (`status` = ? AND `price` < ?)"
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})
}

func TestDeleteBuilder_OrCondition(t *testing.T) {
	t.Run("DELETE with OR condition", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Delete("users").
			Where(Eq("status", "inactive")).
			Or(Eq("status", "suspended"))

		sql, args := builder.Build()

		if !strings.Contains(sql, "OR") {
			t.Error("Expected OR in SQL")
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})
}

func TestDeleteBuilder_ComplexConditions(t *testing.T) {
	t.Run("complex WHERE conditions", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Delete("posts").
			Where(And(
				Lt("view_count", 100),
				Eq("status", "draft"),
			))

		sql, args := builder.Build()

		if !strings.Contains(sql, "AND") {
			t.Error("Expected AND in SQL")
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})

	t.Run("IN condition", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Delete("users").
			Where(In("status", "inactive", "suspended", "banned"))

		sql, args := builder.Build()

		expectedSQL := `DELETE FROM "users" WHERE "status" IN ($1, $2, $3)`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(args))
		}
	})
}

func TestDeleteBuilder_Returning(t *testing.T) {
	t.Run("PostgreSQL with RETURNING", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Delete("users").
			Where(Eq("id", 123)).
			Returning("id", "deleted_at")

		sql, args := builder.Build()

		expectedSQL := `DELETE FROM "users" WHERE "id" = $1 RETURNING id, deleted_at`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got %d", len(args))
		}
	})

	t.Run("SQLite with RETURNING", func(t *testing.T) {
		builder := New(&sqlite.SQLiteDialect{}).
			Delete("users").
			Where(Eq("id", 456)).
			Returning("id")

		sql, args := builder.Build()

		expectedSQL := `DELETE FROM "users" WHERE "id" = ? RETURNING "id"`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got %d", len(args))
		}
	})

	t.Run("MySQL without RETURNING support", func(t *testing.T) {
		builder := New(&mysql.MySQLDialect{}).
			Delete("users").
			Where(Eq("id", 789)).
			Returning("id")

		sql, args := builder.Build()

		expectedSQL := "DELETE FROM `users` WHERE `id` = ?"
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got %d", len(args))
		}
	})
}

func TestDeleteBuilder_ParameterNumbering(t *testing.T) {
	t.Run("PostgreSQL parameter numbering", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Delete("users").
			Where(Eq("id", 123)).
			And(Eq("tenant_id", 456)).
			And(Eq("status", "inactive"))

		sql, _ := builder.Build()

		expected := `DELETE FROM "users" WHERE ("id" = $1 AND "tenant_id" = $2 AND "status" = $3)`
		if sql != expected {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expected, sql)
		}
	})

	t.Run("MySQL parameter numbering", func(t *testing.T) {
		builder := New(&mysql.MySQLDialect{}).
			Delete("products").
			Where(In("status", "draft", "inactive", "archived"))

		sql, _ := builder.Build()

		expected := "DELETE FROM `products` WHERE `status` IN (?, ?, ?)"
		if sql != expected {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expected, sql)
		}
	})
}

func TestDeleteBuilder_FluentInterface(t *testing.T) {
	builder := New(&postgres.PostgresDialect{}).
		Delete("users").
		Where(Eq("id", 123)).
		And(Eq("tenant_id", 456)).
		Returning("id")

	sql, args := builder.Build()

	if !strings.Contains(sql, "DELETE FROM") {
		t.Error("Expected DELETE FROM in SQL")
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}
}

func TestDeleteBuilder_WithoutWhere(t *testing.T) {
	t.Run("delete without WHERE", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Delete("temp_data")

		sql, args := builder.Build()

		expectedSQL := `DELETE FROM "temp_data"`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 0 {
			t.Errorf("Expected 0 args, got %d", len(args))
		}
	})
}

func TestDeleteBuilder_EdgeCases(t *testing.T) {
	t.Run("delete with nil in condition", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Delete("users").
			Where(IsNull("deleted_at"))

		sql, args := builder.Build()

		expectedSQL := `DELETE FROM "users" WHERE "deleted_at" IS NULL`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 0 {
			t.Errorf("Expected 0 args, got %d", len(args))
		}
	})

	t.Run("delete with BETWEEN", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Delete("sessions").
			Where(Between("created_at", "2024-01-01", "2024-12-31"))

		sql, args := builder.Build()

		expectedSQL := `DELETE FROM "sessions" WHERE "created_at" BETWEEN $1 AND $2`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})

	t.Run("delete with LIKE", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Delete("users").
			Where(Like("email", "%@spam.com"))

		sql, args := builder.Build()

		expectedSQL := `DELETE FROM "users" WHERE "email" LIKE $1`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got %d", len(args))
		}
	})
}
