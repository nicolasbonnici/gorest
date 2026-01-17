package query

import (
	"testing"

	"github.com/nicolasbonnici/gorest/database/mysql"
	"github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/database/sqlite"
)

// Test Basic GROUP BY

func TestGroupBy_SingleColumn_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select("status").
		SelectExpr(As(Count(Col("*")), "count")).
		From("orders").
		GroupBy("status")

	sql, args := query.Build()

	expectedSQL := `SELECT "status", COUNT("*") AS "count" FROM "orders" GROUP BY "status"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestGroupBy_MultipleColumns_MySQL(t *testing.T) {
	dialect := &mysql.MySQLDialect{}
	builder := New(dialect)

	query := builder.Select("user_id", "status").
		SelectExpr(As(Sum(Col("total")), "total_amount")).
		From("orders").
		GroupBy("user_id", "status")

	sql, args := query.Build()

	expectedSQL := "SELECT `user_id`, `status`, SUM(`total`) AS `total_amount` FROM `orders` GROUP BY `user_id`, `status`"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestGroupBy_WithWhere_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select("category").
		SelectExpr(As(Count(Col("*")), "product_count")).
		From("products").
		Where(Gt("price", 100)).
		GroupBy("category")

	sql, args := query.Build()

	expectedSQL := `SELECT "category", COUNT("*") AS "product_count" FROM "products" WHERE "price" > $1 GROUP BY "category"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != 100 {
		t.Errorf("Expected args [100], got %v", args)
	}
}

func TestGroupBy_WithOrderBy_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select("status").
		SelectExpr(As(Count(Col("*")), "count")).
		From("orders").
		GroupBy("status").
		OrderBy("status", ASC)

	sql, args := query.Build()

	expectedSQL := `SELECT "status", COUNT("*") AS "count" FROM "orders" GROUP BY "status" ORDER BY "status" ASC`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

// Test HAVING Clause

func TestHaving_Simple_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	countExpr := Count(Col("*"))

	query := builder.Select("status").
		SelectExpr(As(countExpr, "count")).
		From("orders").
		GroupBy("status").
		Having(Gt("COUNT(*)", 10))

	sql, args := query.Build()

	expectedSQL := `SELECT "status", COUNT("*") AS "count" FROM "orders" GROUP BY "status" HAVING "COUNT(*)" > $1`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != 10 {
		t.Errorf("Expected args [10], got %v", args)
	}
}

func TestHaving_Multiple_MySQL(t *testing.T) {
	dialect := &mysql.MySQLDialect{}
	builder := New(dialect)

	query := builder.Select("user_id").
		SelectExpr(
			As(Count(Col("*")), "order_count"),
			As(Sum(Col("total")), "total_spent"),
		).
		From("orders").
		GroupBy("user_id").
		Having(Gt("COUNT(*)", 5)).
		Having(Gt("SUM(total)", 1000))

	sql, args := query.Build()

	expectedSQL := "SELECT `user_id`, COUNT(`*`) AS `order_count`, SUM(`total`) AS `total_spent` FROM `orders` GROUP BY `user_id` HAVING `COUNT(*)` > ? AND `SUM(total)` > ?"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 2 || args[0] != 5 || args[1] != 1000 {
		t.Errorf("Expected args [5, 1000], got %v", args)
	}
}

func TestHaving_WithWhereAndOrderBy_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select("category").
		SelectExpr(As(Avg(Col("price")), "avg_price")).
		From("products").
		Where(Eq("active", true)).
		GroupBy("category").
		Having(Gt("AVG(price)", 50)).
		OrderBy("category", ASC)

	sql, args := query.Build()

	expectedSQL := `SELECT "category", AVG("price") AS "avg_price" FROM "products" WHERE "active" = $1 GROUP BY "category" HAVING "AVG(price)" > $2 ORDER BY "category" ASC`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 2 || args[0] != true || args[1] != 50 {
		t.Errorf("Expected args [true, 50], got %v", args)
	}
}

// Test GROUP BY with Expressions

func TestGroupByExpr_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(
			As(RawExpr("DATE(created_at)"), "date"),
			As(Count(Col("*")), "daily_count"),
		).
		From("orders").
		GroupByExpr(RawExpr("DATE(created_at)"))

	sql, args := query.Build()

	expectedSQL := `SELECT DATE(created_at) AS "date", COUNT("*") AS "daily_count" FROM "orders" GROUP BY DATE(created_at)`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestGroupByExpr_WithParams_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(
			As(RawExpr("FLOOR(price / ?)", 10), "price_bracket"),
			As(Count(Col("*")), "count"),
		).
		From("products").
		GroupByExpr(RawExpr("FLOOR(price / ?)", 10))

	sql, args := query.Build()

	expectedSQL := `SELECT FLOOR(price / $1) AS "price_bracket", COUNT("*") AS "count" FROM "products" GROUP BY FLOOR(price / $2)`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 2 || args[0] != 10 || args[1] != 10 {
		t.Errorf("Expected args [10, 10], got %v", args)
	}
}

