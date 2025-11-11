package postgres

import (
	"testing"

	"github.com/nicolasbonnici/gorest/database"
)

func TestPostgresDialect_Placeholder(t *testing.T) {
	d := &PostgresDialect{}

	tests := []struct {
		n        int
		expected string
	}{
		{1, "$1"},
		{2, "$2"},
		{10, "$10"},
		{100, "$100"},
	}

	for _, tt := range tests {
		result := d.Placeholder(tt.n)
		if result != tt.expected {
			t.Errorf("Placeholder(%d) = %q, want %q", tt.n, result, tt.expected)
		}
	}
}

func TestPostgresDialect_SupportsReturning(t *testing.T) {
	d := &PostgresDialect{}
	if !d.SupportsReturning() {
		t.Error("PostgreSQL should support RETURNING clause")
	}
}

func TestPostgresDialect_ReturningClause(t *testing.T) {
	d := &PostgresDialect{}

	tests := []struct {
		name     string
		cols     []string
		expected string
	}{
		{"no columns", []string{}, "RETURNING id"},
		{"single column", []string{"id"}, "RETURNING id"},
		{"multiple columns", []string{"id", "name"}, "RETURNING id, name"},
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

func TestPostgresDialect_QuoteIdentifier(t *testing.T) {
	d := &PostgresDialect{}

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

func TestPostgresDialect_MapType(t *testing.T) {
	d := &PostgresDialect{}

	tests := []struct {
		stdType  string
		expected string
	}{
		{string(database.TypeInteger), "INTEGER"},
		{string(database.TypeBigInt), "BIGINT"},
		{string(database.TypeString), "VARCHAR(255)"},
		{string(database.TypeText), "TEXT"},
		{string(database.TypeBoolean), "BOOLEAN"},
		{string(database.TypeTimestamp), "TIMESTAMP WITH TIME ZONE"},
		{string(database.TypeUUID), "UUID"},
		{string(database.TypeJSON), "JSONB"},
		{string(database.TypeFloat), "DOUBLE PRECISION"},
		{string(database.TypeDecimal), "NUMERIC"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		result := d.MapType(tt.stdType)
		if result != tt.expected {
			t.Errorf("MapType(%q) = %q, want %q", tt.stdType, result, tt.expected)
		}
	}
}

func TestPostgresDialect_LimitOffset(t *testing.T) {
	d := &PostgresDialect{}

	tests := []struct {
		name     string
		limit    int
		offset   int
		expected string
	}{
		{"both", 10, 5, "LIMIT 10 OFFSET 5"},
		{"only limit", 10, 0, "LIMIT 10"},
		{"only offset", 0, 5, "OFFSET 5"},
		{"neither", 0, 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.LimitOffset(tt.limit, tt.offset)
			if result != tt.expected {
				t.Errorf("LimitOffset(%d, %d) = %q, want %q", tt.limit, tt.offset, result, tt.expected)
			}
		})
	}
}
