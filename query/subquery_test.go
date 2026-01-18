package query

import (
	"testing"

	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/database/mysql"
	"github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/database/sqlite"
)

func TestInSubquery_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	subquery := builder.Select("user_id").From("orders").Where(Gt("total", 100))
	query := builder.Select("name", "email").
		From("users").
		Where(InSubquery("id", subquery))

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "name", "email" FROM "users" WHERE "id" IN (SELECT "user_id" FROM "orders" WHERE "total" > $1)`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != 100 {
		t.Errorf("Expected args [100], got %v", args)
	}
}

func TestInSubquery_MySQL(t *testing.T) {
	dialect := &mysql.MySQLDialect{}
	builder := New(dialect)

	subquery := builder.Select("user_id").From("orders").Where(Gt("total", 100))
	query := builder.Select("name", "email").
		From("users").
		Where(InSubquery("id", subquery))

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := "SELECT `name`, `email` FROM `users` WHERE `id` IN (SELECT `user_id` FROM `orders` WHERE `total` > ?)"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != 100 {
		t.Errorf("Expected args [100], got %v", args)
	}
}

func TestInSubquery_SQLite(t *testing.T) {
	dialect := &sqlite.SQLiteDialect{}
	builder := New(dialect)

	subquery := builder.Select("user_id").From("orders").Where(Gt("total", 100))
	query := builder.Select("name", "email").
		From("users").
		Where(InSubquery("id", subquery))

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "name", "email" FROM "users" WHERE "id" IN (SELECT "user_id" FROM "orders" WHERE "total" > ?)`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != 100 {
		t.Errorf("Expected args [100], got %v", args)
	}
}

func TestNotInSubquery_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	subquery := builder.Select("id").From("blocked_users")
	query := builder.Select("*").
		From("users").
		Where(NotInSubquery("id", subquery))

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "*" FROM "users" WHERE "id" NOT IN (SELECT "id" FROM "blocked_users")`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestNotInSubquery_MySQL(t *testing.T) {
	dialect := &mysql.MySQLDialect{}
	builder := New(dialect)

	subquery := builder.Select("id").From("blocked_users")
	query := builder.Select("*").
		From("users").
		Where(NotInSubquery("id", subquery))

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := "SELECT `*` FROM `users` WHERE `id` NOT IN (SELECT `id` FROM `blocked_users`)"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestExists_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	subquery := builder.Select().SelectExpr(RawExpr("1")).From("orders").Where(ColEq("orders.user_id", "users.id"))
	query := builder.Select("name").
		From("users").
		Where(Exists(subquery))

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "name" FROM "users" WHERE EXISTS (SELECT 1 FROM "orders" WHERE "orders"."user_id" = "users"."id")`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestExists_MySQL(t *testing.T) {
	dialect := &mysql.MySQLDialect{}
	builder := New(dialect)

	subquery := builder.Select().SelectExpr(RawExpr("1")).From("orders").Where(ColEq("orders.user_id", "users.id"))
	query := builder.Select("name").
		From("users").
		Where(Exists(subquery))

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := "SELECT `name` FROM `users` WHERE EXISTS (SELECT 1 FROM `orders` WHERE `orders`.`user_id` = `users`.`id`)"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestNotExists_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	subquery := builder.Select().SelectExpr(RawExpr("1")).From("orders").Where(ColEq("orders.user_id", "users.id"))
	query := builder.Select("name").
		From("users").
		Where(NotExists(subquery))

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "name" FROM "users" WHERE NOT EXISTS (SELECT 1 FROM "orders" WHERE "orders"."user_id" = "users"."id")`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestNotExists_MySQL(t *testing.T) {
	dialect := &mysql.MySQLDialect{}
	builder := New(dialect)

	subquery := builder.Select().SelectExpr(RawExpr("1")).From("orders").Where(ColEq("orders.user_id", "users.id"))
	query := builder.Select("name").
		From("users").
		Where(NotExists(subquery))

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := "SELECT `name` FROM `users` WHERE NOT EXISTS (SELECT 1 FROM `orders` WHERE `orders`.`user_id` = `users`.`id`)"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestSubqueryWithMultipleConditions_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	subquery := builder.Select("user_id").
		From("orders").
		Where(Gt("total", 100)).
		Where(Eq("status", "completed"))

	query := builder.Select("name", "email").
		From("users").
		Where(InSubquery("id", subquery)).
		Where(Eq("active", true))

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "name", "email" FROM "users" WHERE "id" IN (SELECT "user_id" FROM "orders" WHERE "total" > $1 AND "status" = $2) AND "active" = $3`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d: %v", len(args), args)
	}
	if args[0] != 100 || args[1] != "completed" || args[2] != true {
		t.Errorf("Expected args [100, completed, true], got %v", args)
	}
}

