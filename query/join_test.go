package query

import (
	"strings"
	"testing"

	"github.com/nicolasbonnici/gorest/database/mysql"
	"github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/database/sqlite"
)

func TestSelectBuilder_InnerJoin(t *testing.T) {
	tests := []struct {
		name         string
		dialect      string
		builder      func() *SelectBuilder
		expectedSQL  string
		expectedArgs []any
	}{
		{
			name:    "PostgreSQL INNER JOIN",
			dialect: "postgres",
			builder: func() *SelectBuilder {
				return New(&postgres.PostgresDialect{}).
					Select("u.id", "u.name", "p.title").
					From("users").As("u").
					InnerJoin("posts", ColEq("u.id", "p.user_id")).
					Where(Eq("u.status", "active"))
			},
			expectedSQL:  `SELECT "u.id", "u.name", "p.title" FROM "users" AS "u" INNER JOIN "posts" ON "u.id" = "p.user_id" WHERE "u.status" = $1`,
			expectedArgs: []any{"active"},
		},
		{
			name:    "MySQL INNER JOIN",
			dialect: "mysql",
			builder: func() *SelectBuilder {
				return New(&mysql.MySQLDialect{}).
					Select("u.id", "u.name", "o.total").
					From("users").As("u").
					InnerJoin("orders", ColEq("u.id", "o.user_id")).
					Where(Gt("o.total", 100))
			},
			expectedSQL:  "SELECT `u.id`, `u.name`, `o.total` FROM `users` AS `u` INNER JOIN `orders` ON `u.id` = `o.user_id` WHERE `o.total` > ?",
			expectedArgs: []any{100},
		},
		{
			name:    "SQLite INNER JOIN with alias",
			dialect: "sqlite",
			builder: func() *SelectBuilder {
				return New(&sqlite.SQLiteDialect{}).
					Select("u.name", "p.title").
					From("users").As("u").
					JoinAs("posts", "p", ColEq("u.id", "p.user_id"))
			},
			expectedSQL:  `SELECT "u.name", "p.title" FROM "users" AS "u" INNER JOIN "posts" AS "p" ON "u.id" = "p.user_id"`,
			expectedArgs: []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args := tt.builder().Build()

			if sql != tt.expectedSQL {
				t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", tt.expectedSQL, sql)
			}

			if len(args) != len(tt.expectedArgs) {
				t.Errorf("Args count mismatch: expected %d, got %d", len(tt.expectedArgs), len(args))
			}

			for i, arg := range args {
				if arg != tt.expectedArgs[i] {
					t.Errorf("Arg[%d] mismatch: expected %v, got %v", i, tt.expectedArgs[i], arg)
				}
			}
		})
	}
}

func TestSelectBuilder_LeftJoin(t *testing.T) {
	tests := []struct {
		name         string
		dialect      string
		builder      func() *SelectBuilder
		expectedSQL  string
		expectedArgs []any
	}{
		{
			name:    "PostgreSQL LEFT JOIN",
			dialect: "postgres",
			builder: func() *SelectBuilder {
				return New(&postgres.PostgresDialect{}).
					Select("u.id", "u.name", "p.title").
					From("users").As("u").
					LeftJoin("posts", ColEq("u.id", "p.user_id"))
			},
			expectedSQL:  `SELECT "u.id", "u.name", "p.title" FROM "users" AS "u" LEFT JOIN "posts" ON "u.id" = "p.user_id"`,
			expectedArgs: []any{},
		},
		{
			name:    "MySQL LEFT JOIN with WHERE",
			dialect: "mysql",
			builder: func() *SelectBuilder {
				return New(&mysql.MySQLDialect{}).
					Select("u.id", "u.name", "c.count").
					From("users").As("u").
					LeftJoin("comment_counts", ColEq("u.id", "c.user_id")).
					Where(IsNotNull("c.count"))
			},
			expectedSQL:  "SELECT `u.id`, `u.name`, `c.count` FROM `users` AS `u` LEFT JOIN `comment_counts` ON `u.id` = `c.user_id` WHERE `c.count` IS NOT NULL",
			expectedArgs: []any{},
		},
		{
			name:    "SQLite LEFT JOIN with alias",
			dialect: "sqlite",
			builder: func() *SelectBuilder {
				return New(&sqlite.SQLiteDialect{}).
					Select("u.name", "p.title").
					From("users").As("u").
					LeftJoinAs("posts", "p", ColEq("u.id", "p.user_id"))
			},
			expectedSQL:  `SELECT "u.name", "p.title" FROM "users" AS "u" LEFT JOIN "posts" AS "p" ON "u.id" = "p.user_id"`,
			expectedArgs: []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args := tt.builder().Build()

			if sql != tt.expectedSQL {
				t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", tt.expectedSQL, sql)
			}

			if len(args) != len(tt.expectedArgs) {
				t.Errorf("Args count mismatch: expected %d, got %d", len(tt.expectedArgs), len(args))
			}
		})
	}
}

