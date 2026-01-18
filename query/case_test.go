package query

import (
	"testing"

	"github.com/nicolasbonnici/gorest/database/mysql"
	"github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/database/sqlite"
)

// Test Simple CASE Expressions

func TestSimpleCase_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	caseExpr := Case(Col("status")).
		When(Literal("pending"), Literal("Waiting")).
		When(Literal("active"), Literal("In Progress")).
		Else(Literal("Unknown")).
		End()

	query := builder.Select("id", "name").
		SelectExpr(As(caseExpr, "status_text")).
		From("orders")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "id", "name", CASE "status" WHEN $1 THEN $2 WHEN $3 THEN $4 ELSE $5 END AS "status_text" FROM "orders"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 5 {
		t.Errorf("Expected 5 args, got %d: %v", len(args), args)
	}
	if args[0] != "pending" || args[1] != "Waiting" || args[2] != "active" || args[3] != "In Progress" || args[4] != "Unknown" {
		t.Errorf("Expected args [pending, Waiting, active, In Progress, Unknown], got %v", args)
	}
}

func TestSimpleCase_MySQL(t *testing.T) {
	dialect := &mysql.MySQLDialect{}
	builder := New(dialect)

	caseExpr := Case(Col("priority")).
		When(Literal(1), Literal("Low")).
		When(Literal(2), Literal("Medium")).
		When(Literal(3), Literal("High")).
		End()

	query := builder.Select().
		SelectExpr(As(caseExpr, "priority_name")).
		From("tasks")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := "SELECT CASE `priority` WHEN ? THEN ? WHEN ? THEN ? WHEN ? THEN ? END AS `priority_name` FROM `tasks`"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 6 {
		t.Errorf("Expected 6 args, got %d: %v", len(args), args)
	}
}

func TestSimpleCase_NoElse_SQLite(t *testing.T) {
	dialect := &sqlite.SQLiteDialect{}
	builder := New(dialect)

	caseExpr := Case(Col("type")).
		When(Literal("A"), Literal(1)).
		When(Literal("B"), Literal(2)).
		End()

	query := builder.Select("id").
		SelectExpr(As(caseExpr, "type_value")).
		From("items")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "id", CASE "type" WHEN ? THEN ? WHEN ? THEN ? END AS "type_value" FROM "items"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 4 {
		t.Errorf("Expected 4 args, got %d: %v", len(args), args)
	}
}

// Test Searched CASE Expressions

func TestSearchedCase_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	caseExpr := Case(nil).
		WhenCond(Gt("price", 100), Literal("Expensive")).
		WhenCond(Between("price", 50, 100), Literal("Moderate")).
		WhenCond(Lt("price", 50), Literal("Cheap")).
		Else(Literal("Unknown")).
		End()

	query := builder.Select("id", "name", "price").
		SelectExpr(As(caseExpr, "price_category")).
		From("products")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "id", "name", "price", CASE WHEN "price" > $1 THEN $2 WHEN "price" BETWEEN $3 AND $4 THEN $5 WHEN "price" < $6 THEN $7 ELSE $8 END AS "price_category" FROM "products"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 8 {
		t.Errorf("Expected 8 args, got %d: %v", len(args), args)
	}
	if args[0] != 100 || args[1] != "Expensive" || args[2] != 50 || args[3] != 100 || args[4] != "Moderate" {
		t.Errorf("Unexpected args: %v", args)
	}
}

func TestSearchedCase_MySQL(t *testing.T) {
	dialect := &mysql.MySQLDialect{}
	builder := New(dialect)

	caseExpr := Case(nil).
		WhenCond(Eq("status", "completed"), Literal(1)).
		WhenCond(Eq("status", "pending"), Literal(0)).
		Else(Literal(-1)).
		End()

	query := builder.Select("id").
		SelectExpr(As(caseExpr, "status_code")).
		From("orders")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := "SELECT `id`, CASE WHEN `status` = ? THEN ? WHEN `status` = ? THEN ? ELSE ? END AS `status_code` FROM `orders`"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 5 {
		t.Errorf("Expected 5 args, got %d: %v", len(args), args)
	}
}

func TestSearchedCase_ComplexConditions_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	caseExpr := Case(nil).
		WhenCond(And(Eq("status", "active"), Gt("age", 18)), Literal("Adult Active")).
		WhenCond(And(Eq("status", "active"), Lte("age", 18)), Literal("Minor Active")).
		WhenCond(Eq("status", "inactive"), Literal("Inactive")).
		End()

	query := builder.Select("name").
		SelectExpr(As(caseExpr, "user_category")).
		From("users")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "name", CASE WHEN ("status" = $1 AND "age" > $2) THEN $3 WHEN ("status" = $4 AND "age" <= $5) THEN $6 WHEN "status" = $7 THEN $8 END AS "user_category" FROM "users"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 8 {
		t.Errorf("Expected 8 args, got %d: %v", len(args), args)
	}
}

// Test CASE in Different Contexts

