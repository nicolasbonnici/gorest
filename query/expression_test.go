package query

import (
	"testing"

	"github.com/nicolasbonnici/gorest/database/mysql"
	"github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/database/sqlite"
)

// Test Aggregate Functions

func TestCount_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(Count(Col("*"))).
		From("users")

	sql, args := query.Build()

	expectedSQL := `SELECT COUNT("*") FROM "users"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestCountDistinct_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(As(CountDistinct(Col("email")), "unique_emails")).
		From("users")

	sql, args := query.Build()

	expectedSQL := `SELECT COUNT(DISTINCT "email") AS "unique_emails" FROM "users"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestSum_MySQL(t *testing.T) {
	dialect := &mysql.MySQLDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(As(Sum(Col("amount")), "total")).
		From("orders")

	sql, args := query.Build()

	expectedSQL := "SELECT SUM(`amount`) AS `total` FROM `orders`"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestAvg_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(As(Avg(Col("price")), "avg_price")).
		From("products")

	sql, args := query.Build()

	expectedSQL := `SELECT AVG("price") AS "avg_price" FROM "products"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestMinMax_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(
			As(Min(Col("price")), "min_price"),
			As(Max(Col("price")), "max_price"),
		).
		From("products")

	sql, args := query.Build()

	expectedSQL := `SELECT MIN("price") AS "min_price", MAX("price") AS "max_price" FROM "products"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

// Test String Functions

func TestUpper_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(Upper(Col("name"))).
		From("users")

	sql, args := query.Build()

	expectedSQL := `SELECT UPPER("name") FROM "users"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestLower_MySQL(t *testing.T) {
	dialect := &mysql.MySQLDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(As(Lower(Col("email")), "lowercase_email")).
		From("users")

	sql, args := query.Build()

	expectedSQL := "SELECT LOWER(`email`) AS `lowercase_email` FROM `users`"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestConcat_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(As(Concat(Col("first_name"), Literal(" "), Col("last_name")), "full_name")).
		From("users")

	sql, args := query.Build()

	expectedSQL := `SELECT CONCAT("first_name", $1, "last_name") AS "full_name" FROM "users"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != " " {
		t.Errorf("Expected args [\" \"], got %v", args)
	}
}

func TestLength_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select("name").
		SelectExpr(As(Length(Col("name")), "name_length")).
		From("users")

	sql, args := query.Build()

	expectedSQL := `SELECT "name", LENGTH("name") AS "name_length" FROM "users"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestTrim_MySQL(t *testing.T) {
	dialect := &mysql.MySQLDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(Trim(Col("name"))).
		From("users")

	sql, args := query.Build()

	expectedSQL := "SELECT TRIM(`name`) FROM `users`"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestSubstring_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(As(Substring(Col("name"), Literal(1), Literal(5)), "short_name")).
		From("users")

	sql, args := query.Build()

	expectedSQL := `SELECT SUBSTRING("name", $1, $2) AS "short_name" FROM "users"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 2 || args[0] != 1 || args[1] != 5 {
		t.Errorf("Expected args [1, 5], got %v", args)
	}
}

// Test Math Functions

func TestAbs_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(As(Abs(Col("balance")), "abs_balance")).
		From("accounts")

	sql, args := query.Build()

	expectedSQL := `SELECT ABS("balance") AS "abs_balance" FROM "accounts"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestRound_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(As(Round(Col("price"), Literal(2)), "rounded_price")).
		From("products")

	sql, args := query.Build()

	expectedSQL := `SELECT ROUND("price", $1) AS "rounded_price" FROM "products"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != 2 {
		t.Errorf("Expected args [2], got %v", args)
	}
}

