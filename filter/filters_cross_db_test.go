package filter

import (
	"net/url"
	"testing"

	"github.com/nicolasbonnici/gorest/database/mysql"
	"github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/database/sqlite"
)

func TestFilterSet_ILike_PostgreSQL(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	query, _ := url.ParseQuery("name[ilike]=test")
	fs := NewFilterSet([]string{"name"}, dialect)
	fs.ParseFromQuery(query)

	whereClause, args := fs.BuildWhereClause()

	expected := "WHERE name ILIKE $1"
	if whereClause != expected {
		t.Errorf("expected %q, got %q", expected, whereClause)
	}

	if len(args) != 1 || args[0] != "%test%" {
		t.Errorf("expected args [%%test%%], got %v", args)
	}
}

func TestFilterSet_ILike_MySQL(t *testing.T) {
	dialect := &mysql.MySQLDialect{}
	query, _ := url.ParseQuery("name[ilike]=test")
	fs := NewFilterSet([]string{"name"}, dialect)
	fs.ParseFromQuery(query)

	whereClause, args := fs.BuildWhereClause()

	expected := "WHERE LOWER(name) LIKE LOWER(?)"
	if whereClause != expected {
		t.Errorf("expected %q, got %q", expected, whereClause)
	}

	if len(args) != 1 || args[0] != "%test%" {
		t.Errorf("expected args [%%test%%], got %v", args)
	}
}

func TestFilterSet_ILike_SQLite(t *testing.T) {
	dialect := &sqlite.SQLiteDialect{}
	query, _ := url.ParseQuery("name[ilike]=test")
	fs := NewFilterSet([]string{"name"}, dialect)
	fs.ParseFromQuery(query)

	whereClause, args := fs.BuildWhereClause()

	expected := "WHERE LOWER(name) LIKE LOWER(?)"
	if whereClause != expected {
		t.Errorf("expected %q, got %q", expected, whereClause)
	}

	if len(args) != 1 || args[0] != "%test%" {
		t.Errorf("expected args [%%test%%], got %v", args)
	}
}

func TestFilterSet_In_MultipleDialects(t *testing.T) {
	tests := []struct {
		name          string
		dialect       interface{ Placeholder(n int) string }
		expectedWhere string
	}{
		{
			name:          "PostgreSQL",
			dialect:       &postgres.PostgresDialect{},
			expectedWhere: "WHERE status IN ($1, $2)",
		},
		{
			name:          "MySQL",
			dialect:       &mysql.MySQLDialect{},
			expectedWhere: "WHERE status IN (?, ?)",
		},
		{
			name:          "SQLite",
			dialect:       &sqlite.SQLiteDialect{},
			expectedWhere: "WHERE status IN (?, ?)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, _ := url.ParseQuery("status[]=active&status[]=pending")
			fs := NewFilterSet([]string{"status"}, tt.dialect.(interface {
				Placeholder(n int) string
				CaseInsensitiveLike() string
				LimitOffset(limit, offset int) string
				MapType(dbType string) string
				QuoteIdentifier(name string) string
				ReturningClause(cols ...string) string
				SupportsReturning() bool
			}))
			fs.ParseFromQuery(query)

			whereClause, _ := fs.BuildWhereClause()

			if whereClause != tt.expectedWhere {
				t.Errorf("expected %q, got %q", tt.expectedWhere, whereClause)
			}
		})
	}
}