func TestSubqueryParameterRenumbering_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	// Subquery with parameters
	subquery := builder.Select("user_id").
		From("orders").
		Where(Gt("total", 500)).
		Where(Eq("status", "pending"))

	// Main query with its own parameters
	query := builder.Select("*").
		From("users").
		Where(Eq("country", "US")).
		Where(InSubquery("id", subquery)).
		Where(Gt("age", 18))

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "*" FROM "users" WHERE "country" = $1 AND "id" IN (SELECT "user_id" FROM "orders" WHERE "total" > $2 AND "status" = $3) AND "age" > $4`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 4 {
		t.Errorf("Expected 4 args, got %d: %v", len(args), args)
	}
	if args[0] != "US" || args[1] != 500 || args[2] != "pending" || args[3] != 18 {
		t.Errorf("Expected args [US, 500, pending, 18], got %v", args)
	}
}

func TestSubqueryParameterRenumbering_MySQL(t *testing.T) {
	dialect := &mysql.MySQLDialect{}
	builder := New(dialect)

	subquery := builder.Select("user_id").
		From("orders").
		Where(Gt("total", 500)).
		Where(Eq("status", "pending"))

	query := builder.Select("*").
		From("users").
		Where(Eq("country", "US")).
		Where(InSubquery("id", subquery)).
		Where(Gt("age", 18))

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// MySQL uses ? for all parameters, so renumbering doesn't change appearance
	expectedSQL := "SELECT `*` FROM `users` WHERE `country` = ? AND `id` IN (SELECT `user_id` FROM `orders` WHERE `total` > ? AND `status` = ?) AND `age` > ?"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 4 {
		t.Errorf("Expected 4 args, got %d: %v", len(args), args)
	}
	if args[0] != "US" || args[1] != 500 || args[2] != "pending" || args[3] != 18 {
		t.Errorf("Expected args [US, 500, pending, 18], got %v", args)
	}
}

func TestComplexSubqueryWithJoin_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	// Subquery with JOIN
	subquery := builder.Select("o.user_id").
		From("orders").As("o").
		InnerJoin("order_items", ColEq("order_items.order_id", "o.id")).
		Where(Gt("order_items.quantity", 5))

	query := builder.Select("name", "email").
		From("users").
		Where(InSubquery("id", subquery))

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "name", "email" FROM "users" WHERE "id" IN (SELECT "o"."user_id" FROM "orders" AS "o" INNER JOIN "order_items" ON "order_items"."order_id" = "o"."id" WHERE "order_items"."quantity" > $1)`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != 5 {
		t.Errorf("Expected args [5], got %v", args)
	}
}

