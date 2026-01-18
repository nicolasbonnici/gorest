package query

import (
	"testing"

	"github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/database/sqlite"
)

// Test Ranking Functions

func TestRowNumber_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	win := Window().
		PartitionBy(Col("category")).
		OrderBy(Col("price"), DESC)

	query := builder.Select("id", "name", "category", "price").
		SelectExpr(As(RowNumber(win), "row_num")).
		From("products")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "id", "name", "category", "price", ROW_NUMBER() OVER (PARTITION BY "category" ORDER BY "price" DESC) AS "row_num" FROM "products"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestRank_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	win := Window().OrderBy(Col("score"), DESC)

	query := builder.Select("name", "score").
		SelectExpr(As(Rank(win), "rank")).
		From("players")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "name", "score", RANK() OVER (ORDER BY "score" DESC) AS "rank" FROM "players"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestDenseRank_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	win := Window().
		PartitionBy(Col("department")).
		OrderBy(Col("salary"), DESC)

	query := builder.Select("name", "department", "salary").
		SelectExpr(As(DenseRank(win), "dense_rank")).
		From("employees")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "name", "department", "salary", DENSE_RANK() OVER (PARTITION BY "department" ORDER BY "salary" DESC) AS "dense_rank" FROM "employees"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestNtile_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	win := Window().OrderBy(Col("revenue"), DESC)

	query := builder.Select("company", "revenue").
		SelectExpr(As(Ntile(Literal(4), win), "quartile")).
		From("companies")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "company", "revenue", NTILE($1) OVER (ORDER BY "revenue" DESC) AS "quartile" FROM "companies"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != 4 {
		t.Errorf("Expected args [4], got %v", args)
	}
}

// Test Value Functions

func TestFirstValue_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	win := Window().
		PartitionBy(Col("user_id")).
		OrderBy(Col("created_at"), ASC)

	query := builder.Select("user_id", "product_name", "created_at").
		SelectExpr(As(FirstValue(Col("product_name"), win), "first_purchase")).
		From("purchases")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "user_id", "product_name", "created_at", FIRST_VALUE("product_name") OVER (PARTITION BY "user_id" ORDER BY "created_at" ASC) AS "first_purchase" FROM "purchases"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestLastValue_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	win := Window().
		PartitionBy(Col("session_id")).
		OrderBy(Col("timestamp"), ASC).
		RowsBetween(UnboundedPreceding, UnboundedFollowing)

	query := builder.Select("session_id", "page", "timestamp").
		SelectExpr(As(LastValue(Col("page"), win), "last_page")).
		From("page_views")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "session_id", "page", "timestamp", LAST_VALUE("page") OVER (PARTITION BY "session_id" ORDER BY "timestamp" ASC ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) AS "last_page" FROM "page_views"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

// Test Offset Functions

func TestLag_Simple_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	win := Window().OrderBy(Col("date"), ASC)

	query := builder.Select("date", "price").
		SelectExpr(As(Lag(Col("price"), nil, nil, win), "previous_price")).
		From("stock_prices")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "date", "price", LAG("price") OVER (ORDER BY "date" ASC) AS "previous_price" FROM "stock_prices"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestLag_WithOffset_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	win := Window().
		PartitionBy(Col("product_id")).
		OrderBy(Col("date"), ASC)

	query := builder.Select("product_id", "date", "sales").
		SelectExpr(As(Lag(Col("sales"), Literal(7), nil, win), "sales_week_ago")).
		From("daily_sales")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "product_id", "date", "sales", LAG("sales", $1) OVER (PARTITION BY "product_id" ORDER BY "date" ASC) AS "sales_week_ago" FROM "daily_sales"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != 7 {
		t.Errorf("Expected args [7], got %v", args)
	}
}

func TestLag_WithDefault_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	win := Window().OrderBy(Col("month"), ASC)

	query := builder.Select("month", "revenue").
		SelectExpr(As(Lag(Col("revenue"), Literal(1), Literal(0), win), "prev_revenue")).
		From("monthly_revenue")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "month", "revenue", LAG("revenue", $1, $2) OVER (ORDER BY "month" ASC) AS "prev_revenue" FROM "monthly_revenue"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 2 || args[0] != 1 || args[1] != 0 {
		t.Errorf("Expected args [1, 0], got %v", args)
	}
}

func TestLead_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	win := Window().OrderBy(Col("date"), ASC)

	query := builder.Select("date", "temperature").
		SelectExpr(As(Lead(Col("temperature"), Literal(1), nil, win), "next_day_temp")).
		From("weather")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "date", "temperature", LEAD("temperature", $1) OVER (ORDER BY "date" ASC) AS "next_day_temp" FROM "weather"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != 1 {
		t.Errorf("Expected args [1], got %v", args)
	}
}

