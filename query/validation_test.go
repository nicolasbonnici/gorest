package query

import (
	"strings"
	"testing"
)

func TestValidateIdentifier(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		errorMsg  string
	}{
		// Valid identifiers
		{"valid simple", "users", false, ""},
		{"valid with underscore", "user_name", false, ""},
		{"valid starts with underscore", "_private", false, ""},
		{"valid mixed case", "UserName", false, ""},
		{"valid with numbers", "user123", false, ""},
		{"valid long", strings.Repeat("a", 63), false, ""},

		// Invalid identifiers
		{"empty", "", true, "cannot be empty"},
		{"too long", strings.Repeat("a", 64), true, "exceeds maximum length"},
		{"starts with number", "123user", true, "invalid characters"},
		{"contains hyphen", "user-name", true, "invalid characters"},
		{"contains space", "user name", true, "invalid characters"},
		{"contains dot", "user.name", true, "invalid characters"},
		{"contains special char", "user@name", true, "invalid characters"},
		{"SQL injection attempt", "name'; DROP TABLE users--", true, "invalid characters"},

		// Reserved words
		{"reserved SELECT", "SELECT", true, "reserved word"},
		{"reserved select lowercase", "select", true, "reserved word"},
		{"reserved WHERE", "WHERE", true, "reserved word"},
		{"reserved FROM", "FROM", true, "reserved word"},
		{"reserved JOIN", "JOIN", true, "reserved word"},
		{"reserved ORDER", "ORDER", true, "reserved word"},
		{"reserved GROUP", "GROUP", true, "reserved word"},
		{"reserved UNION", "UNION", true, "reserved word"},
		{"reserved INSERT", "INSERT", true, "reserved word"},
		{"reserved UPDATE", "UPDATE", true, "reserved word"},
		{"reserved DELETE", "DELETE", true, "reserved word"},
		{"reserved CREATE", "CREATE", true, "reserved word"},
		{"reserved DROP", "DROP", true, "reserved word"},
		{"reserved TABLE", "TABLE", true, "reserved word"},
		{"reserved NULL", "NULL", true, "reserved word"},
		{"reserved TRUE", "TRUE", true, "reserved word"},
		{"reserved FALSE", "FALSE", true, "reserved word"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIdentifier(tt.input)
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateIdentifier(%q) expected error containing %q, got nil", tt.input, tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateIdentifier(%q) error = %v, want error containing %q", tt.input, err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateIdentifier(%q) unexpected error: %v", tt.input, err)
				}
			}
		})
	}
}

func TestValidateLimit(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		wantError bool
	}{
		{"valid zero", 0, false},
		{"valid small", 10, false},
		{"valid medium", 1000, false},
		{"valid max", MaxLimitValue, false},
		{"invalid negative", -1, true},
		{"invalid too large", MaxLimitValue + 1, true},
		{"invalid very large", 999999999, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLimit(tt.limit)
			if tt.wantError && err == nil {
				t.Errorf("ValidateLimit(%d) expected error, got nil", tt.limit)
			}
			if !tt.wantError && err != nil {
				t.Errorf("ValidateLimit(%d) unexpected error: %v", tt.limit, err)
			}
		})
	}
}

func TestValidateOffset(t *testing.T) {
	tests := []struct {
		name      string
		offset    int
		wantError bool
	}{
		{"valid zero", 0, false},
		{"valid small", 100, false},
		{"valid medium", 50000, false},
		{"valid max", MaxOffsetValue, false},
		{"invalid negative", -1, true},
		{"invalid too large", MaxOffsetValue + 1, true},
		{"invalid very large", 999999999, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOffset(tt.offset)
			if tt.wantError && err == nil {
				t.Errorf("ValidateOffset(%d) expected error, got nil", tt.offset)
			}
			if !tt.wantError && err != nil {
				t.Errorf("ValidateOffset(%d) unexpected error: %v", tt.offset, err)
			}
		})
	}
}

func TestValidateWindowFrame(t *testing.T) {
	tests := []struct {
		name      string
		frame     string
		wantError bool
	}{
		// Valid frames
		{"empty", "", false},
		{"ROWS UNBOUNDED PRECEDING", "ROWS UNBOUNDED PRECEDING", false},
		{"ROWS CURRENT ROW", "ROWS CURRENT ROW", false},
		{"ROWS BETWEEN", "ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW", false},
		{"ROWS BETWEEN both unbounded", "ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING", false},
		{"RANGE UNBOUNDED PRECEDING", "RANGE UNBOUNDED PRECEDING", false},
		{"RANGE CURRENT ROW", "RANGE CURRENT ROW", false},
		{"RANGE BETWEEN", "RANGE BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW", false},
		{"lowercase rows", "rows unbounded preceding", false},
		{"mixed case", "Rows Unbounded Preceding", false},
		{"numeric offset preceding", "ROWS 5 PRECEDING", false},
		{"numeric offset following", "ROWS 10 FOLLOWING", false},
		{"numeric BETWEEN", "ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING", false},
		{"numeric RANGE", "RANGE 3 PRECEDING", false},

		// Invalid frames
		{"invalid keyword", "INVALID UNBOUNDED PRECEDING", true},
		{"missing keyword", "UNBOUNDED PRECEDING", true},
		{"invalid structure", "ROWS SOMETHING WRONG", true},
		{"SQL injection in frame", "ROWS; DROP TABLE users--", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWindowFrame(tt.frame)
			if tt.wantError && err == nil {
				t.Errorf("ValidateWindowFrame(%q) expected error, got nil", tt.frame)
			}
			if !tt.wantError && err != nil {
				t.Errorf("ValidateWindowFrame(%q) unexpected error: %v", tt.frame, err)
			}
		})
	}
}

func TestMaxConstants(t *testing.T) {
	// Verify constants are reasonable
	if MaxIdentifierLength != 63 {
		t.Errorf("MaxIdentifierLength = %d, want 63 (PostgreSQL limit)", MaxIdentifierLength)
	}
	if MaxLimitValue != 10000 {
		t.Errorf("MaxLimitValue = %d, want 10000", MaxLimitValue)
	}
	if MaxOffsetValue != 1000000 {
		t.Errorf("MaxOffsetValue = %d, want 1000000", MaxOffsetValue)
	}
	if MaxSubqueryDepth != 3 {
		t.Errorf("MaxSubqueryDepth = %d, want 3", MaxSubqueryDepth)
	}
}

func TestReservedWordsCompleteness(t *testing.T) {
	// Verify common reserved words are present
	requiredWords := []string{
		"SELECT", "INSERT", "UPDATE", "DELETE", "FROM", "WHERE",
		"JOIN", "ORDER", "GROUP", "UNION", "CREATE", "DROP", "TABLE",
	}

	for _, word := range requiredWords {
		if !sqlReservedWords[word] {
			t.Errorf("sqlReservedWords missing required word: %s", word)
		}
	}
}

func BenchmarkValidateIdentifier(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ValidateIdentifier("user_name")
	}
}

func BenchmarkValidateIdentifierReserved(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ValidateIdentifier("SELECT")
	}
}
