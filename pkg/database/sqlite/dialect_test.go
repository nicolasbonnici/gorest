package sqlite

import (
	"testing"

	"github.com/nicolasbonnici/gorest/pkg/database"
)

func TestSQLiteDialect_Placeholder(t *testing.T) {
	d := &SQLiteDialect{}

	for i := 1; i <= 10; i++ {
		result := d.Placeholder(i)
		if result != "?" {
			t.Errorf("Placeholder(%d) = %q, want %q", i, result, "?")
		}
	}
}

func TestSQLiteDialect_SupportsReturning(t *testing.T) {
	d := &SQLiteDialect{}
	if !d.SupportsReturning() {
		t.Error("SQLite should support RETURNING clause")
	}
}

func TestSQLiteDialect_ReturningClause(t *testing.T) {
	d := &SQLiteDialect{}

	tests := []struct {
		name     string
		cols     []string
		expected string
	}{
		{"no columns", []string{}, "RETURNING id"},
		{"single column", []string{"id"}, `RETURNING "id"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.ReturningClause(tt.cols...)
			if result != tt.expected {
				t.Errorf("ReturningClause(%v) = %q, want %q", tt.cols, result, tt.expected)
			}
		})
	}
}

func TestSQLiteDialect_QuoteIdentifier(t *testing.T) {
	d := &SQLiteDialect{}

	tests := []struct {
		name     string
		expected string
	}{
		{"id", `"id"`},
		{"user_name", `"user_name"`},
		{"table", `"table"`},
	}

	for _, tt := range tests {
		result := d.QuoteIdentifier(tt.name)
		if result != tt.expected {
			t.Errorf("QuoteIdentifier(%q) = %q, want %q", tt.name, result, tt.expected)
		}
	}
}

func TestSQLiteDialect_MapType(t *testing.T) {
	d := &SQLiteDialect{}

	tests := []struct {
		stdType  string
		expected string
	}{
		{string(database.TypeInteger), "INTEGER"},
		{string(database.TypeBigInt), "INTEGER"},
		{string(database.TypeString), "TEXT"},
		{string(database.TypeText), "TEXT"},
		{string(database.TypeBoolean), "INTEGER"},
		{string(database.TypeTimestamp), "TEXT"},
		{string(database.TypeUUID), "TEXT"},
		{string(database.TypeJSON), "TEXT"},
		{string(database.TypeFloat), "REAL"},
		{string(database.TypeDecimal), "REAL"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		result := d.MapType(tt.stdType)
		if result != tt.expected {
			t.Errorf("MapType(%q) = %q, want %q", tt.stdType, result, tt.expected)
		}
	}
}
