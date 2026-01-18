package query

import (
	"testing"

	"github.com/nicolasbonnici/gorest/database/mysql"
	"github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/database/sqlite"
)

func TestSelectBuilder_SimpleQuery(t *testing.T) {
	tests := []struct {
		name     string
		builder  *SelectBuilder
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "PostgreSQL - SELECT all columns",
			builder:  New(&postgres.PostgresDialect{}).Select().From("users"),
			wantSQL:  `SELECT * FROM "users"`,
			wantArgs: []any{},
		},
		{
			name:     "MySQL - SELECT all columns",
			builder:  New(&mysql.MySQLDialect{}).Select().From("users"),
			wantSQL:  "SELECT * FROM `users`",
			wantArgs: []any{},
		},
		{
			name:     "SQLite - SELECT all columns",
			builder:  New(&sqlite.SQLiteDialect{}).Select().From("users"),
			wantSQL:  `SELECT * FROM "users"`,
			wantArgs: []any{},
		},
		{
			name:     "PostgreSQL - SELECT specific columns",
			builder:  New(&postgres.PostgresDialect{}).Select("id", "name", "email").From("users"),
			wantSQL:  `SELECT "id", "name", "email" FROM "users"`,
			wantArgs: []any{},
		},
		{
			name:     "MySQL - SELECT specific columns",
			builder:  New(&mysql.MySQLDialect{}).Select("id", "name", "email").From("users"),
			wantSQL:  "SELECT `id`, `name`, `email` FROM `users`",
			wantArgs: []any{},
		},
		{
			name:     "SQLite - SELECT specific columns",
			builder:  New(&sqlite.SQLiteDialect{}).Select("id", "name", "email").From("users"),
			wantSQL:  `SELECT "id", "name", "email" FROM "users"`,
			wantArgs: []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.builder.Build()
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if sql != tt.wantSQL {
				t.Errorf("Build() SQL = %v, want %v", sql, tt.wantSQL)
			}
			if len(args) != len(tt.wantArgs) {
				t.Errorf("Build() args length = %v, want %v", len(args), len(tt.wantArgs))
			}
		})
	}
}

func TestSelectBuilder_Distinct(t *testing.T) {
	tests := []struct {
		name     string
		builder  *SelectBuilder
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "PostgreSQL - DISTINCT",
			builder:  New(&postgres.PostgresDialect{}).Select("email").From("users").Distinct(),
			wantSQL:  `SELECT DISTINCT "email" FROM "users"`,
			wantArgs: []any{},
		},
		{
			name:     "MySQL - DISTINCT",
			builder:  New(&mysql.MySQLDialect{}).Select("email").From("users").Distinct(),
			wantSQL:  "SELECT DISTINCT `email` FROM `users`",
			wantArgs: []any{},
		},
		{
			name:     "SQLite - DISTINCT",
			builder:  New(&sqlite.SQLiteDialect{}).Select("email").From("users").Distinct(),
			wantSQL:  `SELECT DISTINCT "email" FROM "users"`,
			wantArgs: []any{},
		},
		{
			name:     "PostgreSQL - DISTINCT with all columns",
			builder:  New(&postgres.PostgresDialect{}).Select().From("users").Distinct(),
			wantSQL:  `SELECT DISTINCT * FROM "users"`,
			wantArgs: []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.builder.Build()
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if sql != tt.wantSQL {
				t.Errorf("Build() SQL = %v, want %v", sql, tt.wantSQL)
			}
			if len(args) != len(tt.wantArgs) {
				t.Errorf("Build() args length = %v, want %v", len(args), len(tt.wantArgs))
			}
		})
	}
}

