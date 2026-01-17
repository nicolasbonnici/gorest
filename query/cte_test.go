package query

import (
	"testing"

	"github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/database/sqlite"
)

// Test Basic CTE

func TestCTE_Basic_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	// Subquery for CTE
	activeUsers := builder.Select("id", "name", "email").
		From("users").
		Where(Eq("status", "active"))

	// Main query using CTE
	query := builder.WithCTE("active_users", activeUsers).
		Select("*").
		From("active_users")

	sql, args := query.Build()

	expectedSQL := `WITH "active_users" AS (SELECT "id", "name", "email" FROM "users" WHERE "status" = $1) SELECT "*" FROM "active_users"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != "active" {
		t.Errorf("Expected args [active], got %v", args)
	}
}

func TestCTE_WithColumns_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	// Subquery for CTE
	userStats := builder.Select("user_id").
		SelectExpr(As(Count(Col("*")), "order_count")).
		From("orders").
		GroupBy("user_id")

	// Main query using CTE with explicit columns
	query := builder.WithCTE("user_stats", userStats, "id", "count").
		Select("u.name", "us.count").
		From("users").As("u").
		JoinAs("user_stats", "us", ColEq("us.id", "u.id"))

	sql, args := query.Build()

	expectedSQL := `WITH "user_stats" ("id", "count") AS (SELECT "user_id", COUNT("*") AS "order_count" FROM "orders" GROUP BY "user_id") SELECT "u.name", "us.count" FROM "users" AS "u" INNER JOIN "user_stats" AS "us" ON "us.id" = "u.id"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

// Test Multiple CTEs

func TestCTE_Multiple_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	// First CTE - active users
	activeUsers := builder.Select("id", "name").
		From("users").
		Where(Eq("status", "active"))

	// Second CTE - recent orders
	recentOrders := builder.Select("user_id", "total").
		From("orders").
		Where(Gt("created_at", "2024-01-01"))

	// Main query using both CTEs
	query := builder.WithCTE("active_users", activeUsers).
		AndCTE("recent_orders", recentOrders).
		Select("au.name", "ro.total").
		From("active_users").As("au").
		JoinAs("recent_orders", "ro", ColEq("ro.user_id", "au.id"))

	sql, args := query.Build()

	expectedSQL := `WITH "active_users" AS (SELECT "id", "name" FROM "users" WHERE "status" = $1), "recent_orders" AS (SELECT "user_id", "total" FROM "orders" WHERE "created_at" > $2) SELECT "au.name", "ro.total" FROM "active_users" AS "au" INNER JOIN "recent_orders" AS "ro" ON "ro.user_id" = "au.id"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 2 || args[0] != "active" || args[1] != "2024-01-01" {
		t.Errorf("Expected args [active, 2024-01-01], got %v", args)
	}
}

// Test Recursive CTE

func TestCTE_Recursive_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	// Recursive CTE for organizational hierarchy
	// Note: This is a simplified example. Real recursive queries would use UNION
	orgHierarchy := builder.Select("id", "name", "manager_id", "level").
		From("employees").
		Where(IsNull("manager_id"))

	query := builder.WithRecursiveCTE("org_hierarchy", orgHierarchy).
		Select("*").
		From("org_hierarchy")

	sql, args := query.Build()

	expectedSQL := `WITH RECURSIVE "org_hierarchy" AS (SELECT "id", "name", "manager_id", "level" FROM "employees" WHERE "manager_id" IS NULL) SELECT "*" FROM "org_hierarchy"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

// Test CTE with Complex Main Query

func TestCTE_ComplexMain_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	// CTE with aggregation
	orderSummary := builder.Select("user_id").
		SelectExpr(
			As(Sum(Col("total")), "total_spent"),
			As(Count(Col("*")), "order_count"),
		).
		From("orders").
		Where(Eq("status", "completed")).
		GroupBy("user_id")

	// Main query with JOIN, WHERE, ORDER BY
	query := builder.WithCTE("order_summary", orderSummary).
		Select("u.name", "os.total_spent", "os.order_count").
		From("users").As("u").
		JoinAs("order_summary", "os", ColEq("os.user_id", "u.id")).
		Where(Gt("os.total_spent", 1000)).
		OrderBy("os.total_spent", DESC).
		Limit(10)

	sql, args := query.Build()

	expectedSQL := `WITH "order_summary" AS (SELECT "user_id", SUM("total") AS "total_spent", COUNT("*") AS "order_count" FROM "orders" WHERE "status" = $1 GROUP BY "user_id") SELECT "u.name", "os.total_spent", "os.order_count" FROM "users" AS "u" INNER JOIN "order_summary" AS "os" ON "os.user_id" = "u.id" WHERE "os.total_spent" > $2 ORDER BY "os.total_spent" DESC LIMIT 10`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 2 || args[0] != "completed" || args[1] != 1000 {
		t.Errorf("Expected args [completed, 1000], got %v", args)
	}
}

