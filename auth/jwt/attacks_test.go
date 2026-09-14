package jwt

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

const attackSecret = "a-test-signing-secret-of-at-least-32-chars"

// The classic ways a JWT verifier gets talked into trusting a token it never
// issued. Each of these must fail closed.

func TestRejectsAlgNone(t *testing.T) {
	svc := NewService(attackSecret, 900)

	// A well-formed unsigned token: header says none, signature is empty.
	header := b64(t, map[string]string{"alg": "none", "typ": "JWT"})
	claims := b64(t, map[string]any{
		"user_id": "attacker",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})

	for _, forged := range []string{
		header + "." + claims + ".",
		header + "." + claims,
		header + "." + claims + ".bogus",
	} {
		if _, err := svc.ValidateToken(forged); err == nil {
			t.Errorf("alg:none token was accepted: %s", forged)
		}
	}
}

func TestRejectsStrippedSignature(t *testing.T) {
	svc := NewService(attackSecret, 900)

	valid, err := svc.GenerateToken("legitimate-user")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	parts := strings.Split(valid, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 token segments, got %d", len(parts))
	}

	for name, forged := range map[string]string{
		"empty signature":   parts[0] + "." + parts[1] + ".",
		"missing segment":   parts[0] + "." + parts[1],
		"truncated":         parts[0] + "." + parts[1] + "." + parts[2][:len(parts[2])-4],
		"flipped last char": parts[0] + "." + parts[1] + "." + flipLast(parts[2]),
	} {
		if _, err := svc.ValidateToken(forged); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestRejectsTokenSignedWithAnotherSecret(t *testing.T) {
	attacker := NewService("a-completely-different-secret-32-chars-x", 900)
	victim := NewService(attackSecret, 900)

	forged, err := attacker.GenerateToken("attacker")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if _, err := victim.ValidateToken(forged); err == nil {
		t.Error("token signed with a foreign secret was accepted")
	}
}

func TestRejectsExpiredToken(t *testing.T) {
	// A negative TTL mints a token that was already expired when issued.
	svc := NewService(attackSecret, -3600)

	expired, err := svc.GenerateToken("legitimate-user")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if _, err := NewService(attackSecret, 900).ValidateToken(expired); err == nil {
		t.Error("expired token was accepted")
	}
}

// An HS256 verifier that accepts an RS256 header can be handed a token signed
// with the public key as its HMAC secret.
func TestRejectsAlgorithmConfusion(t *testing.T) {
	svc := NewService(attackSecret, 900)

	for _, alg := range []string{"HS384", "HS512", "RS256", "ES256", "PS256"} {
		tok := jwtlib.NewWithClaims(signingMethod(alg), jwtlib.MapClaims{
			"user_id": "attacker",
			"exp":     time.Now().Add(time.Hour).Unix(),
		})
		// Sign with the secret as if it were the right key; only the header
		// algorithm is what is under test.
		signed, err := tok.SignedString([]byte(attackSecret))
		if err != nil {
			continue // asymmetric methods cannot sign with a byte slice
		}
		if _, err := svc.ValidateToken(signed); err == nil {
			t.Errorf("token with alg %s was accepted", alg)
		}
	}
}

func TestRejectsTokenWithoutUserID(t *testing.T) {
	svc := NewService(attackSecret, 900)

	tok := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := tok.SignedString([]byte(attackSecret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := svc.ValidateToken(signed); err == nil {
		t.Error("token with no user_id claim was accepted")
	}
}

func TestRejectsGarbage(t *testing.T) {
	svc := NewService(attackSecret, 900)

	for _, s := range []string{"", ".", "..", "not-a-token", "a.b.c", strings.Repeat("A", 4096)} {
		if _, err := svc.ValidateToken(s); err == nil {
			t.Errorf("garbage %q was accepted", s)
		}
	}
}

func signingMethod(alg string) jwtlib.SigningMethod {
	if m := jwtlib.GetSigningMethod(alg); m != nil {
		return m
	}
	return jwtlib.SigningMethodHS256
}

func b64(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func flipLast(s string) string {
	if s == "" {
		return "x"
	}
	last := s[len(s)-1]
	if last == 'A' {
		return s[:len(s)-1] + "B"
	}
	return s[:len(s)-1] + "A"
}