func TestSelectBuilder_TableAlias(t *testing.T) {
	tests := []struct {
		name     string
		builder  *SelectBuilder
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "PostgreSQL - table alias",
			builder:  New(&postgres.PostgresDialect{}).Select("u.id", "u.name").From("users").As("u"),
			wantSQL:  `SELECT "u"."id", "u"."name" FROM "users" AS "u"`,
			wantArgs: []any{},
		},
		{
			name:     "MySQL - table alias",
			builder:  New(&mysql.MySQLDialect{}).Select("u.id", "u.name").From("users").As("u"),
			wantSQL:  "SELECT `u`.`id`, `u`.`name` FROM `users` AS `u`",
			wantArgs: []any{},
		},
		{
			name:     "SQLite - table alias",
			builder:  New(&sqlite.SQLiteDialect{}).Select("u.id", "u.name").From("users").As("u"),
			wantSQL:  `SELECT "u"."id", "u"."name" FROM "users" AS "u"`,
			wantArgs: []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.builder.Build()
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if sql != tt.wantSQL {
				t.Errorf("Build() SQL = %v, want %v", sql, tt.wantSQL)
			}
			if len(args) != len(tt.wantArgs) {
				t.Errorf("Build() args length = %v, want %v", len(args), len(tt.wantArgs))
			}
		})
	}
}