func TestCeilFloor_MySQL(t *testing.T) {
	dialect := &mysql.MySQLDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(
			As(Ceil(Col("price")), "ceil_price"),
			As(Floor(Col("price")), "floor_price"),
		).
		From("products")

	sql, args := query.Build()

	expectedSQL := "SELECT CEIL(`price`) AS `ceil_price`, FLOOR(`price`) AS `floor_price` FROM `products`"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

// Test Date/Time Functions

func TestNow_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select("id", "name").
		SelectExpr(As(Now(), "current_time")).
		From("users")

	sql, args := query.Build()

	expectedSQL := `SELECT "id", "name", NOW() AS "current_time" FROM "users"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestCurrentDate_MySQL(t *testing.T) {
	dialect := &mysql.MySQLDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(As(CurrentDate(), "today")).
		From("dual")

	sql, args := query.Build()

	expectedSQL := "SELECT CURRENT_DATE AS `today` FROM `dual`"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestCurrentTime_SQLite(t *testing.T) {
	dialect := &sqlite.SQLiteDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(As(CurrentTime(), "now")).
		From("users").
		Limit(1)

	sql, args := query.Build()

	expectedSQL := `SELECT CURRENT_TIME AS "now" FROM "users" LIMIT 1`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

// Test Conditional Functions

func TestCoalesce_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(As(Coalesce(Col("nickname"), Col("first_name"), Literal("Anonymous")), "display_name")).
		From("users")

	sql, args := query.Build()

	expectedSQL := `SELECT COALESCE("nickname", "first_name", $1) AS "display_name" FROM "users"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != "Anonymous" {
		t.Errorf("Expected args [\"Anonymous\"], got %v", args)
	}
}

func TestNullif_MySQL(t *testing.T) {
	dialect := &mysql.MySQLDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(As(Nullif(Col("status"), Literal("unknown")), "clean_status")).
		From("orders")

	sql, args := query.Build()

	expectedSQL := "SELECT NULLIF(`status`, ?) AS `clean_status` FROM `orders`"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != "unknown" {
		t.Errorf("Expected args [\"unknown\"], got %v", args)
	}
}

// Test Complex Expressions

func TestMultipleAggregates_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(
			As(Count(Col("*")), "total_users"),
			As(Count(Col("email")), "users_with_email"),
			As(Avg(Col("age")), "avg_age"),
			As(Sum(Col("purchases")), "total_purchases"),
		).
		From("users")

	sql, args := query.Build()

	expectedSQL := `SELECT COUNT("*") AS "total_users", COUNT("email") AS "users_with_email", AVG("age") AS "avg_age", SUM("purchases") AS "total_purchases" FROM "users"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestMixedColumnsAndExpressions_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select("id", "name").
		SelectExpr(
			As(Upper(Col("email")), "upper_email"),
			As(Length(Col("name")), "name_length"),
		).
		From("users").
		Where(Gt("age", 18))

	sql, args := query.Build()

	expectedSQL := `SELECT "id", "name", UPPER("email") AS "upper_email", LENGTH("name") AS "name_length" FROM "users" WHERE "age" > $1`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != 18 {
		t.Errorf("Expected args [18], got %v", args)
	}
}

func TestOrderByExpression_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select("name", "email").
		From("users").
		OrderByExpr(Length(Col("name")), DESC)

	sql, args := query.Build()

	expectedSQL := `SELECT "name", "email" FROM "users" ORDER BY LENGTH("name") DESC`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestMixedOrderBy_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(As(Count(Col("*")), "count")).
		From("orders").
		OrderBy("status", ASC).
		OrderByExpr(Count(Col("*")), DESC)

	sql, args := query.Build()

	expectedSQL := `SELECT COUNT("*") AS "count" FROM "orders" ORDER BY "status" ASC, COUNT("*") DESC`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestRawExpr_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select("name").
		SelectExpr(As(RawExpr("EXTRACT(YEAR FROM created_at)"), "year")).
		From("users")

	sql, args := query.Build()

	expectedSQL := `SELECT "name", EXTRACT(YEAR FROM created_at) AS "year" FROM "users"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestRawExprWithParams_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select("name").
		SelectExpr(As(RawExpr("age * ?", 2), "double_age")).
		From("users")

	sql, args := query.Build()

	expectedSQL := `SELECT "name", age * $1 AS "double_age" FROM "users"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != 2 {
		t.Errorf("Expected args [2], got %v", args)
	}
}

func TestNestedFunctions_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(As(Upper(Trim(Col("name"))), "clean_name")).
		From("users")

	sql, args := query.Build()

	expectedSQL := `SELECT UPPER(TRIM("name")) AS "clean_name" FROM "users"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}
