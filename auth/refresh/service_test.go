package refresh

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
	"github.com/nicolasbonnici/gorest/internal/testhelpers"
	"github.com/nicolasbonnici/gorest/migrations"
	coremigrations "github.com/nicolasbonnici/gorest/migrations/core"
	"github.com/nicolasbonnici/gorest/query"
)

// setup builds a database with the real auth migrations applied and returns a
// service plus a user row the refresh tokens can point at.
func setup(t *testing.T) (database.Database, *Service, uuid.UUID) {
	t.Helper()

	db := testhelpers.SetupSQLite(t)
	ctx := context.Background()

	migrator := migrations.NewMigrator(db, coremigrations.GetAuthMigrations())
	if err := migrator.Up(ctx); err != nil && !errors.Is(err, migrations.ErrNoPendingMigrations) {
		t.Fatalf("failed to run auth migrations: %v", err)
	}

	userID := uuid.New()
	insertQuery, args, err := query.New(db.Dialect()).
		Insert("users").
		ValuesMap(map[string]any{
			"id":        userID,
			"firstname": "Ada",
			"lastname":  "Lovelace",
			"email":     "ada@example.com",
		}).
		Build()
	if err != nil {
		t.Fatalf("failed to build user insert: %v", err)
	}
	if _, err := db.Exec(ctx, insertQuery, args...); err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	return db, New(db, 3600), userID
}

func TestIssueAndValidate(t *testing.T) {
	_, svc, userID := setup(t)
	ctx := context.Background()

	plaintext, err := svc.Issue(ctx, userID)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	if plaintext == "" {
		t.Fatal("Issue returned an empty token")
	}

	token, err := svc.Validate(ctx, plaintext)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if token.UserID != userID {
		t.Errorf("expected user %s, got %s", userID, token.UserID)
	}
}

func TestIssueDoesNotStorePlaintext(t *testing.T) {
	db, svc, userID := setup(t)
	ctx := context.Background()

	plaintext, err := svc.Issue(ctx, userID)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	selectQuery, args, err := query.New(db.Dialect()).
		Select("token_hash").
		From(tableName).
		Where(query.Eq("user_id", userID)).
		Build()
	if err != nil {
		t.Fatalf("failed to build select: %v", err)
	}

	var stored string
	if err := db.QueryRow(ctx, selectQuery, args...).Scan(&stored); err != nil {
		t.Fatalf("failed to read stored token: %v", err)
	}

	if stored == plaintext {
		t.Error("plaintext refresh token was stored in the database")
	}
	if stored != hashToken(plaintext) {
		t.Error("stored value is not the hash of the issued token")
	}
}

