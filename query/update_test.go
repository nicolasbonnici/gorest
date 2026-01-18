package query

import (
	"strings"
	"testing"

	"github.com/nicolasbonnici/gorest/database/mysql"
	"github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/database/sqlite"
)

func TestUpdateBuilder_SingleColumn(t *testing.T) {
	tests := []struct {
		name           string
		dialect        string
		builder        func() *UpdateBuilder
		expectedSQL    string
		expectedArgs   []any
		expectedParams int
	}{
		{
			name:    "PostgreSQL single column",
			dialect: "postgres",
			builder: func() *UpdateBuilder {
				return New(&postgres.PostgresDialect{}).
					Update("users").
					Set("name", "John Updated").
					Where(Eq("id", 123))
			},
			expectedSQL:    `UPDATE "users" SET "name" = $1 WHERE "id" = $2`,
			expectedArgs:   []any{"John Updated", 123},
			expectedParams: 2,
		},
		{
			name:    "MySQL single column",
			dialect: "mysql",
			builder: func() *UpdateBuilder {
				return New(&mysql.MySQLDialect{}).
					Update("products").
					Set("price", 19.99).
					Where(Eq("id", 456))
			},
			expectedSQL:    "UPDATE `products` SET `price` = ? WHERE `id` = ?",
			expectedArgs:   []any{19.99, 456},
			expectedParams: 2,
		},
		{
			name:    "SQLite single column",
			dialect: "sqlite",
			builder: func() *UpdateBuilder {
				return New(&sqlite.SQLiteDialect{}).
					Update("posts").
					Set("title", "Updated Title").
					Where(Eq("id", 789))
			},
			expectedSQL:    `UPDATE "posts" SET "title" = ? WHERE "id" = ?`,
			expectedArgs:   []any{"Updated Title", 789},
			expectedParams: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.builder().Build()
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}

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

func TestUpdateBuilder_MultipleColumns(t *testing.T) {
	t.Run("PostgreSQL multiple columns", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Update("users").
			Set("name", "John Updated").
			Set("email", "john.new@example.com").
			Set("age", 35).
			Where(Eq("id", 123))

		sql, args, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		expectedSQL := `UPDATE "users" SET "name" = $1, "email" = $2, "age" = $3 WHERE "id" = $4`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 4 {
			t.Errorf("Expected 4 args, got %d", len(args))
		}

		expectedArgs := []any{"John Updated", "john.new@example.com", 35, 123}
		for i, arg := range args {
			if arg != expectedArgs[i] {
				t.Errorf("Arg[%d] mismatch: expected %v, got %v", i, expectedArgs[i], arg)
			}
		}
	})

	t.Run("MySQL multiple columns", func(t *testing.T) {
		builder := New(&mysql.MySQLDialect{}).
			Update("products").
			Set("name", "Product Updated").
			Set("price", 29.99).
			Where(Eq("id", 456))

		sql, args, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		expectedSQL := "UPDATE `products` SET `name` = ?, `price` = ? WHERE `id` = ?"
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(args))
		}
	})
}

func TestUpdateBuilder_SetMap(t *testing.T) {
	t.Run("SetMap single call", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Update("users").
			SetMap(map[string]any{
				"name":  "John Updated",
				"email": "john.new@example.com",
				"age":   35,
			}).
			Where(Eq("id", 123))

		sql, args, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		if !strings.Contains(sql, "UPDATE") {
			t.Error("Expected UPDATE in SQL")
		}

		if len(args) != 4 {
			t.Errorf("Expected 4 args, got %d", len(args))
		}
	})

	t.Run("SetMap combined with Set", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Update("users").
			Set("name", "John").
			SetMap(map[string]any{
				"email": "john@example.com",
				"age":   30,
			}).
			Where(Eq("id", 123))

		sql, args, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		if !strings.Contains(sql, "UPDATE") {
			t.Error("Expected UPDATE in SQL")
		}

		if len(args) != 4 {
			t.Errorf("Expected 4 args, got %d", len(args))
		}
	})
}

func TestUpdateBuilder_WhereConditions(t *testing.T) {
	t.Run("single WHERE condition", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Update("users").
			Set("status", "inactive").
			Where(Eq("id", 123))

		sql, args, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		expectedSQL := `UPDATE "users" SET "status" = $1 WHERE "id" = $2`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})

	t.Run("multiple WHERE conditions with And", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Update("users").
			Set("status", "inactive").
			Where(Eq("id", 123)).
			And(Eq("tenant_id", 456))

		sql, args, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		expectedSQL := `UPDATE "users" SET "status" = $1 WHERE ("id" = $2 AND "tenant_id" = $3)`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(args))
		}
	})

	t.Run("WHERE with OR condition", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Update("users").
			Set("verified", true).
			Where(Eq("email", "john@example.com")).
			Or(Eq("email", "john.doe@example.com"))

		sql, args, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		if !strings.Contains(sql, "OR") {
			t.Error("Expected OR in SQL")
		}

		if len(args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(args))
		}
	})

	t.Run("complex WHERE conditions", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Update("posts").
			Set("status", "archived").
			Where(And(
				Lt("view_count", 100),
				Eq("status", "published"),
			))

		sql, args, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		if !strings.Contains(sql, "AND") {
			t.Error("Expected AND in SQL")
		}

		if len(args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(args))
		}
	})
}