func TestMultipleSubqueries_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	subquery1 := builder.Select("user_id").From("premium_users")
	subquery2 := builder.Select("user_id").From("banned_users")

	query := builder.Select("*").
		From("users").
		Where(InSubquery("id", subquery1)).
		Where(NotInSubquery("id", subquery2))

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "*" FROM "users" WHERE "id" IN (SELECT "user_id" FROM "premium_users") AND "id" NOT IN (SELECT "user_id" FROM "banned_users")`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestSubqueryWithDistinct_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	subquery := builder.Select("user_id").From("orders").Distinct()
	query := builder.Select("*").
		From("users").
		Where(InSubquery("id", subquery))

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "*" FROM "users" WHERE "id" IN (SELECT DISTINCT "user_id" FROM "orders")`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestSubqueryWithOrderByLimit_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	subquery := builder.Select("user_id").
		From("orders").
		Where(Eq("status", "active")).
		OrderBy("created_at", DESC).
		Limit(10)

	query := builder.Select("*").
		From("users").
		Where(InSubquery("id", subquery))

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "*" FROM "users" WHERE "id" IN (SELECT "user_id" FROM "orders" WHERE "status" = $1 ORDER BY "created_at" DESC LIMIT 10)`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != "active" {
		t.Errorf("Expected args [active], got %v", args)
	}
}

func TestExistsWithMultipleConditions_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	subquery := builder.Select().SelectExpr(RawExpr("1")).
		From("orders").
		Where(ColEq("orders.user_id", "users.id")).
		Where(Gt("orders.total", 1000)).
		Where(Eq("orders.status", "completed"))

	query := builder.Select("name", "email").
		From("users").
		Where(Exists(subquery)).
		Where(Eq("users.active", true))

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "name", "email" FROM "users" WHERE EXISTS (SELECT 1 FROM "orders" WHERE "orders"."user_id" = "users"."id" AND "orders"."total" > $1 AND "orders"."status" = $2) AND "users"."active" = $3`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d: %v", len(args), args)
	}
	if args[0] != 1000 || args[1] != "completed" || args[2] != true {
		t.Errorf("Expected args [1000, completed, true], got %v", args)
	}
}

func TestCombinedExistsAndIn_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	existsSubquery := builder.Select().SelectExpr(RawExpr("1")).
		From("orders").
		Where(ColEq("orders.user_id", "users.id"))

	inSubquery := builder.Select("id").
		From("premium_plans").
		Where(Gt("price", 100))

	query := builder.Select("*").
		From("users").
		Where(Exists(existsSubquery)).
		Where(InSubquery("plan_id", inSubquery))

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "*" FROM "users" WHERE EXISTS (SELECT 1 FROM "orders" WHERE "orders"."user_id" = "users"."id") AND "plan_id" IN (SELECT "id" FROM "premium_plans" WHERE "price" > $1)`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != 100 {
		t.Errorf("Expected args [100], got %v", args)
	}
}

func TestRenumberParameters(t *testing.T) {
	tests := []struct {
		name      string
		dialect   string
		sql       string
		argCount  int
		startFrom int
		expected  string
	}{
		{
			name:      "PostgreSQL renumber from 1 to 2",
			dialect:   "postgres",
			sql:       "SELECT * FROM users WHERE age > $1 AND status = $2",
			argCount:  2,
			startFrom: 2,
			expected:  "SELECT * FROM users WHERE age > $2 AND status = $3",
		},
		{
			name:      "PostgreSQL renumber from 1 to 5",
			dialect:   "postgres",
			sql:       "SELECT * FROM orders WHERE total > $1",
			argCount:  1,
			startFrom: 5,
			expected:  "SELECT * FROM orders WHERE total > $5",
		},
		{
			name:      "MySQL no change (? stays ?)",
			dialect:   "mysql",
			sql:       "SELECT * FROM users WHERE age > ? AND status = ?",
			argCount:  2,
			startFrom: 2,
			expected:  "SELECT * FROM users WHERE age > ? AND status = ?",
		},
		{
			name:      "SQLite no change (? stays ?)",
			dialect:   "sqlite",
			sql:       "SELECT * FROM users WHERE age > ?",
			argCount:  1,
			startFrom: 5,
			expected:  "SELECT * FROM users WHERE age > ?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dialect database.Dialect
			switch tt.dialect {
			case "postgres":
				dialect = &postgres.PostgresDialect{}
			case "mysql":
				dialect = &mysql.MySQLDialect{}
			case "sqlite":
				dialect = &sqlite.SQLiteDialect{}
			}

			result := renumberParameters(dialect, tt.sql, tt.argCount, tt.startFrom)
			if result != tt.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tt.expected, result)
			}
		})
	}
}