// Test CTE with Window Functions

func TestCTE_WithWindowFunction_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	win := Window().
		PartitionBy(Col("category")).
		OrderBy(Col("price"), DESC)

	// CTE with window function
	rankedProducts := builder.Select("id", "name", "category", "price").
		SelectExpr(As(RowNumber(win), "rank")).
		From("products")

	// Main query filtering ranked results
	query := builder.WithCTE("ranked_products", rankedProducts).
		Select("*").
		From("ranked_products").
		Where(Lte("rank", 3))

	sql, args := query.Build()

	expectedSQL := `WITH "ranked_products" AS (SELECT "id", "name", "category", "price", ROW_NUMBER() OVER (PARTITION BY "category" ORDER BY "price" DESC) AS "rank" FROM "products") SELECT "*" FROM "ranked_products" WHERE "rank" <= $1`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != 3 {
		t.Errorf("Expected args [3], got %v", args)
	}
}

// Test CTE with Subqueries

func TestCTE_WithSubquery_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	// Subquery for CTE WHERE clause
	premiumUsers := builder.Select("id").From("premium_memberships")

	// CTE using subquery
	activePremiumUsers := builder.Select("*").
		From("users").
		Where(InSubquery("id", premiumUsers)).
		Where(Eq("status", "active"))

	// Main query
	query := builder.WithCTE("active_premium_users", activePremiumUsers).
		Select("name", "email").
		From("active_premium_users")

	sql, args := query.Build()

	expectedSQL := `WITH "active_premium_users" AS (SELECT "*" FROM "users" WHERE "id" IN (SELECT "id" FROM "premium_memberships") AND "status" = $1) SELECT "name", "email" FROM "active_premium_users"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != "active" {
		t.Errorf("Expected args [active], got %v", args)
	}
}

// Test CTE with GROUP BY and HAVING

func TestCTE_WithGroupByHaving_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	// CTE with GROUP BY and HAVING
	topCategories := builder.Select("category").
		SelectExpr(As(Count(Col("*")), "product_count")).
		From("products").
		GroupBy("category").
		Having(Gt("COUNT(*)", 10))

	// Main query
	query := builder.WithCTE("top_categories", topCategories).
		Select("*").
		From("top_categories").
		OrderBy("product_count", DESC)

	sql, args := query.Build()

	expectedSQL := `WITH "top_categories" AS (SELECT "category", COUNT("*") AS "product_count" FROM "products" GROUP BY "category" HAVING "COUNT(*)" > $1) SELECT "*" FROM "top_categories" ORDER BY "product_count" DESC`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != 10 {
		t.Errorf("Expected args [10], got %v", args)
	}
}

// Test SQLite Support

func TestCTE_SQLite(t *testing.T) {
	dialect := &sqlite.SQLiteDialect{}
	builder := New(dialect)

	activeUsers := builder.Select("id", "name").
		From("users").
		Where(Eq("active", 1))

	query := builder.WithCTE("active_users", activeUsers).
		Select("*").
		From("active_users").
		Limit(100)

	sql, args := query.Build()

	expectedSQL := `WITH "active_users" AS (SELECT "id", "name" FROM "users" WHERE "active" = ?) SELECT "*" FROM "active_users" LIMIT 100`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != 1 {
		t.Errorf("Expected args [1], got %v", args)
	}
}

// Test Multiple CTEs with Dependencies

func TestCTE_MultipleDependencies_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	// First CTE
	regionalSales := builder.Select("region").
		SelectExpr(As(Sum(Col("amount")), "total_sales")).
		From("orders").
		GroupBy("region")

	// Second CTE referencing first (in a real scenario)
	topRegions := builder.Select("region").
		From("regional_sales").
		Where(Gt("total_sales", 100000))

	// Note: In this implementation, the second CTE can reference "regional_sales"
	// Main query
	query := builder.WithCTE("regional_sales", regionalSales).
		AndCTE("top_regions", topRegions).
		Select("*").
		From("top_regions")

	sql, args := query.Build()

	expectedSQL := `WITH "regional_sales" AS (SELECT "region", SUM("amount") AS "total_sales" FROM "orders" GROUP BY "region"), "top_regions" AS (SELECT "region" FROM "regional_sales" WHERE "total_sales" > $1) SELECT "*" FROM "top_regions"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != 100000 {
		t.Errorf("Expected args [100000], got %v", args)
	}
}
