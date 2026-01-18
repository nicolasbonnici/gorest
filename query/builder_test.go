package query

import (
	"testing"

	"github.com/nicolasbonnici/gorest/database/mysql"
	"github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/database/sqlite"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		dialect interface{}
	}{
		{
			name:    "PostgreSQL dialect",
			dialect: &postgres.PostgresDialect{},
		},
		{
			name:    "MySQL dialect",
			dialect: &mysql.MySQLDialect{},
		},
		{
			name:    "SQLite dialect",
			dialect: &sqlite.SQLiteDialect{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New(tt.dialect.(interface {
				Placeholder(n int) string
				SupportsReturning() bool
				ReturningClause(cols ...string) string
				LimitOffset(limit, offset int) string
				QuoteIdentifier(name string) string
				MapType(dbType string) string
				CaseInsensitiveLike() string
				SupportsFullJoin() bool
				SupportsWindowFunctions() bool
				SupportsCTE() bool
				SupportsArrays() bool
				OnConflictClause(columns []string, action string) string
				UpsertSupport() bool
			}))

			if builder == nil {
				t.Fatal("Expected builder to be created, got nil")
			}

			if builder.dialect == nil {
				t.Fatal("Expected builder.dialect to be set, got nil")
			}
		})
	}
}

func TestBuilder_Select(t *testing.T) {
	tests := []struct {
		name    string
		dialect interface{}
		columns []string
	}{
		{
			name:    "PostgreSQL - no columns",
			dialect: &postgres.PostgresDialect{},
			columns: []string{},
		},
		{
			name:    "PostgreSQL - single column",
			dialect: &postgres.PostgresDialect{},
			columns: []string{"id"},
		},
		{
			name:    "PostgreSQL - multiple columns",
			dialect: &postgres.PostgresDialect{},
			columns: []string{"id", "name", "email"},
		},
		{
			name:    "MySQL - multiple columns",
			dialect: &mysql.MySQLDialect{},
			columns: []string{"id", "created_at"},
		},
		{
			name:    "SQLite - multiple columns",
			dialect: &sqlite.SQLiteDialect{},
			columns: []string{"id", "status"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New(tt.dialect.(interface {
				Placeholder(n int) string
				SupportsReturning() bool
				ReturningClause(cols ...string) string
				LimitOffset(limit, offset int) string
				QuoteIdentifier(name string) string
				MapType(dbType string) string
				CaseInsensitiveLike() string
				SupportsFullJoin() bool
				SupportsWindowFunctions() bool
				SupportsCTE() bool
				SupportsArrays() bool
				OnConflictClause(columns []string, action string) string
				UpsertSupport() bool
			}))

			selectBuilder := builder.Select(tt.columns...)

			if selectBuilder == nil {
				t.Fatal("Expected SelectBuilder to be created, got nil")
			}

			if selectBuilder.dialect == nil {
				t.Fatal("Expected SelectBuilder.dialect to be set, got nil")
			}

			if len(selectBuilder.columns) != len(tt.columns) {
				t.Errorf("Expected %d columns, got %d", len(tt.columns), len(selectBuilder.columns))
			}

			for i, col := range tt.columns {
				if selectBuilder.columns[i] != col {
					t.Errorf("Expected column %d to be %s, got %s", i, col, selectBuilder.columns[i])
				}
			}
		})
	}
}

func TestBuilder_Insert(t *testing.T) {
	tests := []struct {
		name    string
		dialect interface{}
		table   string
	}{
		{
			name:    "PostgreSQL - users table",
			dialect: &postgres.PostgresDialect{},
			table:   "users",
		},
		{
			name:    "MySQL - products table",
			dialect: &mysql.MySQLDialect{},
			table:   "products",
		},
		{
			name:    "SQLite - orders table",
			dialect: &sqlite.SQLiteDialect{},
			table:   "orders",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New(tt.dialect.(interface {
				Placeholder(n int) string
				SupportsReturning() bool
				ReturningClause(cols ...string) string
				LimitOffset(limit, offset int) string
				QuoteIdentifier(name string) string
				MapType(dbType string) string
				CaseInsensitiveLike() string
				SupportsFullJoin() bool
				SupportsWindowFunctions() bool
				SupportsCTE() bool
				SupportsArrays() bool
				OnConflictClause(columns []string, action string) string
				UpsertSupport() bool
			}))

			insertBuilder := builder.Insert(tt.table)

			if insertBuilder == nil {
				t.Fatal("Expected InsertBuilder to be created, got nil")
			}

			if insertBuilder.dialect == nil {
				t.Fatal("Expected InsertBuilder.dialect to be set, got nil")
			}

			if insertBuilder.table != tt.table {
				t.Errorf("Expected table to be %s, got %s", tt.table, insertBuilder.table)
			}
		})
	}
}

func TestBuilder_Update(t *testing.T) {
	tests := []struct {
		name    string
		dialect interface{}
		table   string
	}{
		{
			name:    "PostgreSQL - users table",
			dialect: &postgres.PostgresDialect{},
			table:   "users",
		},
		{
			name:    "MySQL - products table",
			dialect: &mysql.MySQLDialect{},
			table:   "products",
		},
		{
			name:    "SQLite - orders table",
			dialect: &sqlite.SQLiteDialect{},
			table:   "orders",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New(tt.dialect.(interface {
				Placeholder(n int) string
				SupportsReturning() bool
				ReturningClause(cols ...string) string
				LimitOffset(limit, offset int) string
				QuoteIdentifier(name string) string
				MapType(dbType string) string
				CaseInsensitiveLike() string
				SupportsFullJoin() bool
				SupportsWindowFunctions() bool
				SupportsCTE() bool
				SupportsArrays() bool
				OnConflictClause(columns []string, action string) string
				UpsertSupport() bool
			}))

			updateBuilder := builder.Update(tt.table)

			if updateBuilder == nil {
				t.Fatal("Expected UpdateBuilder to be created, got nil")
			}

			if updateBuilder.dialect == nil {
				t.Fatal("Expected UpdateBuilder.dialect to be set, got nil")
			}

			if updateBuilder.table != tt.table {
				t.Errorf("Expected table to be %s, got %s", tt.table, updateBuilder.table)
			}
		})
	}
}

