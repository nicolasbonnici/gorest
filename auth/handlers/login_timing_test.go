package handlers

import (
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestDummyHashCostsAFullRound(t *testing.T) {
	if _, err := bcrypt.Cost(dummyHash); err != nil {
		t.Fatalf("dummyHash is not a valid bcrypt digest: %v", err)
	}
	start := time.Now()
	equalizeLoginTiming("some-candidate-password")
	if d := time.Since(start); d < 10*time.Millisecond {
		t.Fatalf("equalizeLoginTiming returned in %v; it is not doing real work", d)
	}
}