func TestSelectBuilder_RightJoin(t *testing.T) {
	t.Run("PostgreSQL RIGHT JOIN", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Select("u.id", "u.name", "o.total").
			From("users").As("u").
			RightJoin("orders", ColEq("u.id", "o.user_id"))

		sql, args := builder.Build()

		expectedSQL := `SELECT "u.id", "u.name", "o.total" FROM "users" AS "u" RIGHT JOIN "orders" ON "u.id" = "o.user_id"`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 0 {
			t.Errorf("Expected 0 args, got %d", len(args))
		}
	})

	t.Run("MySQL RIGHT JOIN", func(t *testing.T) {
		builder := New(&mysql.MySQLDialect{}).
			Select("*").
			From("users").As("u").
			RightJoin("orders", ColEq("u.id", "o.user_id"))

		sql, _ := builder.Build()

		if !strings.Contains(sql, "RIGHT JOIN") {
			t.Error("Expected RIGHT JOIN in SQL")
		}
	})
}

func TestSelectBuilder_FullJoin(t *testing.T) {
	t.Run("PostgreSQL FULL JOIN", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Select("u.id", "u.name", "o.total").
			From("users").As("u").
			FullJoin("orders", ColEq("u.id", "o.user_id"))

		sql, args := builder.Build()

		expectedSQL := `SELECT "u.id", "u.name", "o.total" FROM "users" AS "u" FULL OUTER JOIN "orders" ON "u.id" = "o.user_id"`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 0 {
			t.Errorf("Expected 0 args, got %d", len(args))
		}
	})

	t.Run("MySQL FULL JOIN panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for MySQL FULL JOIN, but didn't panic")
			}
		}()

		New(&mysql.MySQLDialect{}).
			Select("*").
			From("users").
			FullJoin("orders", ColEq("u.id", "o.user_id")).
			Build()
	})

	t.Run("SQLite FULL JOIN panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for SQLite FULL JOIN, but didn't panic")
			}
		}()

		New(&sqlite.SQLiteDialect{}).
			Select("*").
			From("users").
			FullJoin("orders", ColEq("u.id", "o.user_id")).
			Build()
	})
}