func TestBuilder_Delete(t *testing.T) {
	tests := []struct {
		name    string
		dialect interface{}
		table   string
	}{
		{
			name:    "PostgreSQL - users table",
			dialect: &postgres.PostgresDialect{},
			table:   "users",
		},
		{
			name:    "MySQL - products table",
			dialect: &mysql.MySQLDialect{},
			table:   "products",
		},
		{
			name:    "SQLite - orders table",
			dialect: &sqlite.SQLiteDialect{},
			table:   "orders",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New(tt.dialect.(interface {
				Placeholder(n int) string
				SupportsReturning() bool
				ReturningClause(cols ...string) string
				LimitOffset(limit, offset int) string
				QuoteIdentifier(name string) string
				MapType(dbType string) string
				CaseInsensitiveLike() string
				SupportsFullJoin() bool
				SupportsWindowFunctions() bool
				SupportsCTE() bool
				SupportsArrays() bool
				OnConflictClause(columns []string, action string) string
				UpsertSupport() bool
			}))

			deleteBuilder := builder.Delete(tt.table)

			if deleteBuilder == nil {
				t.Fatal("Expected DeleteBuilder to be created, got nil")
			}

			if deleteBuilder.dialect == nil {
				t.Fatal("Expected DeleteBuilder.dialect to be set, got nil")
			}

			if deleteBuilder.table != tt.table {
				t.Errorf("Expected table to be %s, got %s", tt.table, deleteBuilder.table)
			}
		})
	}
}

func TestOrder_String(t *testing.T) {
	tests := []struct {
		name     string
		order    Order
		expected string
	}{
		{
			name:     "ASC order",
			order:    ASC,
			expected: "ASC",
		},
		{
			name:     "DESC order",
			order:    DESC,
			expected: "DESC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.order.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestOrderClause(t *testing.T) {
	clause := orderClause{
		column:    "created_at",
		direction: DESC,
	}

	if clause.column != "created_at" {
		t.Errorf("Expected column to be created_at, got %s", clause.column)
	}

	if clause.direction != DESC {
		t.Errorf("Expected direction to be DESC, got %v", clause.direction)
	}
}

func TestBuilder_DialectAccess(t *testing.T) {
	tests := []struct {
		name                  string
		dialect               interface{}
		expectedPlaceholder   string
		expectedQuoteChar     string
		expectedSupportsArray bool
	}{
		{
			name:                  "PostgreSQL dialect access",
			dialect:               &postgres.PostgresDialect{},
			expectedPlaceholder:   "$1",
			expectedQuoteChar:     `"`,
			expectedSupportsArray: true,
		},
		{
			name:                  "MySQL dialect access",
			dialect:               &mysql.MySQLDialect{},
			expectedPlaceholder:   "?",
			expectedQuoteChar:     "`",
			expectedSupportsArray: false,
		},
		{
			name:                  "SQLite dialect access",
			dialect:               &sqlite.SQLiteDialect{},
			expectedPlaceholder:   "?",
			expectedQuoteChar:     `"`,
			expectedSupportsArray: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.dialect.(interface {
				Placeholder(n int) string
				SupportsReturning() bool
				ReturningClause(cols ...string) string
				LimitOffset(limit, offset int) string
				QuoteIdentifier(name string) string
				MapType(dbType string) string
				CaseInsensitiveLike() string
				SupportsFullJoin() bool
				SupportsWindowFunctions() bool
				SupportsCTE() bool
				SupportsArrays() bool
				OnConflictClause(columns []string, action string) string
				UpsertSupport() bool
			})

			builder := New(d)

			selectBuilder := builder.Select("id")
			if selectBuilder.dialect.Placeholder(1) != tt.expectedPlaceholder {
				t.Errorf("Expected placeholder %s, got %s",
					tt.expectedPlaceholder, selectBuilder.dialect.Placeholder(1))
			}

			quoted := selectBuilder.dialect.QuoteIdentifier("test")
			if quoted != tt.expectedQuoteChar+"test"+tt.expectedQuoteChar {
				t.Errorf("Expected quoted identifier %s, got %s",
					tt.expectedQuoteChar+"test"+tt.expectedQuoteChar, quoted)
			}

			if selectBuilder.dialect.SupportsArrays() != tt.expectedSupportsArray {
				t.Errorf("Expected SupportsArrays to be %v, got %v",
					tt.expectedSupportsArray, selectBuilder.dialect.SupportsArrays())
			}

			insertBuilder := builder.Insert("users")
			if insertBuilder.dialect.Placeholder(1) != tt.expectedPlaceholder {
				t.Errorf("Expected placeholder %s, got %s",
					tt.expectedPlaceholder, insertBuilder.dialect.Placeholder(1))
			}

			updateBuilder := builder.Update("users")
			if updateBuilder.dialect.Placeholder(1) != tt.expectedPlaceholder {
				t.Errorf("Expected placeholder %s, got %s",
					tt.expectedPlaceholder, updateBuilder.dialect.Placeholder(1))
			}

			deleteBuilder := builder.Delete("users")
			if deleteBuilder.dialect.Placeholder(1) != tt.expectedPlaceholder {
				t.Errorf("Expected placeholder %s, got %s",
					tt.expectedPlaceholder, deleteBuilder.dialect.Placeholder(1))
			}
		})
	}
}
