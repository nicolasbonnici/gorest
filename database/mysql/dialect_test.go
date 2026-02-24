package mysql

import (
	"testing"

	"github.com/nicolasbonnici/gorest/database"
)

func TestMySQLDialect_Placeholder(t *testing.T) {
	d := &MySQLDialect{}

	for i := 1; i <= 10; i++ {
		result := d.Placeholder(i)
		if result != "?" {
			t.Errorf("Placeholder(%d) = %q, want %q", i, result, "?")
		}
	}
}

func TestMySQLDialect_SupportsReturning(t *testing.T) {
	d := &MySQLDialect{}
	if d.SupportsReturning() {
		t.Error("MySQL should not support RETURNING clause")
	}
}

func TestMySQLDialect_ReturningClause(t *testing.T) {
	d := &MySQLDialect{}
	result := d.ReturningClause("id")
	if result != "" {
		t.Errorf("ReturningClause() = %q, want empty string", result)
	}
}

func TestMySQLDialect_QuoteIdentifier(t *testing.T) {
	d := &MySQLDialect{}

	tests := []struct {
		name     string
		expected string
	}{
		{"id", "`id`"},
		{"user_name", "`user_name`"},
		{"table", "`table`"},
	}

	for _, tt := range tests {
		result := d.QuoteIdentifier(tt.name)
		if result != tt.expected {
			t.Errorf("QuoteIdentifier(%q) = %q, want %q", tt.name, result, tt.expected)
		}
	}
}

func TestMySQLDialect_MapType(t *testing.T) {
	d := &MySQLDialect{}

	tests := []struct {
		stdType  string
		expected string
	}{
		{string(database.TypeInteger), "INT"},
		{string(database.TypeBigInt), "BIGINT"},
		{string(database.TypeString), "VARCHAR(255)"},
		{string(database.TypeText), "TEXT"},
		{string(database.TypeBoolean), "TINYINT(1)"},
		{string(database.TypeTimestamp), "DATETIME"},
		{string(database.TypeUUID), "CHAR(36)"},
		{string(database.TypeJSON), "JSON"},
		{string(database.TypeFloat), "DOUBLE"},
		{string(database.TypeDecimal), "DECIMAL(10,2)"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		result := d.MapType(tt.stdType)
		if result != tt.expected {
			t.Errorf("MapType(%q) = %q, want %q", tt.stdType, result, tt.expected)
		}
	}
}

func TestMySQLDialect_CaseInsensitiveLike(t *testing.T) {
	d := &MySQLDialect{}
	result := d.CaseInsensitiveLike()

	expected := "LOWER"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestMySQLDialect_QuoteIdentifier_EdgeCases(t *testing.T) {
	d := &MySQLDialect{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "users", "`users`"},
		{"SQL keyword SELECT", "select", "`select`"},
		{"SQL keyword FROM", "from", "`from`"},
		{"SQL keyword WHERE", "where", "`where`"},
		{"embedded backtick", "user`name", "`user``name`"},
		{"double backticks", "`users`", "```users```"},
		{"special chars dash", "user-name", "`user-name`"},
		{"special chars underscore", "user_name_123", "`user_name_123`"},
		{"special chars dot", "schema.table", "`schema.table`"},
		{"mixed case", "UserName", "`UserName`"},
		{"all caps", "TABLENAME", "`TABLENAME`"},
		{"unicode", "用户", "`用户`"},
		{"empty string", "", "``"},
		{"numbers", "table123", "`table123`"},
		{"starts with number", "123table", "`123table`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.QuoteIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("QuoteIdentifier(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