func TestCase_InWhere_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	// Using CASE in WHERE is less common but valid
	query := builder.Select("*").
		From("users").
		Where(Eq("status", "active"))

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "*" FROM "users" WHERE "status" = $1`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 {
		t.Errorf("Expected 1 arg, got %d: %v", len(args), args)
	}
}

func TestCase_InOrderBy_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	caseExpr := Case(Col("priority")).
		When(Literal("high"), Literal(1)).
		When(Literal("medium"), Literal(2)).
		When(Literal("low"), Literal(3)).
		End()

	query := builder.Select("id", "name", "priority").
		From("tasks").
		OrderByExpr(caseExpr, ASC)

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "id", "name", "priority" FROM "tasks" ORDER BY CASE "priority" WHEN $1 THEN $2 WHEN $3 THEN $4 WHEN $5 THEN $6 END ASC`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 6 {
		t.Errorf("Expected 6 args, got %d: %v", len(args), args)
	}
}

func TestCase_InGroupBy_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	caseExpr := Case(nil).
		WhenCond(Lt("age", 18), Literal("Minor")).
		Else(Literal("Adult")).
		End()

	query := builder.Select().
		SelectExpr(
			As(caseExpr, "age_group"),
			As(Count(Col("*")), "count"),
		).
		From("users").
		GroupByExpr(caseExpr)

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT CASE WHEN "age" < $1 THEN $2 ELSE $3 END AS "age_group", COUNT("*") AS "count" FROM "users" GROUP BY CASE WHEN "age" < $4 THEN $5 ELSE $6 END`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	// Parameters are duplicated (once in SELECT, once in GROUP BY)
	if len(args) != 6 {
		t.Errorf("Expected 6 args, got %d: %v", len(args), args)
	}
}

// Test Nested CASE

func TestNestedCase_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	innerCase := Case(Col("type")).
		When(Literal("A"), Literal("Type A")).
		When(Literal("B"), Literal("Type B")).
		Else(Literal("Other")).
		End()

	outerCase := Case(nil).
		WhenCond(Eq("status", "active"), innerCase).
		Else(Literal("Inactive")).
		End()

	query := builder.Select("id").
		SelectExpr(As(outerCase, "description")).
		From("items")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "id", CASE WHEN "status" = $1 THEN CASE "type" WHEN $2 THEN $3 WHEN $4 THEN $5 ELSE $6 END ELSE $7 END AS "description" FROM "items"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 7 {
		t.Errorf("Expected 7 args, got %d: %v", len(args), args)
	}
}

// Test CASE with Column References

func TestCase_WithColumnReferences_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	caseExpr := Case(nil).
		WhenCond(Gt("quantity", 100), Col("bulk_price")).
		WhenCond(Gt("quantity", 10), Col("wholesale_price")).
		Else(Col("retail_price")).
		End()

	query := builder.Select("id", "name").
		SelectExpr(As(caseExpr, "effective_price")).
		From("products")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "id", "name", CASE WHEN "quantity" > $1 THEN "bulk_price" WHEN "quantity" > $2 THEN "wholesale_price" ELSE "retail_price" END AS "effective_price" FROM "products"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d: %v", len(args), args)
	}
	if args[0] != 100 || args[1] != 10 {
		t.Errorf("Expected args [100, 10], got %v", args)
	}
}

// Test CASE with Functions

func TestCase_WithFunctions_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	caseExpr := Case(nil).
		WhenCond(IsNull("nickname"), Col("first_name")).
		Else(Concat(Col("nickname"), Literal(" ("), Col("first_name"), Literal(")"))).
		End()

	query := builder.Select("id").
		SelectExpr(As(caseExpr, "display_name")).
		From("users")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "id", CASE WHEN "nickname" IS NULL THEN "first_name" ELSE CONCAT("nickname", $1, "first_name", $2) END AS "display_name" FROM "users"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 2 || args[0] != " (" || args[1] != ")" {
		t.Errorf("Expected args [\" (\", \")\"], got %v", args)
	}
}

// Test Multiple CASE Expressions

func TestMultipleCaseExpressions_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	statusCase := Case(Col("status")).
		When(Literal("pending"), Literal("P")).
		When(Literal("active"), Literal("A")).
		Else(Literal("X")).
		End()

	priorityCase := Case(Col("priority")).
		When(Literal(1), Literal("Low")).
		When(Literal(2), Literal("Med")).
		When(Literal(3), Literal("High")).
		End()

	query := builder.Select("id").
		SelectExpr(
			As(statusCase, "status_code"),
			As(priorityCase, "priority_name"),
		).
		From("tasks")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "id", CASE "status" WHEN $1 THEN $2 WHEN $3 THEN $4 ELSE $5 END AS "status_code", CASE "priority" WHEN $6 THEN $7 WHEN $8 THEN $9 WHEN $10 THEN $11 END AS "priority_name" FROM "tasks"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 11 {
		t.Errorf("Expected 11 args, got %d: %v", len(args), args)
	}
}
