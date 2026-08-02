//go:build integration

package refresh

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	authmigrations "github.com/nicolasbonnici/gorest/auth/migrations"
	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/mysql"
	_ "github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/internal/testhelpers"
	"github.com/nicolasbonnici/gorest/migrations"
	"github.com/nicolasbonnici/gorest/query"
)

const refreshTokensMigrationVersion = "20250121000002000"

// setupCrossDB applies the auth migrations to a shared integration database and
// creates a throwaway user. Rows are removed afterwards so repeat runs against
// the same container stay clean.
func setupCrossDB(t *testing.T, db database.Database) (*Service, uuid.UUID) {
	t.Helper()

	ctx := context.Background()

	// `make test-schema` drops application tables but leaves schema_migrations
	// populated, so the migrator would consider this migration already applied
	// and never recreate the table. Clear both so the DDL is genuinely exercised.
	if _, err := db.Exec(ctx, "DROP TABLE IF EXISTS refresh_tokens"); err != nil {
		t.Fatalf("failed to drop refresh_tokens: %v", err)
	}
	deleteRecord, recordArgs, err := query.New(db.Dialect()).
		Delete("schema_migrations").
		Where(query.Eq("version", refreshTokensMigrationVersion)).
		And(query.Eq("source", "auth")).
		Build()
	if err != nil {
		t.Fatalf("failed to build migration reset: %v", err)
	}
	if _, err := db.Exec(ctx, deleteRecord, recordArgs...); err != nil {
		t.Fatalf("failed to reset migration record: %v", err)
	}

	migrator := migrations.NewMigrator(db, authmigrations.GetMigrations())
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
			"email":     fmt.Sprintf("ada+%s@example.com", userID),
		}).
		Build()
	if err != nil {
		t.Fatalf("failed to build user insert: %v", err)
	}
	if _, err := db.Exec(ctx, insertQuery, args...); err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	t.Cleanup(func() {
		delTokens, delArgs, err := query.New(db.Dialect()).
			Delete(tableName).
			Where(query.Eq("user_id", userID)).
			Build()
		if err == nil {
			_, _ = db.Exec(ctx, delTokens, delArgs...)
		}

		delUser, userArgs, err := query.New(db.Dialect()).
			Delete("users").
			Where(query.Eq("id", userID)).
			Build()
		if err == nil {
			_, _ = db.Exec(ctx, delUser, userArgs...)
		}
	})

	return NewService(db, 3600), userID
}

// runLifecycle exercises the paths where dialect differences would show up:
// integer expiry comparison, NULL handling on revoked_at, UUID binding, and the
// transactional rotation.
func runLifecycle(t *testing.T, db database.Database) {
	t.Helper()

	svc, userID := setupCrossDB(t, db)
	ctx := context.Background()

	original, err := svc.Issue(ctx, userID)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	token, err := svc.Validate(ctx, original)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if token.UserID != userID {
		t.Errorf("expected user %s, got %s", userID, token.UserID)
	}
	if token.RevokedAt != nil {
		t.Error("a freshly issued token should not be revoked")
	}

	rotated, replacement, err := svc.Rotate(ctx, original)
	if err != nil {
		t.Fatalf("Rotate failed: %v", err)
	}
	if rotated.UserID != userID {
		t.Errorf("expected user %s, got %s", userID, rotated.UserID)
	}
	if _, err := svc.Validate(ctx, replacement); err != nil {
		t.Fatalf("replacement should be valid: %v", err)
	}

	if _, _, err := svc.Rotate(ctx, original); !errors.Is(err, ErrTokenReuse) {
		t.Fatalf("expected ErrTokenReuse, got %v", err)
	}
	if _, err := svc.Validate(ctx, replacement); !errors.Is(err, ErrTokenReuse) {
		t.Fatalf("expected family revocation, got %v", err)
	}

	// Expiry is compared as an integer, so verify it works on this engine.
	expiredSvc := NewService(db, 3600)
	expiredSvc.ttl = -time.Hour
	expired, err := expiredSvc.Issue(ctx, userID)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	if _, err := svc.Validate(ctx, expired); !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}

	deleted, err := svc.DeleteExpired(ctx, time.Now())
	if err != nil {
		t.Fatalf("DeleteExpired failed: %v", err)
	}
	if deleted < 1 {
		t.Errorf("expected the expired token to be deleted, got %d", deleted)
	}
}

func TestRefreshTokenLifecyclePostgres(t *testing.T) {
	runLifecycle(t, testhelpers.SetupPostgres(t))
}

func TestRefreshTokenLifecycleMySQL(t *testing.T) {
	runLifecycle(t, testhelpers.SetupMySQL(t))
}