// Test Aggregate Window Functions

func TestSumOver_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	win := Window().
		PartitionBy(Col("user_id")).
		OrderBy(Col("order_date"), ASC).
		RowsBetween(UnboundedPreceding, CurrentRow)

	query := builder.Select("user_id", "order_date", "amount").
		SelectExpr(As(SumOver(Col("amount"), win), "running_total")).
		From("orders")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "user_id", "order_date", "amount", SUM("amount") OVER (PARTITION BY "user_id" ORDER BY "order_date" ASC ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS "running_total" FROM "orders"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestAvgOver_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	win := Window().
		PartitionBy(Col("category")).
		OrderBy(Col("id"), ASC).
		RowsFrame("2 PRECEDING", "2 FOLLOWING")

	query := builder.Select("id", "category", "price").
		SelectExpr(As(AvgOver(Col("price"), win), "moving_avg")).
		From("products")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "id", "category", "price", AVG("price") OVER (PARTITION BY "category" ORDER BY "id" ASC ROWS BETWEEN 2 PRECEDING AND 2 FOLLOWING) AS "moving_avg" FROM "products"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestCountOver_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	win := Window().
		PartitionBy(Col("department")).
		OrderBy(Col("hire_date"), ASC)

	query := builder.Select("name", "department", "hire_date").
		SelectExpr(As(CountOver(Col("*"), win), "employee_number")).
		From("employees")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "name", "department", "hire_date", COUNT("*") OVER (PARTITION BY "department" ORDER BY "hire_date" ASC) AS "employee_number" FROM "employees"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

// Test Complex Window Specifications

func TestMultipleWindowFunctions_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	win1 := Window().
		PartitionBy(Col("department")).
		OrderBy(Col("salary"), DESC)

	win2 := Window().OrderBy(Col("salary"), DESC)

	query := builder.Select("name", "department", "salary").
		SelectExpr(
			As(RowNumber(win1), "dept_rank"),
			As(Rank(win2), "overall_rank"),
		).
		From("employees")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "name", "department", "salary", ROW_NUMBER() OVER (PARTITION BY "department" ORDER BY "salary" DESC) AS "dept_rank", RANK() OVER (ORDER BY "salary" DESC) AS "overall_rank" FROM "employees"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestWindowWithMultiplePartitions_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	win := Window().
		PartitionBy(Col("region"), Col("category")).
		OrderBy(Col("sales"), DESC)

	query := builder.Select("product", "region", "category", "sales").
		SelectExpr(As(RowNumber(win), "rank_in_region_category")).
		From("product_sales")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "product", "region", "category", "sales", ROW_NUMBER() OVER (PARTITION BY "region", "category" ORDER BY "sales" DESC) AS "rank_in_region_category" FROM "product_sales"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestWindowWithMultipleOrderBy_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	win := Window().
		OrderBy(Col("score"), DESC).
		OrderBy(Col("name"), ASC)

	query := builder.Select("name", "score").
		SelectExpr(As(Rank(win), "rank")).
		From("contestants")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "name", "score", RANK() OVER (ORDER BY "score" DESC, "name" ASC) AS "rank" FROM "contestants"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

// Test Frame Specifications

func TestRangeFrame_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	win := Window().
		OrderBy(Col("date"), ASC).
		RangeFrame("INTERVAL '7 days' PRECEDING", CurrentRow)

	query := builder.Select("date", "sales").
		SelectExpr(As(SumOver(Col("sales"), win), "week_total")).
		From("daily_sales")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "date", "sales", SUM("sales") OVER (ORDER BY "date" ASC RANGE BETWEEN INTERVAL '7 days' PRECEDING AND CURRENT ROW) AS "week_total" FROM "daily_sales"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestEmptyWindow_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	win := Window()

	query := builder.Select("name", "salary").
		SelectExpr(As(RowNumber(win), "row_num")).
		From("employees")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "name", "salary", ROW_NUMBER() OVER () AS "row_num" FROM "employees"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

// Test SQLite Support

func TestRowNumber_SQLite(t *testing.T) {
	dialect := &sqlite.SQLiteDialect{}
	builder := New(dialect)

	win := Window().
		PartitionBy(Col("category")).
		OrderBy(Col("price"), DESC)

	query := builder.Select("id", "name", "category", "price").
		SelectExpr(As(RowNumber(win), "row_num")).
		From("products")

	sql, args, err := query.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	expectedSQL := `SELECT "id", "name", "category", "price", ROW_NUMBER() OVER (PARTITION BY "category" ORDER BY "price" DESC) AS "row_num" FROM "products"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}
