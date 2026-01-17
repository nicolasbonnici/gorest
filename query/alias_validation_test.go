package query

import (
	"strings"
	"testing"

	"github.com/nicolasbonnici/gorest/database/postgres"
)

func TestSelectBuilder_TableAliasValidation(t *testing.T) {
	tests := []struct {
		name        string
		alias       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid alias",
			alias:       "u",
			expectError: false,
		},
		{
			name:        "valid alias with underscore",
			alias:       "user_data",
			expectError: false,
		},
		{
			name:        "valid alias with numbers",
			alias:       "user123",
			expectError: false,
		},
		{
			name:        "reserved word SELECT",
			alias:       "SELECT",
			expectError: true,
			errorMsg:    "is a SQL reserved word",
		},
		{
			name:        "reserved word FROM",
			alias:       "FROM",
			expectError: true,
			errorMsg:    "is a SQL reserved word",
		},
		{
			name:        "reserved word WHERE",
			alias:       "WHERE",
			expectError: true,
			errorMsg:    "is a SQL reserved word",
		},
		{
			name:        "reserved word JOIN",
			alias:       "JOIN",
			expectError: true,
			errorMsg:    "is a SQL reserved word",
		},
		{
			name:        "invalid characters - space",
			alias:       "user alias",
			expectError: true,
			errorMsg:    "contains invalid characters",
		},
		{
			name:        "invalid characters - dash",
			alias:       "user-alias",
			expectError: true,
			errorMsg:    "contains invalid characters",
		},
		{
			name:        "invalid characters - dot",
			alias:       "user.alias",
			expectError: true,
			errorMsg:    "contains invalid characters",
		},
		{
			name:        "starts with number",
			alias:       "123user",
			expectError: true,
			errorMsg:    "contains invalid characters",
		},
		{
			name:        "empty alias",
			alias:       "",
			expectError: true,
			errorMsg:    "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New(&postgres.PostgresDialect{}).
				Select("id", "name").
				From("users").
				As(tt.alias)

			_, _, err := builder.Build()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for alias %q, but got none", tt.alias)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for alias %q, but got: %v", tt.alias, err)
				}
			}
		})
	}
}

func TestSelectBuilder_JoinAliasValidation(t *testing.T) {
	tests := []struct {
		name        string
		alias       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid join alias",
			alias:       "p",
			expectError: false,
		},
		{
			name:        "valid join alias with underscore",
			alias:       "post_data",
			expectError: false,
		},
		{
			name:        "reserved word as join alias",
			alias:       "SELECT",
			expectError: true,
			errorMsg:    "is a SQL reserved word",
		},
		{
			name:        "invalid characters in join alias",
			alias:       "post-alias",
			expectError: true,
			errorMsg:    "contains invalid characters",
		},
		{
			name:        "empty join alias",
			alias:       "",
			expectError: true,
			errorMsg:    "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New(&postgres.PostgresDialect{}).
				Select("u.id", "p.title").
				From("users").As("u").
				JoinAs("posts", tt.alias, ColEq("u.id", "p.user_id"))

			_, _, err := builder.Build()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for join alias %q, but got none", tt.alias)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for join alias %q, but got: %v", tt.alias, err)
				}
			}
		})
	}
}

func TestSelectBuilder_LeftJoinAliasValidation(t *testing.T) {
	tests := []struct {
		name        string
		alias       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid left join alias",
			alias:       "c",
			expectError: false,
		},
		{
			name:        "reserved word in left join alias",
			alias:       "WHERE",
			expectError: true,
			errorMsg:    "is a SQL reserved word",
		},
		{
			name:        "invalid characters in left join alias",
			alias:       "comment.alias",
			expectError: true,
			errorMsg:    "contains invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New(&postgres.PostgresDialect{}).
				Select("u.id", "c.content").
				From("users").As("u").
				LeftJoinAs("comments", tt.alias, ColEq("u.id", "c.user_id"))

			_, _, err := builder.Build()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for left join alias %q, but got none", tt.alias)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for left join alias %q, but got: %v", tt.alias, err)
				}
			}
		})
	}
}

func TestSelectBuilder_TableValidationInJoinAs(t *testing.T) {
	t.Run("invalid table name in JoinAs", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Select("*").
			From("users").
			JoinAs("SELECT", "p", ColEq("users.id", "p.user_id"))

		_, _, err := builder.Build()

		if err == nil {
			t.Error("Expected error for reserved word table name in JoinAs")
		} else if !strings.Contains(err.Error(), "is a SQL reserved word") {
			t.Errorf("Expected error about reserved word, got: %v", err)
		}
	})

	t.Run("invalid table name in LeftJoinAs", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Select("*").
			From("users").
			LeftJoinAs("FROM", "p", ColEq("users.id", "p.user_id"))

		_, _, err := builder.Build()

		if err == nil {
			t.Error("Expected error for reserved word table name in LeftJoinAs")
		} else if !strings.Contains(err.Error(), "is a SQL reserved word") {
			t.Errorf("Expected error about reserved word, got: %v", err)
		}
	})
}

func TestSelectBuilder_ValidAliasesWorkCorrectly(t *testing.T) {
	t.Run("table alias generates correct SQL", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Select("u.id", "u.name").
			From("users").
			As("u").
			Where(Eq("u.status", "active"))

		sql, args, err := builder.Build()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		expectedSQL := `SELECT "u.id", "u.name" FROM "users" AS "u" WHERE "u.status" = $1`
		if sql != expectedSQL {
			t.Errorf("Expected SQL: %s\nGot: %s", expectedSQL, sql)
		}

		if len(args) != 1 || args[0] != "active" {
			t.Errorf("Expected args: [active], got: %v", args)
		}
	})

	t.Run("join alias generates correct SQL", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Select("u.id", "p.title").
			From("users").As("u").
			JoinAs("posts", "p", ColEq("u.id", "p.user_id"))

		sql, _, err := builder.Build()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		expectedSQL := `SELECT "u.id", "p.title" FROM "users" AS "u" INNER JOIN "posts" AS "p" ON "u.id" = "p.user_id"`
		if sql != expectedSQL {
			t.Errorf("Expected SQL: %s\nGot: %s", expectedSQL, sql)
		}
	})

	t.Run("left join alias generates correct SQL", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Select("u.id", "c.content").
			From("users").As("u").
			LeftJoinAs("comments", "c", ColEq("u.id", "c.user_id"))

		sql, _, err := builder.Build()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		expectedSQL := `SELECT "u.id", "c.content" FROM "users" AS "u" LEFT JOIN "comments" AS "c" ON "u.id" = "c.user_id"`
		if sql != expectedSQL {
			t.Errorf("Expected SQL: %s\nGot: %s", expectedSQL, sql)
		}
	})
}
