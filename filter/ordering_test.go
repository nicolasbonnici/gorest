package filter

import (
	"net/url"
	"testing"
)

func TestOrderSet_ParseFromQuery(t *testing.T) {
	tests := []struct {
		name          string
		queryString   string
		allowedFields []string
		expectedCount int
		expectedDirs  []OrderDirection
	}{
		{
			name:          "single ascending order",
			queryString:   "order[created_at]=asc",
			allowedFields: []string{"created_at"},
			expectedCount: 1,
			expectedDirs:  []OrderDirection{OrderAsc},
		},
		{
			name:          "single descending order",
			queryString:   "order[priority]=desc",
			allowedFields: []string{"priority"},
			expectedCount: 1,
			expectedDirs:  []OrderDirection{OrderDesc},
		},
		{
			name:          "multiple orders",
			queryString:   "order[priority]=desc&order[created_at]=asc",
			allowedFields: []string{"priority", "created_at"},
			expectedCount: 2,
			expectedDirs:  []OrderDirection{OrderAsc, OrderDesc},
		},
		{
			name:          "default to asc for invalid direction",
			queryString:   "order[name]=invalid",
			allowedFields: []string{"name"},
			expectedCount: 1,
			expectedDirs:  []OrderDirection{OrderAsc},
		},
		{
			name:          "skip non-allowed fields",
			queryString:   "order[name]=asc&order[invalid]=desc",
			allowedFields: []string{"name"},
			expectedCount: 1,
			expectedDirs:  []OrderDirection{OrderAsc},
		},
		{
			name:          "no orders",
			queryString:   "",
			allowedFields: []string{"name"},
			expectedCount: 0,
			expectedDirs:  []OrderDirection{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, _ := url.ParseQuery(tt.queryString)
			os := NewOrderSet(tt.allowedFields)
			err := os.ParseFromQuery(query)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(os.Orders) != tt.expectedCount {
				t.Errorf("expected %d orders, got %d", tt.expectedCount, len(os.Orders))
			}

			for i, order := range os.Orders {
				if i < len(tt.expectedDirs) && order.Direction != tt.expectedDirs[i] {
					t.Errorf("order %d: expected direction %s, got %s", i, tt.expectedDirs[i], order.Direction)
				}
			}
		})
	}
}

func TestOrderSet_BuildOrderByClause(t *testing.T) {
	tests := []struct {
		name              string
		queryString       string
		allowedFields     []string
		expectedOrderBy   string
	}{
		{
			name:            "single ascending",
			queryString:     "order[created_at]=asc",
			allowedFields:   []string{"created_at"},
			expectedOrderBy: "ORDER BY created_at ASC",
		},
		{
			name:            "single descending",
			queryString:     "order[priority]=desc",
			allowedFields:   []string{"priority"},
			expectedOrderBy: "ORDER BY priority DESC",
		},
		{
			name:            "multiple fields",
			queryString:     "order[priority]=desc&order[created_at]=asc",
			allowedFields:   []string{"priority", "created_at"},
			expectedOrderBy: "ORDER BY created_at ASC, priority DESC",
		},
		{
			name:            "no orders",
			queryString:     "",
			allowedFields:   []string{"name"},
			expectedOrderBy: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, _ := url.ParseQuery(tt.queryString)
			os := NewOrderSet(tt.allowedFields)
			os.ParseFromQuery(query)

			orderBy := os.BuildOrderByClause()

			if orderBy != tt.expectedOrderBy {
				t.Errorf("expected ORDER BY clause %q, got %q", tt.expectedOrderBy, orderBy)
			}
		})
	}
}