func TestSelectBuilder_CrossJoin(t *testing.T) {
	tests := []struct {
		name        string
		dialect     string
		builder     func() *SelectBuilder
		expectedSQL string
	}{
		{
			name:    "PostgreSQL CROSS JOIN",
			dialect: "postgres",
			builder: func() *SelectBuilder {
				return New(&postgres.PostgresDialect{}).
					Select("*").
					From("users").
					CrossJoin("roles")
			},
			expectedSQL: `SELECT "*" FROM "users" CROSS JOIN "roles"`,
		},
		{
			name:    "MySQL CROSS JOIN",
			dialect: "mysql",
			builder: func() *SelectBuilder {
				return New(&mysql.MySQLDialect{}).
					Select("u.name", "r.name").
					From("users").As("u").
					CrossJoin("roles")
			},
			expectedSQL: "SELECT `u.name`, `r.name` FROM `users` AS `u` CROSS JOIN `roles`",
		},
		{
			name:    "SQLite CROSS JOIN",
			dialect: "sqlite",
			builder: func() *SelectBuilder {
				return New(&sqlite.SQLiteDialect{}).
					Select("*").
					From("colors").
					CrossJoin("sizes")
			},
			expectedSQL: `SELECT "*" FROM "colors" CROSS JOIN "sizes"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args := tt.builder().Build()

			if sql != tt.expectedSQL {
				t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", tt.expectedSQL, sql)
			}

			if len(args) != 0 {
				t.Errorf("Expected 0 args, got %d", len(args))
			}
		})
	}
}

func TestSelectBuilder_MultipleJoins(t *testing.T) {
	t.Run("PostgreSQL multiple JOINs", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Select("u.id", "u.name", "p.title", "c.content").
			From("users").As("u").
			InnerJoin("posts", ColEq("u.id", "p.user_id")).
			LeftJoin("comments", ColEq("p.id", "c.post_id")).
			Where(Eq("u.status", "active"))

		sql, args := builder.Build()

		expectedSQL := `SELECT "u.id", "u.name", "p.title", "c.content" FROM "users" AS "u" INNER JOIN "posts" ON "u.id" = "p.user_id" LEFT JOIN "comments" ON "p.id" = "c.post_id" WHERE "u.status" = $1`
		if sql != expectedSQL {
			t.Errorf("SQL mismatch\nExpected: %s\nGot:      %s", expectedSQL, sql)
		}

		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got %d", len(args))
		}
	})

	t.Run("MySQL complex multi-join query", func(t *testing.T) {
		builder := New(&mysql.MySQLDialect{}).
			Select("u.name", "p.title", "c.count", "t.name").
			From("users").As("u").
			InnerJoin("posts", ColEq("u.id", "p.user_id")).
			LeftJoin("comment_counts", ColEq("p.id", "c.post_id")).
			InnerJoin("tags", ColEq("p.tag_id", "t.id")).
			Where(Gt("c.count", 10)).
			OrderBy("c.count", DESC)

		sql, args := builder.Build()

		if !strings.Contains(sql, "INNER JOIN") {
			t.Error("Expected INNER JOIN in SQL")
		}

		if !strings.Contains(sql, "LEFT JOIN") {
			t.Error("Expected LEFT JOIN in SQL")
		}

		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got %d", len(args))
		}
	})
}

func TestSelectBuilder_JoinWithComplexConditions(t *testing.T) {
	t.Run("JOIN with AND condition", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Select("*").
			From("users").As("u").
			InnerJoin("posts", And(
				ColEq("u.id", "p.user_id"),
				Eq("p.status", "published"),
			))

		sql, args := builder.Build()

		if !strings.Contains(sql, "INNER JOIN") {
			t.Error("Expected INNER JOIN in SQL")
		}

		if !strings.Contains(sql, "AND") {
			t.Error("Expected AND in JOIN condition")
		}

		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got %d", len(args))
		}
	})

	t.Run("JOIN with OR condition", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Select("*").
			From("users").As("u").
			LeftJoin("posts", Or(
				ColEq("u.id", "p.user_id"),
				ColEq("u.id", "p.author_id"),
			))

		sql, _ := builder.Build()

		if !strings.Contains(sql, "LEFT JOIN") {
			t.Error("Expected LEFT JOIN in SQL")
		}

		if !strings.Contains(sql, "OR") {
			t.Error("Expected OR in JOIN condition")
		}
	})
}

func TestSelectBuilder_JoinParameterNumbering(t *testing.T) {
	t.Run("PostgreSQL JOIN parameter numbering", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Select("*").
			From("users").As("u").
			InnerJoin("posts", And(
				ColEq("u.id", "p.user_id"),
				Eq("p.status", "published"),
			)).
			Where(Eq("u.role", "admin")).
			And(Gt("u.age", 18))

		sql, args := builder.Build()

		if !strings.Contains(sql, "$1") || !strings.Contains(sql, "$2") || !strings.Contains(sql, "$3") {
			t.Error("Expected proper parameter numbering $1, $2, $3")
		}

		if len(args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(args))
		}

		expectedArgs := []any{"published", "admin", 18}
		for i, arg := range args {
			if arg != expectedArgs[i] {
				t.Errorf("Arg[%d] mismatch: expected %v, got %v", i, expectedArgs[i], arg)
			}
		}
	})
}

func TestSelectBuilder_JoinEdgeCases(t *testing.T) {
	t.Run("JOIN with no WHERE clause", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Select("*").
			From("users").
			InnerJoin("posts", ColEq("users.id", "posts.user_id"))

		sql, args := builder.Build()

		if !strings.Contains(sql, "INNER JOIN") {
			t.Error("Expected INNER JOIN in SQL")
		}

		if strings.Contains(sql, "WHERE") {
			t.Error("Unexpected WHERE clause in SQL")
		}

		if len(args) != 0 {
			t.Errorf("Expected 0 args, got %d", len(args))
		}
	})

	t.Run("Multiple JOINs of same type", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Select("*").
			From("users").
			LeftJoin("posts", ColEq("users.id", "posts.user_id")).
			LeftJoin("comments", ColEq("users.id", "comments.user_id")).
			LeftJoin("likes", ColEq("users.id", "likes.user_id"))

		sql, _ := builder.Build()

		leftJoinCount := strings.Count(sql, "LEFT JOIN")
		if leftJoinCount != 3 {
			t.Errorf("Expected 3 LEFT JOINs, got %d", leftJoinCount)
		}
	})

	t.Run("JOIN with DISTINCT", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Select("u.id", "u.name").
			Distinct().
			From("users").As("u").
			InnerJoin("posts", ColEq("u.id", "p.user_id"))

		sql, _ := builder.Build()

		if !strings.Contains(sql, "SELECT DISTINCT") {
			t.Error("Expected SELECT DISTINCT")
		}

		if !strings.Contains(sql, "INNER JOIN") {
			t.Error("Expected INNER JOIN")
		}
	})

	t.Run("JOIN with ORDER BY and LIMIT", func(t *testing.T) {
		builder := New(&postgres.PostgresDialect{}).
			Select("*").
			From("users").As("u").
			InnerJoin("posts", ColEq("u.id", "p.user_id")).
			OrderBy("p.created_at", DESC).
			Limit(10)

		sql, _ := builder.Build()

		if !strings.Contains(sql, "ORDER BY") {
			t.Error("Expected ORDER BY")
		}

		if !strings.Contains(sql, "LIMIT") {
			t.Error("Expected LIMIT")
		}
	})
}
