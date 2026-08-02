package jwt

import (
	"testing"
	"time"
)

const testSecret = "test-secret-key-minimum-32-characters-long"

func TestTokensMintedInSameSecondDiffer(t *testing.T) {
	svc := NewService(testSecret, 900)

	// exp/iat are whole seconds, so without a jti claim these two tokens would
	// carry identical claims and sign to the same string.
	first, err := svc.GenerateToken("user-1")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	second, err := svc.GenerateToken("user-1")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	if first == second {
		t.Error("two tokens minted in the same second are byte-identical")
	}

	for name, token := range map[string]string{"first": first, "second": second} {
		userID, err := svc.ValidateToken(token)
		if err != nil {
			t.Errorf("%s token failed validation: %v", name, err)
		}
		if userID != "user-1" {
			t.Errorf("%s token carried user %q", name, userID)
		}
	}
}

func TestTokenExpiryTracksIssueTime(t *testing.T) {
	svc := NewService(testSecret, 1)

	token, err := svc.GenerateToken("user-1")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if _, err := svc.ValidateToken(token); err != nil {
		t.Fatalf("token should be valid immediately: %v", err)
	}

	// A refresh always mints a full TTL from *now*, which is why a same-second
	// refresh loses nothing: the token it replaces had the same expiry.
	time.Sleep(2 * time.Second)
	if _, err := svc.ValidateToken(token); err == nil {
		t.Error("token should have expired after its TTL elapsed")
	}
}

func TestValidateRejectsWrongSecret(t *testing.T) {
	token, err := NewService(testSecret, 900).GenerateToken("user-1")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	if _, err := NewService("a-different-secret-at-least-32-chars-long", 900).ValidateToken(token); err == nil {
		t.Error("a token signed with another secret should be rejected")
	}
}

func TestTTLAccessor(t *testing.T) {
	if got := NewService(testSecret, 900).TTL(); got != 900 {
		t.Errorf("expected TTL 900, got %d", got)
	}
}