func TestSelectBuilder_Where(t *testing.T) {
	tests := []struct {
		name     string
		builder  *SelectBuilder
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "PostgreSQL - single WHERE condition",
			builder:  New(&postgres.PostgresDialect{}).Select().From("users").Where(Eq("id", 1)),
			wantSQL:  `SELECT * FROM "users" WHERE "id" = $1`,
			wantArgs: []any{1},
		},
		{
			name:     "MySQL - single WHERE condition",
			builder:  New(&mysql.MySQLDialect{}).Select().From("users").Where(Eq("id", 1)),
			wantSQL:  "SELECT * FROM `users` WHERE `id` = ?",
			wantArgs: []any{1},
		},
		{
			name:     "SQLite - single WHERE condition",
			builder:  New(&sqlite.SQLiteDialect{}).Select().From("users").Where(Eq("id", 1)),
			wantSQL:  `SELECT * FROM "users" WHERE "id" = ?`,
			wantArgs: []any{1},
		},
		{
			name:     "PostgreSQL - multiple WHERE conditions with And",
			builder:  New(&postgres.PostgresDialect{}).Select().From("users").Where(Eq("status", "active")).And(Gt("age", 18)),
			wantSQL:  `SELECT * FROM "users" WHERE "status" = $1 AND "age" > $2`,
			wantArgs: []any{"active", 18},
		},
		{
			name:     "MySQL - multiple WHERE conditions with And",
			builder:  New(&mysql.MySQLDialect{}).Select().From("users").Where(Eq("status", "active")).And(Gt("age", 18)),
			wantSQL:  "SELECT * FROM `users` WHERE `status` = ? AND `age` > ?",
			wantArgs: []any{"active", 18},
		},
		{
			name:     "PostgreSQL - WHERE with IN condition",
			builder:  New(&postgres.PostgresDialect{}).Select().From("users").Where(In("id", 1, 2, 3)),
			wantSQL:  `SELECT * FROM "users" WHERE "id" IN ($1, $2, $3)`,
			wantArgs: []any{1, 2, 3},
		},
		{
			name:     "MySQL - WHERE with IN condition",
			builder:  New(&mysql.MySQLDialect{}).Select().From("users").Where(In("id", 1, 2, 3)),
			wantSQL:  "SELECT * FROM `users` WHERE `id` IN (?, ?, ?)",
			wantArgs: []any{1, 2, 3},
		},
		{
			name:     "PostgreSQL - WHERE with BETWEEN",
			builder:  New(&postgres.PostgresDialect{}).Select().From("users").Where(Between("age", 18, 65)),
			wantSQL:  `SELECT * FROM "users" WHERE "age" BETWEEN $1 AND $2`,
			wantArgs: []any{18, 65},
		},
		{
			name:     "PostgreSQL - WHERE with IS NULL",
			builder:  New(&postgres.PostgresDialect{}).Select().From("users").Where(IsNull("deleted_at")),
			wantSQL:  `SELECT * FROM "users" WHERE "deleted_at" IS NULL`,
			wantArgs: []any{},
		},
		{
			name:     "PostgreSQL - WHERE with LIKE",
			builder:  New(&postgres.PostgresDialect{}).Select().From("users").Where(Like("email", "%@example.com")),
			wantSQL:  `SELECT * FROM "users" WHERE "email" LIKE $1`,
			wantArgs: []any{"%@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.builder.Build()
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if sql != tt.wantSQL {
				t.Errorf("Build() SQL = %v, want %v", sql, tt.wantSQL)
			}
			if len(args) != len(tt.wantArgs) {
				t.Errorf("Build() args length = %v, want %v", len(args), len(tt.wantArgs))
			}
			for i, arg := range args {
				if arg != tt.wantArgs[i] {
					t.Errorf("Build() args[%d] = %v, want %v", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestSelectBuilder_WhereOr(t *testing.T) {
	tests := []struct {
		name     string
		builder  *SelectBuilder
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "PostgreSQL - WHERE with OR",
			builder:  New(&postgres.PostgresDialect{}).Select().From("users").Where(Eq("status", "active")).Or(Eq("status", "pending")),
			wantSQL:  `SELECT * FROM "users" WHERE ("status" = $1 OR "status" = $2)`,
			wantArgs: []any{"active", "pending"},
		},
		{
			name:     "MySQL - WHERE with OR",
			builder:  New(&mysql.MySQLDialect{}).Select().From("users").Where(Eq("status", "active")).Or(Eq("status", "pending")),
			wantSQL:  "SELECT * FROM `users` WHERE (`status` = ? OR `status` = ?)",
			wantArgs: []any{"active", "pending"},
		},
		{
			name:     "PostgreSQL - complex WHERE with AND and OR",
			builder:  New(&postgres.PostgresDialect{}).Select().From("users").Where(Eq("role", "admin")).Or(Eq("role", "moderator")).And(Eq("status", "active")),
			wantSQL:  `SELECT * FROM "users" WHERE ("role" = $1 OR "role" = $2) AND "status" = $3`,
			wantArgs: []any{"admin", "moderator", "active"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.builder.Build()
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if sql != tt.wantSQL {
				t.Errorf("Build() SQL = %v, want %v", sql, tt.wantSQL)
			}
			if len(args) != len(tt.wantArgs) {
				t.Errorf("Build() args length = %v, want %v", len(args), len(tt.wantArgs))
			}
			for i, arg := range args {
				if arg != tt.wantArgs[i] {
					t.Errorf("Build() args[%d] = %v, want %v", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestSelectBuilder_OrderBy(t *testing.T) {
	tests := []struct {
		name     string
		builder  *SelectBuilder
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "PostgreSQL - ORDER BY ASC",
			builder:  New(&postgres.PostgresDialect{}).Select().From("users").OrderBy("name", ASC),
			wantSQL:  `SELECT * FROM "users" ORDER BY "name" ASC`,
			wantArgs: []any{},
		},
		{
			name:     "MySQL - ORDER BY DESC",
			builder:  New(&mysql.MySQLDialect{}).Select().From("users").OrderBy("created_at", DESC),
			wantSQL:  "SELECT * FROM `users` ORDER BY `created_at` DESC",
			wantArgs: []any{},
		},
		{
			name:     "PostgreSQL - multiple ORDER BY",
			builder:  New(&postgres.PostgresDialect{}).Select().From("users").OrderBy("status", ASC).OrderBy("created_at", DESC),
			wantSQL:  `SELECT * FROM "users" ORDER BY "status" ASC, "created_at" DESC`,
			wantArgs: []any{},
		},
		{
			name:     "SQLite - ORDER BY with WHERE",
			builder:  New(&sqlite.SQLiteDialect{}).Select().From("users").Where(Eq("status", "active")).OrderBy("name", ASC),
			wantSQL:  `SELECT * FROM "users" WHERE "status" = ? ORDER BY "name" ASC`,
			wantArgs: []any{"active"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.builder.Build()
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if sql != tt.wantSQL {
				t.Errorf("Build() SQL = %v, want %v", sql, tt.wantSQL)
			}
			if len(args) != len(tt.wantArgs) {
				t.Errorf("Build() args length = %v, want %v", len(args), len(tt.wantArgs))
			}
		})
	}
}

func TestSelectBuilder_LimitOffset(t *testing.T) {
	tests := []struct {
		name     string
		builder  *SelectBuilder
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "PostgreSQL - LIMIT only",
			builder:  New(&postgres.PostgresDialect{}).Select().From("users").Limit(10),
			wantSQL:  `SELECT * FROM "users" LIMIT 10`,
			wantArgs: []any{},
		},
		{
			name:     "MySQL - LIMIT only",
			builder:  New(&mysql.MySQLDialect{}).Select().From("users").Limit(10),
			wantSQL:  "SELECT * FROM `users` LIMIT 10",
			wantArgs: []any{},
		},
		{
			name:     "PostgreSQL - LIMIT and OFFSET",
			builder:  New(&postgres.PostgresDialect{}).Select().From("users").Limit(10).Offset(20),
			wantSQL:  `SELECT * FROM "users" LIMIT 10 OFFSET 20`,
			wantArgs: []any{},
		},
		{
			name:     "MySQL - LIMIT and OFFSET",
			builder:  New(&mysql.MySQLDialect{}).Select().From("users").Limit(10).Offset(20),
			wantSQL:  "SELECT * FROM `users` LIMIT 10 OFFSET 20",
			wantArgs: []any{},
		},
		{
			name:     "SQLite - OFFSET only",
			builder:  New(&sqlite.SQLiteDialect{}).Select().From("users").Offset(10),
			wantSQL:  `SELECT * FROM "users" OFFSET 10`,
			wantArgs: []any{},
		},
		{
			name:     "PostgreSQL - LIMIT with WHERE and ORDER BY",
			builder:  New(&postgres.PostgresDialect{}).Select().From("users").Where(Eq("status", "active")).OrderBy("name", ASC).Limit(5),
			wantSQL:  `SELECT * FROM "users" WHERE "status" = $1 ORDER BY "name" ASC LIMIT 5`,
			wantArgs: []any{"active"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.builder.Build()
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if sql != tt.wantSQL {
				t.Errorf("Build() SQL = %v, want %v", sql, tt.wantSQL)
			}
			if len(args) != len(tt.wantArgs) {
				t.Errorf("Build() args length = %v, want %v", len(args), len(tt.wantArgs))
			}
		})
	}
}

func TestSelectBuilder_ParameterNumbering(t *testing.T) {
	tests := []struct {
		name     string
		builder  *SelectBuilder
		wantSQL  string
		wantArgs []any
	}{
		{
			name: "PostgreSQL - parameter numbering across clauses",
			builder: New(&postgres.PostgresDialect{}).
				Select().
				From("users").
				Where(Eq("status", "active")).
				And(In("role", "admin", "moderator")).
				And(Between("age", 18, 65)),
			wantSQL:  `SELECT * FROM "users" WHERE "status" = $1 AND "role" IN ($2, $3) AND "age" BETWEEN $4 AND $5`,
			wantArgs: []any{"active", "admin", "moderator", 18, 65},
		},
		{
			name: "MySQL - multiple parameters",
			builder: New(&mysql.MySQLDialect{}).
				Select("id", "name").
				From("users").
				Where(Eq("status", "active")).
				And(Gt("age", 18)).
				And(Like("email", "%@example.com")),
			wantSQL:  "SELECT `id`, `name` FROM `users` WHERE `status` = ? AND `age` > ? AND `email` LIKE ?",
			wantArgs: []any{"active", 18, "%@example.com"},
		},
		{
			name: "SQLite - complex parameter numbering",
			builder: New(&sqlite.SQLiteDialect{}).
				Select().
				From("users").
				Where(In("id", 1, 2, 3, 4, 5)).
				And(Eq("status", "active")),
			wantSQL:  `SELECT * FROM "users" WHERE "id" IN (?, ?, ?, ?, ?) AND "status" = ?`,
			wantArgs: []any{1, 2, 3, 4, 5, "active"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.builder.Build()
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if sql != tt.wantSQL {
				t.Errorf("Build() SQL = %v, want %v", sql, tt.wantSQL)
			}
			if len(args) != len(tt.wantArgs) {
				t.Errorf("Build() args length = %v, want %v", len(args), len(tt.wantArgs))
				return
			}
			for i, arg := range args {
				if arg != tt.wantArgs[i] {
					t.Errorf("Build() args[%d] = %v, want %v", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestSelectBuilder_IntegrationScenarios(t *testing.T) {
	tests := []struct {
		name     string
		builder  *SelectBuilder
		wantSQL  string
		wantArgs []any
	}{
		{
			name: "PostgreSQL - full featured query",
			builder: New(&postgres.PostgresDialect{}).
				Select("id", "name", "email", "created_at").
				From("users").
				As("u").
				Where(Eq("status", "active")).
				And(Gt("age", 18)).
				OrderBy("created_at", DESC).
				Limit(10).
				Offset(20),
			wantSQL:  `SELECT "id", "name", "email", "created_at" FROM "users" AS "u" WHERE "status" = $1 AND "age" > $2 ORDER BY "created_at" DESC LIMIT 10 OFFSET 20`,
			wantArgs: []any{"active", 18},
		},
		{
			name: "MySQL - complex query with DISTINCT",
			builder: New(&mysql.MySQLDialect{}).
				Select("email", "country").
				Distinct().
				From("users").
				Where(In("country", "US", "UK", "CA")).
				And(IsNotNull("email")).
				OrderBy("country", ASC).
				OrderBy("email", ASC).
				Limit(100),
			wantSQL:  "SELECT DISTINCT `email`, `country` FROM `users` WHERE `country` IN (?, ?, ?) AND `email` IS NOT NULL ORDER BY `country` ASC, `email` ASC LIMIT 100",
			wantArgs: []any{"US", "UK", "CA"},
		},
		{
			name: "SQLite - query with multiple conditions",
			builder: New(&sqlite.SQLiteDialect{}).
				Select().
				From("products").
				Where(Between("price", 10.0, 100.0)).
				And(Like("name", "%laptop%")).
				And(Eq("in_stock", true)).
				OrderBy("price", ASC).
				Limit(20),
			wantSQL:  `SELECT * FROM "products" WHERE "price" BETWEEN ? AND ? AND "name" LIKE ? AND "in_stock" = ? ORDER BY "price" ASC LIMIT 20`,
			wantArgs: []any{10.0, 100.0, "%laptop%", true},
		},
		{
			name: "PostgreSQL - query without WHERE clause",
			builder: New(&postgres.PostgresDialect{}).
				Select("id", "name").
				From("categories").
				OrderBy("name", ASC),
			wantSQL:  `SELECT "id", "name" FROM "categories" ORDER BY "name" ASC`,
			wantArgs: []any{},
		},
		{
			name: "MySQL - simple count query",
			builder: New(&mysql.MySQLDialect{}).
				Select().
				From("orders").
				Where(Eq("status", "completed")),
			wantSQL:  "SELECT * FROM `orders` WHERE `status` = ?",
			wantArgs: []any{"completed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.builder.Build()
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if sql != tt.wantSQL {
				t.Errorf("Build() SQL = %v, want %v", sql, tt.wantSQL)
			}
			if len(args) != len(tt.wantArgs) {
				t.Errorf("Build() args length = %v, want %v", len(args), len(tt.wantArgs))
				return
			}
			for i, arg := range args {
				if arg != tt.wantArgs[i] {
					t.Errorf("Build() args[%d] = %v, want %v", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestSelectBuilder_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		builder  *SelectBuilder
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "PostgreSQL - no FROM clause",
			builder:  New(&postgres.PostgresDialect{}).Select().SelectExpr(RawExpr("1"), RawExpr("2"), RawExpr("3")),
			wantSQL:  `SELECT 1, 2, 3`,
			wantArgs: []any{},
		},
		{
			name:     "PostgreSQL - empty WHERE with IN()",
			builder:  New(&postgres.PostgresDialect{}).Select().From("users").Where(In("id")),
			wantSQL:  `SELECT * FROM "users" WHERE 1=0`,
			wantArgs: []any{},
		},
		{
			name:     "PostgreSQL - WHERE with NOT IN empty list",
			builder:  New(&postgres.PostgresDialect{}).Select().From("users").Where(NotIn("id")),
			wantSQL:  `SELECT * FROM "users" WHERE 1=1`,
			wantArgs: []any{},
		},
		{
			name:     "MySQL - single column ORDER BY",
			builder:  New(&mysql.MySQLDialect{}).Select().From("users").OrderBy("id", DESC),
			wantSQL:  "SELECT * FROM `users` ORDER BY `id` DESC",
			wantArgs: []any{},
		},
		{
			name:     "SQLite - DISTINCT with single column",
			builder:  New(&sqlite.SQLiteDialect{}).Select("email").Distinct().From("users"),
			wantSQL:  `SELECT DISTINCT "email" FROM "users"`,
			wantArgs: []any{},
		},
		{
			name:     "PostgreSQL - WHERE with column comparison",
			builder:  New(&postgres.PostgresDialect{}).Select().From("orders").Where(ColGt("quantity", "min_quantity")),
			wantSQL:  `SELECT * FROM "orders" WHERE "quantity" > "min_quantity"`,
			wantArgs: []any{},
		},
		{
			name:     "PostgreSQL - WHERE with NOT condition",
			builder:  New(&postgres.PostgresDialect{}).Select().From("users").Where(Not(Eq("status", "deleted"))),
			wantSQL:  `SELECT * FROM "users" WHERE NOT ("status" = $1)`,
			wantArgs: []any{"deleted"},
		},
		{
			name: "PostgreSQL - WHERE with complex logical conditions",
			builder: New(&postgres.PostgresDialect{}).
				Select().
				From("users").
				Where(And(Eq("role", "admin"), Or(Eq("status", "active"), Eq("status", "pending")))),
			wantSQL:  `SELECT * FROM "users" WHERE ("role" = $1 AND ("status" = $2 OR "status" = $3))`,
			wantArgs: []any{"admin", "active", "pending"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.builder.Build()
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if sql != tt.wantSQL {
				t.Errorf("Build() SQL = %v, want %v", sql, tt.wantSQL)
			}
			if len(args) != len(tt.wantArgs) {
				t.Errorf("Build() args length = %v, want %v", len(args), len(tt.wantArgs))
				return
			}
			for i, arg := range args {
				if arg != tt.wantArgs[i] {
					t.Errorf("Build() args[%d] = %v, want %v", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestSelectBuilder_FluentInterface(t *testing.T) {
	builder := New(&postgres.PostgresDialect{}).
		Select("id", "name").
		From("users").
		As("u").
		Distinct().
		Where(Eq("status", "active")).
		And(Gt("age", 18)).
		OrderBy("name", ASC).
		Limit(10).
		Offset(5)

	sql, args, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT DISTINCT "id", "name" FROM "users" AS "u" WHERE "status" = $1 AND "age" > $2 ORDER BY "name" ASC LIMIT 10 OFFSET 5`
	expectedArgs := []any{"active", 18}

	if sql != expectedSQL {
		t.Errorf("Build() SQL = %v, want %v", sql, expectedSQL)
	}

	if len(args) != len(expectedArgs) {
		t.Errorf("Build() args length = %v, want %v", len(args), len(expectedArgs))
		return
	}

	for i, arg := range args {
		if arg != expectedArgs[i] {
			t.Errorf("Build() args[%d] = %v, want %v", i, arg, expectedArgs[i])
		}
	}
}

func TestSelectBuilder_MultipleWhereConditions(t *testing.T) {
	builder := New(&postgres.PostgresDialect{}).
		Select().
		From("users").
		Where(Eq("role", "admin")).
		Where(Eq("status", "active")).
		Where(Gt("age", 18))

	sql, args, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT * FROM "users" WHERE "role" = $1 AND "status" = $2 AND "age" > $3`
	expectedArgs := []any{"admin", "active", 18}

	if sql != expectedSQL {
		t.Errorf("Build() SQL = %v, want %v", sql, expectedSQL)
	}

	if len(args) != len(expectedArgs) {
		t.Errorf("Build() args length = %v, want %v", len(args), len(expectedArgs))
		return
	}

	for i, arg := range args {
		if arg != expectedArgs[i] {
			t.Errorf("Build() args[%d] = %v, want %v", i, arg, expectedArgs[i])
		}
	}
}