func TestUpdateBuilder_Returning(t *testing.T) {
	t.Run("PostgreSQL with RETURNING", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Update("users").
			Set("name", "John Updated").
			Where(Eq("id", 123)).
			Returning("id", "updated_at")

		sql, args, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		expectedSQL := `UPDATE "users" SET "name" = $1 WHERE "id" = $2 RETURNING "id", "updated_at"`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})

	t.Run("SQLite with RETURNING", func(t *testing.T) {
		builder := New(&sqlite.SQLiteDialect{}).
			Update("users").
			Set("name", "Jane Updated").
			Where(Eq("id", 456)).
			Returning("updated_at")

		sql, args, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		expectedSQL := `UPDATE "users" SET "name" = ? WHERE "id" = ? RETURNING "updated_at"`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})

	t.Run("MySQL without RETURNING support", func(t *testing.T) {
		builder := New(&mysql.MySQLDialect{}).
			Update("users").
			Set("name", "Bob Updated").
			Where(Eq("id", 789)).
			Returning("id")

		sql, args, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		expectedSQL := "UPDATE `users` SET `name` = ? WHERE `id` = ?"
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})
}

func TestUpdateBuilder_ParameterNumbering(t *testing.T) {
	t.Run("PostgreSQL parameter numbering", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Update("users").
			Set("a", 1).
			Set("b", 2).
			Set("c", 3).
			Where(Eq("id", 123))

		sql, _, _ := builder.Build()

		expected := `UPDATE "users" SET "a" = $1, "b" = $2, "c" = $3 WHERE "id" = $4`
		if sql != expected {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expected, sql)
		}
	})

	t.Run("MySQL parameter numbering", func(t *testing.T) {
		builder := New(&mysql.MySQLDialect{}).
			Update("users").
			Set("a", 1).
			Set("b", 2).
			Where(Eq("id", 123)).
			And(Eq("tenant_id", 456))

		sql, _, _ := builder.Build()

		expected := "UPDATE `users` SET `a` = ?, `b` = ? WHERE (`id` = ? AND `tenant_id` = ?)"
		if sql != expected {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expected, sql)
		}
	})
}

func TestUpdateBuilder_FluentInterface(t *testing.T) {
	builder := New(&postgres.PostgresDialect{}).
		Update("users").
		Set("name", "John Updated").
		Set("email", "john.new@example.com").
		Set("age", 35).
		Where(Eq("id", 123)).
		And(Eq("tenant_id", 456)).
		Returning("updated_at")

	sql, args, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if !strings.Contains(sql, "UPDATE") {
		t.Error("Expected UPDATE in SQL")
	}

	if len(args) != 5 {
		t.Errorf("Expected 5 args, got %d", len(args))
	}
}

func TestUpdateBuilder_WithoutWhere(t *testing.T) {
	t.Run("update without WHERE", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Update("users").
			Set("verified", true)

		sql, args, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		expectedSQL := `UPDATE "users" SET "verified" = $1`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got %d", len(args))
		}
	})
}

func TestUpdateBuilder_EdgeCases(t *testing.T) {
	t.Run("update with nil value", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Update("users").
			Set("deleted_at", nil).
			Where(Eq("id", 123))

		sql, args, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		expectedSQL := `UPDATE "users" SET "deleted_at" = $1 WHERE "id" = $2`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}

		if args[0] != nil {
			t.Errorf("Expected first arg to be nil, got %v", args[0])
		}
	})

	t.Run("update same column twice preserves order", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Update("users").
			Set("name", "First").
			Set("email", "email@example.com").
			Set("name", "Second").
			Where(Eq("id", 123))

		sql, args, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		expectedSQL := `UPDATE "users" SET "name" = $1, "email" = $2 WHERE "id" = $3`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if args[0] != "Second" {
			t.Errorf("Expected name to be 'Second', got %v", args[0])
		}
	})

	t.Run("many columns", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Update("records")

		for i := 0; i < 10; i++ {
			builder.Set("col"+string(rune('0'+i)), i)
		}

		builder.Where(Eq("id", 999))

		sql, args, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		if len(args) != 11 {
			t.Errorf("Expected 11 args, got %d", len(args))
		}

		if !strings.Contains(sql, "$11") {
			t.Error("Expected $11 placeholder in SQL")
		}
	})
}
