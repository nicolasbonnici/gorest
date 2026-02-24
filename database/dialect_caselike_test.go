package database

import (
	"testing"
)

func TestBaseDialect_CaseInsensitiveLike(t *testing.T) {
	dialect := &BaseDialect{}
	result := dialect.CaseInsensitiveLike()

	expected := "LOWER"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// Test that CaseInsensitiveLike returns consistent results
func TestBaseDialect_CaseInsensitiveLike_Consistency(t *testing.T) {
	dialect := &BaseDialect{}

	// Call multiple times to ensure consistency
	result1 := dialect.CaseInsensitiveLike()
	result2 := dialect.CaseInsensitiveLike()
	result3 := dialect.CaseInsensitiveLike()

	if result1 != result2 || result2 != result3 {
		t.Errorf("CaseInsensitiveLike() returned inconsistent results: %s, %s, %s", result1, result2, result3)
	}

	// Result should be LOWER for base dialect
	if result1 != "LOWER" {
		t.Errorf("Unexpected CaseInsensitiveLike() result: %s, expected LOWER", result1)
	}
}
