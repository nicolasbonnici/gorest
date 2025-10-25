package helpers

import (
	"testing"
)

func TestParseAcceptHeader(t *testing.T) {
	tests := []struct {
		name     string
		accept   string
		expected []string
	}{
		{
			name:     "Single content type",
			accept:   "application/json",
			expected: []string{"application/json"},
		},
		{
			name:     "Multiple content types",
			accept:   "application/json,application/ld+json",
			expected: []string{"application/json", "application/ld+json"},
		},
		{
			name:     "With quality values",
			accept:   "application/json;q=0.9,text/html;q=0.8",
			expected: []string{"application/json", "text/html"},
		},
		{
			name:     "Empty header",
			accept:   "",
			expected: []string{},
		},
		{
			name:     "With spaces",
			accept:   "application/json, application/ld+json",
			expected: []string{"application/json", "application/ld+json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAcceptHeader(tt.accept)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d content types, got %d", len(tt.expected), len(result))
				return
			}
			for i, ct := range result {
				if ct != tt.expected[i] {
					t.Errorf("Expected content type %s at index %d, got %s", tt.expected[i], i, ct)
				}
			}
		})
	}
}
