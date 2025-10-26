package database

import (
	"testing"
)

func TestBaseDialect_LimitOffset(t *testing.T) {
	dialect := &BaseDialect{}

	tests := []struct {
		name     string
		limit    int
		offset   int
		expected string
	}{
		{
			name:     "both limit and offset",
			limit:    10,
			offset:   20,
			expected: "LIMIT 10 OFFSET 20",
		},
		{
			name:     "limit only",
			limit:    10,
			offset:   0,
			expected: "LIMIT 10",
		},
		{
			name:     "offset only",
			limit:    0,
			offset:   20,
			expected: "OFFSET 20",
		},
		{
			name:     "neither limit nor offset",
			limit:    0,
			offset:   0,
			expected: "",
		},
		{
			name:     "large limit and offset",
			limit:    1000,
			offset:   5000,
			expected: "LIMIT 1000 OFFSET 5000",
		},
		{
			name:     "limit 1",
			limit:    1,
			offset:   0,
			expected: "LIMIT 1",
		},
		{
			name:     "offset 1",
			limit:    0,
			offset:   1,
			expected: "OFFSET 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dialect.LimitOffset(tt.limit, tt.offset)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
