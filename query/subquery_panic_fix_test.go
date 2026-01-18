package query

import (
	"strings"
	"testing"

	"github.com/nicolasbonnici/gorest/database/postgres"
)

// TestInSubquery_InvalidSubquery_NoPanic verifies that invalid subqueries don't cause panics
func TestInSubquery_InvalidSubquery_NoPanic(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	// Create a deliberately invalid subquery (uses invalid column identifiers)
	// This would previously panic in ToSQL()
	invalidSubquery := builder.Select("invalid*column").From("table")

	// This should handle the error gracefully, not panic
	condition := InSubquery("user_id", invalidSubquery)

	// Use it in a query - this should not panic
	mainQuery := builder.Select("name").From("users").Where(condition)

	// Build() should succeed (not panic) but SQL will contain an error comment
	sql, args, err := mainQuery.Build()

	if err != nil {
		t.Fatalf("Build() should not return error (error handling is in SQL), got: %v", err)
	}

	// The SQL should contain an error comment showing the problem
	if !strings.Contains(sql, "/* ERROR:") {
		t.Errorf("Expected SQL to contain error comment, got: %s", sql)
	}

	if !strings.Contains(sql, "InSubquery") {
		t.Errorf("Expected SQL error to mention 'InSubquery', got: %s", sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args for error condition, got: %v", args)
	}

	t.Logf("Successfully prevented panic. SQL contains error: %s", sql)
}

// TestNotInSubquery_InvalidSubquery_NoPanic verifies NOT IN handles invalid subqueries
func TestNotInSubquery_InvalidSubquery_NoPanic(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	invalidSubquery := builder.Select("invalid*column").From("table")
	condition := NotInSubquery("user_id", invalidSubquery)

	mainQuery := builder.Select("name").From("users").Where(condition)

	sql, args, err := mainQuery.Build()

	if err != nil {
		t.Fatalf("Build() should not return error (error handling is in SQL), got: %v", err)
	}

	if !strings.Contains(sql, "/* ERROR:") || !strings.Contains(sql, "1=0") {
		t.Errorf("Expected SQL to contain error comment and fallback, got: %s", sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args for error condition, got: %v", args)
	}

	t.Logf("Successfully prevented panic. SQL contains error: %s", sql)
}

// TestExists_InvalidSubquery_NoPanic verifies EXISTS handles invalid subqueries
func TestExists_InvalidSubquery_NoPanic(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	invalidSubquery := builder.Select("invalid*column").From("table")
	condition := Exists(invalidSubquery)

	mainQuery := builder.Select("name").From("users").Where(condition)

	sql, args, err := mainQuery.Build()

	if err != nil {
		t.Fatalf("Build() should not return error (error handling is in SQL), got: %v", err)
	}

	if !strings.Contains(sql, "/* ERROR:") || !strings.Contains(sql, "EXISTS") {
		t.Errorf("Expected SQL to contain ERROR and EXISTS, got: %s", sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args for error condition, got: %v", args)
	}

	t.Logf("Successfully prevented panic. SQL contains error: %s", sql)
}

// TestNotExists_InvalidSubquery_NoPanic verifies NOT EXISTS handles invalid subqueries
func TestNotExists_InvalidSubquery_NoPanic(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	invalidSubquery := builder.Select("invalid*column").From("table")
	condition := NotExists(invalidSubquery)

	mainQuery := builder.Select("name").From("users").Where(condition)

	sql, args, err := mainQuery.Build()

	if err != nil {
		t.Fatalf("Build() should not return error (error handling is in SQL), got: %v", err)
	}

	if !strings.Contains(sql, "/* ERROR:") || !strings.Contains(sql, "NOT EXISTS") {
		t.Errorf("Expected SQL to contain ERROR and NOT EXISTS, got: %s", sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected no args for error condition, got: %v", args)
	}

	t.Logf("Successfully prevented panic. SQL contains error: %s", sql)
}

// TestSubquery_ValidCase_StillWorks ensures valid subqueries still work correctly
func TestSubquery_ValidCase_StillWorks(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	builder := New(dialect)

	// Valid subquery
	subquery := builder.Select("id").From("orders").Where(Gt("total", 100))
	condition := InSubquery("user_id", subquery)

	mainQuery := builder.Select("name").From("users").Where(condition)

	sql, args, err := mainQuery.Build()

	if err != nil {
		t.Fatalf("Valid subquery should not produce error, got: %v", err)
	}

	if !strings.Contains(sql, "IN") {
		t.Errorf("Expected SQL to contain IN clause, got: %s", sql)
	}

	if len(args) != 1 || args[0] != 100 {
		t.Errorf("Expected args to be [100], got: %v", args)
	}

	t.Logf("Valid subquery works correctly. SQL: %s", sql)
}
