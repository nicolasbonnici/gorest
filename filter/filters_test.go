package filter

import (
	"net/url"
	"strings"
	"testing"

	"github.com/nicolasbonnici/gorest/database/postgres"
)

func TestFilterSet_ParseFromQuery(t *testing.T) {
	dialect := &postgres.PostgresDialect{}

	tests := []struct {
		name          string
		queryString   string
		allowedFields []string
		expectedCount int
		expectedOps   []FilterOperator
	}{
		{
			name:          "simple equality",
			queryString:   "status=active",
			allowedFields: []string{"status"},
			expectedCount: 1,
			expectedOps:   []FilterOperator{OpEqual},
		},
		{
			name:          "multiple values with array syntax",
			queryString:   "status[]=active&status[]=archived",
			allowedFields: []string{"status"},
			expectedCount: 1,
			expectedOps:   []FilterOperator{OpIn},
		},
		{
			name:          "greater than or equal",
			queryString:   "priority[gte]=5",
			allowedFields: []string{"priority"},
			expectedCount: 1,
			expectedOps:   []FilterOperator{OpGreaterThanOrEqual},
		},
		{
			name:          "less than",
			queryString:   "priority[lt]=10",
			allowedFields: []string{"priority"},
			expectedCount: 1,
			expectedOps:   []FilterOperator{OpLessThan},
		},
		{
			name:          "like operator",
			queryString:   "name[like]=test",
			allowedFields: []string{"name"},
			expectedCount: 1,
			expectedOps:   []FilterOperator{OpLike},
		},
		{
			name:          "multiple filters",
			queryString:   "status=active&priority[gte]=5&name[like]=todo",
			allowedFields: []string{"status", "priority", "name"},
			expectedCount: 3,
			expectedOps:   []FilterOperator{},
		},
		{
			name:          "skip non-allowed fields",
			queryString:   "status=active&invalid=test",
			allowedFields: []string{"status"},
			expectedCount: 1,
			expectedOps:   []FilterOperator{OpEqual},
		},
		{
			name:          "skip pagination params",
			queryString:   "status=active&page=2&limit=10",
			allowedFields: []string{"status", "page", "limit"},
			expectedCount: 1,
			expectedOps:   []FilterOperator{OpEqual},
		},
		{
			name:          "multiple values without brackets (auto IN)",
			queryString:   "status=active&status=archived",
			allowedFields: []string{"status"},
			expectedCount: 1,
			expectedOps:   []FilterOperator{OpIn},
		},
		{
			name:          "NOT IN operator with brackets",
			queryString:   "status[nin][]=draft&status[nin][]=deleted",
			allowedFields: []string{"status"},
			expectedCount: 1,
			expectedOps:   []FilterOperator{OpNotIn},
		},
		{
			name:          "NOT IN operator without brackets",
			queryString:   "status[nin]=draft&status[nin]=deleted",
			allowedFields: []string{"status"},
			expectedCount: 1,
			expectedOps:   []FilterOperator{OpNotIn},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, _ := url.ParseQuery(tt.queryString)
			fs := NewFilterSet(tt.allowedFields, dialect)
			err := fs.ParseFromQuery(query)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(fs.Filters) != tt.expectedCount {
				t.Errorf("expected %d filters, got %d", tt.expectedCount, len(fs.Filters))
			}

			for i, filter := range fs.Filters {
				if i < len(tt.expectedOps) && filter.Operator != tt.expectedOps[i] {
					t.Errorf("filter %d: expected operator %s, got %s", i, tt.expectedOps[i], filter.Operator)
				}
			}
		})
	}
}

func TestFilterSet_BuildWhereClause(t *testing.T) {
	dialect := &postgres.PostgresDialect{}

	tests := []struct {
		name          string
		queryString   string
		allowedFields []string
		expectedWhere string
		expectedArgs  int
	}{
		{
			name:          "single equality",
			queryString:   "status=active",
			allowedFields: []string{"status"},
			expectedWhere: "WHERE status = $1",
			expectedArgs:  1,
		},
		{
			name:          "IN operator",
			queryString:   "status[]=active&status[]=archived",
			allowedFields: []string{"status"},
			expectedWhere: "WHERE status IN ($1, $2)",
			expectedArgs:  2,
		},
		{
			name:          "comparison operators",
			queryString:   "priority[gte]=5&priority[lte]=10",
			allowedFields: []string{"priority"},
			expectedWhere: "",
			expectedArgs:  2,
		},
		{
			name:          "LIKE operator",
			queryString:   "name[like]=test",
			allowedFields: []string{"name"},
			expectedWhere: "WHERE name LIKE $1",
			expectedArgs:  1,
		},
		{
			name:          "multiple different fields",
			queryString:   "status=active&priority[gte]=5",
			allowedFields: []string{"status", "priority"},
			expectedWhere: "",
			expectedArgs:  2,
		},
		{
			name:          "no filters",
			queryString:   "",
			allowedFields: []string{"status"},
			expectedWhere: "",
			expectedArgs:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, _ := url.ParseQuery(tt.queryString)
			fs := NewFilterSet(tt.allowedFields, dialect)
			fs.ParseFromQuery(query)

			whereClause, args := fs.BuildWhereClause()

			if tt.expectedWhere != "" && whereClause != tt.expectedWhere {
				t.Errorf("expected WHERE clause %q, got %q", tt.expectedWhere, whereClause)
			}

			if tt.expectedWhere == "" && whereClause != "" {
				if !strings.HasPrefix(whereClause, "WHERE ") {
					t.Errorf("WHERE clause should start with 'WHERE ', got %q", whereClause)
				}
			}

			if len(args) != tt.expectedArgs {
				t.Errorf("expected %d args, got %d", tt.expectedArgs, len(args))
			}
		})
	}
}

func TestFilterSet_LikeOperatorAddsWildcards(t *testing.T) {
	dialect := &postgres.PostgresDialect{}
	query, _ := url.ParseQuery("name[like]=test")
	fs := NewFilterSet([]string{"name"}, dialect)
	fs.ParseFromQuery(query)

	_, args := fs.BuildWhereClause()

	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}

	if args[0] != "%test%" {
		t.Errorf("expected arg to be %%test%%, got %v", args[0])
	}
}