func TestMixedGroupBy_ColumnsAndExpressions_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select("category").
		SelectExpr(
			As(RawExpr("EXTRACT(YEAR FROM created_at)"), "year"),
			As(Sum(Col("total")), "yearly_total"),
		).
		From("orders").
		GroupBy("category").
		GroupByExpr(RawExpr("EXTRACT(YEAR FROM created_at)"))

	sql, args := query.Build()

	expectedSQL := `SELECT "category", EXTRACT(YEAR FROM created_at) AS "year", SUM("total") AS "yearly_total" FROM "orders" GROUP BY "category", EXTRACT(YEAR FROM created_at)`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

// Test Complex Queries

func TestComplexAggregation_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select("user_id", "status").
		SelectExpr(
			As(Count(Col("*")), "order_count"),
			As(Sum(Col("total")), "total_amount"),
			As(Avg(Col("total")), "avg_amount"),
			As(Min(Col("created_at")), "first_order"),
			As(Max(Col("created_at")), "last_order"),
		).
		From("orders").
		Where(Gt("created_at", "2024-01-01")).
		GroupBy("user_id", "status").
		Having(Gt("COUNT(*)", 3)).
		Having(Gt("SUM(total)", 500)).
		OrderBy("user_id", ASC).
		OrderByExpr(Sum(Col("total")), DESC)

	sql, args := query.Build()

	expectedSQL := `SELECT "user_id", "status", COUNT("*") AS "order_count", SUM("total") AS "total_amount", AVG("total") AS "avg_amount", MIN("created_at") AS "first_order", MAX("created_at") AS "last_order" FROM "orders" WHERE "created_at" > $1 GROUP BY "user_id", "status" HAVING "COUNT(*)" > $2 AND "SUM(total)" > $3 ORDER BY "user_id" ASC, SUM("total") DESC`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d: %v", len(args), args)
	}
	if args[0] != "2024-01-01" || args[1] != 3 || args[2] != 500 {
		t.Errorf("Expected args [2024-01-01, 3, 500], got %v", args)
	}
}

func TestGroupByWithJoin_MySQL(t *testing.T) {
	dialect := &mysql.MySQLDialect{}
	builder := New(dialect)

	query := builder.Select("u.name", "u.id").
		SelectExpr(As(Count(Col("o.id")), "order_count")).
		From("users").As("u").
		LeftJoinAs("orders", "o", ColEq("o.user_id", "u.id")).
		GroupBy("u.id", "u.name").
		Having(Gt("COUNT(o.id)", 0))

	sql, args := query.Build()

	expectedSQL := "SELECT `u.name`, `u.id`, COUNT(`o.id`) AS `order_count` FROM `users` AS `u` LEFT JOIN `orders` AS `o` ON `o.user_id` = `u.id` GROUP BY `u.id`, `u.name` HAVING `COUNT(o.id)` > ?"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != 0 {
		t.Errorf("Expected args [0], got %v", args)
	}
}

func TestGroupByWithDistinct_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select().
		SelectExpr(
			Col("status"),
			As(CountDistinct(Col("user_id")), "unique_users"),
		).
		From("orders").
		GroupBy("status")

	sql, args := query.Build()

	expectedSQL := `SELECT "status", COUNT(DISTINCT "user_id") AS "unique_users" FROM "orders" GROUP BY "status"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestGroupByWithLimit_SQLite(t *testing.T) {
	dialect := &sqlite.SQLiteDialect{}
	builder := New(dialect)

	query := builder.Select("category").
		SelectExpr(As(Count(Col("*")), "count")).
		From("products").
		GroupBy("category").
		OrderByExpr(Count(Col("*")), DESC).
		Limit(10)

	sql, args := query.Build()

	expectedSQL := `SELECT "category", COUNT("*") AS "count" FROM "products" GROUP BY "category" ORDER BY COUNT("*") DESC LIMIT 10`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

// Test Edge Cases

func TestGroupByOnly_NoAggregates_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	query := builder.Select("status", "priority").
		From("tasks").
		GroupBy("status", "priority")

	sql, args := query.Build()

	expectedSQL := `SELECT "status", "priority" FROM "tasks" GROUP BY "status", "priority"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestHavingOnly_WithoutGroupBy_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	// This is technically valid SQL (HAVING without GROUP BY)
	query := builder.Select().
		SelectExpr(As(Count(Col("*")), "total")).
		From("users").
		Having(Gt("COUNT(*)", 100))

	sql, args := query.Build()

	expectedSQL := `SELECT COUNT("*") AS "total" FROM "users" HAVING "COUNT(*)" > $1`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != 100 {
		t.Errorf("Expected args [100], got %v", args)
	}
}
