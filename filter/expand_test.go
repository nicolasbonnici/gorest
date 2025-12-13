package filter

import (
	"net/url"
	"testing"
)

func TestExpandSet_ParseFromQuery(t *testing.T) {
	tests := []struct {
		name          string
		queryString   string
		allowedFields []string
		expectedCount int
		expectedRels  []string
	}{
		{
			name:          "single expand",
			queryString:   "expand[]=user",
			allowedFields: []string{"user"},
			expectedCount: 1,
			expectedRels:  []string{"user"},
		},
		{
			name:          "multiple expands",
			queryString:   "expand[]=user&expand[]=comments",
			allowedFields: []string{"user", "comments"},
			expectedCount: 2,
			expectedRels:  []string{"user", "comments"},
		},
		{
			name:          "disallowed field ignored",
			queryString:   "expand[]=password",
			allowedFields: []string{"user"},
			expectedCount: 0,
			expectedRels:  []string{},
		},
		{
			name:          "duplicate values",
			queryString:   "expand[]=user&expand[]=user",
			allowedFields: []string{"user"},
			expectedCount: 1,
			expectedRels:  []string{"user"},
		},
		{
			name:          "empty query",
			queryString:   "",
			allowedFields: []string{"user"},
			expectedCount: 0,
			expectedRels:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, _ := url.ParseQuery(tt.queryString)
			es := NewExpandSet(tt.allowedFields)
			err := es.ParseFromQuery(query)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(es.Relations) != tt.expectedCount {
				t.Errorf("expected %d relations, got %d", tt.expectedCount, len(es.Relations))
			}

			for _, expected := range tt.expectedRels {
				if !es.Has(expected) {
					t.Errorf("expected relation %s not found", expected)
				}
			}
		})
	}
}