func TestValidateUnknownToken(t *testing.T) {
	_, svc, _ := setup(t)

	_, err := svc.Validate(context.Background(), "not-a-real-token")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestValidateEmptyToken(t *testing.T) {
	_, svc, _ := setup(t)

	_, err := svc.Validate(context.Background(), "")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestValidateExpiredToken(t *testing.T) {
	db, _, userID := setup(t)
	ctx := context.Background()

	// Negative TTL issues a token that is already past its expiry.
	svc := New(db, 3600)
	svc.ttl = -time.Minute

	plaintext, err := svc.Issue(ctx, userID)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	_, err = svc.Validate(ctx, plaintext)
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}

func TestRotateInvalidatesOldToken(t *testing.T) {
	_, svc, userID := setup(t)
	ctx := context.Background()

	original, err := svc.Issue(ctx, userID)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	rotated, replacement, err := svc.Rotate(ctx, original)
	if err != nil {
		t.Fatalf("Rotate failed: %v", err)
	}
	if replacement == original {
		t.Fatal("Rotate returned the same token")
	}
	if rotated.UserID != userID {
		t.Errorf("expected user %s, got %s", userID, rotated.UserID)
	}

	if _, err := svc.Validate(ctx, replacement); err != nil {
		t.Fatalf("replacement token should be valid, got %v", err)
	}
}

func TestRotateChainsReplacedBy(t *testing.T) {
	db, svc, userID := setup(t)
	ctx := context.Background()

	original, err := svc.Issue(ctx, userID)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	rotated, _, err := svc.Rotate(ctx, original)
	if err != nil {
		t.Fatalf("Rotate failed: %v", err)
	}

	selectQuery, args, err := query.New(db.Dialect()).
		Select("replaced_by").
		From(tableName).
		Where(query.Eq("token_hash", hashToken(original))).
		Build()
	if err != nil {
		t.Fatalf("failed to build select: %v", err)
	}

	var replacedBy uuid.UUID
	if err := db.QueryRow(ctx, selectQuery, args...).Scan(&replacedBy); err != nil {
		t.Fatalf("failed to read replaced_by: %v", err)
	}

	if replacedBy != rotated.ID {
		t.Errorf("expected replaced_by %s, got %s", rotated.ID, replacedBy)
	}
}

func TestReuseRevokesFamily(t *testing.T) {
	_, svc, userID := setup(t)
	ctx := context.Background()

	original, err := svc.Issue(ctx, userID)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	_, replacement, err := svc.Rotate(ctx, original)
	if err != nil {
		t.Fatalf("Rotate failed: %v", err)
	}

	// Replaying the already-rotated token is the leak signal.
	if _, _, err := svc.Rotate(ctx, original); !errors.Is(err, ErrTokenReuse) {
		t.Fatalf("expected ErrTokenReuse, got %v", err)
	}

	// The still-live replacement must have been revoked along with it.
	if _, err := svc.Validate(ctx, replacement); !errors.Is(err, ErrTokenReuse) {
		t.Fatalf("expected replacement to be revoked, got %v", err)
	}
}

func TestRevoke(t *testing.T) {
	_, svc, userID := setup(t)
	ctx := context.Background()

	plaintext, err := svc.Issue(ctx, userID)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	if err := svc.Revoke(ctx, plaintext); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	if _, err := svc.Validate(ctx, plaintext); !errors.Is(err, ErrTokenReuse) {
		t.Fatalf("expected revoked token to be rejected, got %v", err)
	}
}

func TestRevokeIsIdempotent(t *testing.T) {
	_, svc, userID := setup(t)
	ctx := context.Background()

	plaintext, err := svc.Issue(ctx, userID)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	if err := svc.Revoke(ctx, plaintext); err != nil {
		t.Fatalf("first Revoke failed: %v", err)
	}
	if err := svc.Revoke(ctx, plaintext); err != nil {
		t.Fatalf("second Revoke failed: %v", err)
	}
	if err := svc.Revoke(ctx, "never-existed"); err != nil {
		t.Fatalf("revoking an unknown token should not error, got %v", err)
	}
}

func TestRevokeAllForUser(t *testing.T) {
	_, svc, userID := setup(t)
	ctx := context.Background()

	first, err := svc.Issue(ctx, userID)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	second, err := svc.Issue(ctx, userID)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	if err := svc.RevokeAllForUser(ctx, userID); err != nil {
		t.Fatalf("RevokeAllForUser failed: %v", err)
	}

	for name, token := range map[string]string{"first": first, "second": second} {
		if _, err := svc.Validate(ctx, token); err == nil {
			t.Errorf("%s token should have been revoked", name)
		}
	}
}

func TestDeleteExpired(t *testing.T) {
	db, svc, userID := setup(t)
	ctx := context.Background()

	live, err := svc.Issue(ctx, userID)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	expiredSvc := New(db, 3600)
	expiredSvc.ttl = -time.Hour
	if _, err := expiredSvc.Issue(ctx, userID); err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	deleted, err := svc.DeleteExpired(ctx, time.Now())
	if err != nil {
		t.Fatalf("DeleteExpired failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted token, got %d", deleted)
	}

	if _, err := svc.Validate(ctx, live); err != nil {
		t.Errorf("live token should have survived cleanup, got %v", err)
	}
}

func TestTokensAreUnique(t *testing.T) {
	_, svc, userID := setup(t)
	ctx := context.Background()

	seen := make(map[string]bool)
	for range 50 {
		plaintext, err := svc.Issue(ctx, userID)
		if err != nil {
			t.Fatalf("Issue failed: %v", err)
		}
		if seen[plaintext] {
			t.Fatal("Issue produced a duplicate token")
		}
		seen[plaintext] = true
	}
}

func TestDefaultTTLApplied(t *testing.T) {
	db := testhelpers.SetupSQLite(t)

	if got := New(db, 0).TTL(); got != DefaultTTL*time.Second {
		t.Errorf("expected default TTL, got %v", got)
	}
	if got := New(db, -5).TTL(); got != DefaultTTL*time.Second {
		t.Errorf("expected default TTL for negative input, got %v", got)
	}
}
