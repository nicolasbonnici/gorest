// Package refresh implements opaque, database-backed refresh tokens with
// rotation and reuse detection.
package refresh

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/query"
)

const (
	tableName = "refresh_tokens"

	// DefaultTTL is used when no refresh_ttl is configured.
	DefaultTTL = 30 * 24 * 60 * 60

	tokenBytes = 32
)

var (
	// ErrInvalidToken covers unknown and malformed tokens. It is deliberately
	// indistinguishable from an expired token to callers so that the endpoint
	// cannot be used to probe which tokens exist.
	ErrInvalidToken = errors.New("invalid refresh token")

	ErrExpiredToken = errors.New("expired refresh token")

	// ErrTokenReuse means an already-revoked token was presented, which implies
	// it leaked. The user's whole token family is revoked when this is returned.
	ErrTokenReuse = errors.New("refresh token reuse detected")
)

// Token is a persisted refresh token record. The plaintext is never stored.
type Token struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type Service struct {
	db  database.Database
	ttl time.Duration
}

// New creates a refresh token service. A ttlSeconds of 0 selects DefaultTTL.
func New(db database.Database, ttlSeconds int) *Service {
	if ttlSeconds <= 0 {
		ttlSeconds = DefaultTTL
	}

	return &Service{
		db:  db,
		ttl: time.Duration(ttlSeconds) * time.Second,
	}
}

func (s *Service) TTL() time.Duration {
	return s.ttl
}

// Issue creates a new refresh token for a user and returns the plaintext, which
// is the only time it is available.
func (s *Service) Issue(ctx context.Context, userID uuid.UUID) (string, error) {
	plaintext, err := generateToken()
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(s.ttl)

	qb := query.New(s.db.Dialect()).
		Insert(tableName).
		ValuesMap(map[string]any{
			"id":         uuid.New(),
			"user_id":    userID,
			"token_hash": hashToken(plaintext),
			"expires_at": expiresAt.Unix(),
		})

	queryStr, args, err := qb.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build insert query: %w", err)
	}

	if _, err := s.db.Exec(ctx, queryStr, args...); err != nil {
		return "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return plaintext, nil
}

// Validate resolves a plaintext token to its record, rejecting unknown, expired
// and revoked tokens. A revoked-but-unexpired token means the token leaked and
// was already rotated, so the user's entire family is revoked and ErrTokenReuse
// is returned.
func (s *Service) Validate(ctx context.Context, plaintext string) (*Token, error) {
	token, err := s.lookup(ctx, plaintext)
	if err != nil {
		return nil, err
	}

	if token.RevokedAt != nil {
		if err := s.RevokeAllForUser(ctx, token.UserID); err != nil {
			return nil, fmt.Errorf("failed to revoke token family after reuse: %w", err)
		}
		return nil, ErrTokenReuse
	}

	if time.Now().After(token.ExpiresAt) {
		return nil, ErrExpiredToken
	}

	return token, nil
}

// Rotate validates a token, revokes it, and issues a replacement linked to it.
// The old and new tokens are chained through replaced_by so a leaked token can
// be traced to the session it belongs to.
func (s *Service) Rotate(ctx context.Context, plaintext string) (*Token, string, error) {
	token, err := s.Validate(ctx, plaintext)
	if err != nil {
		return nil, "", err
	}

	newPlaintext, err := generateToken()
	if err != nil {
		return nil, "", err
	}

	newID := uuid.New()
	expiresAt := time.Now().Add(s.ttl)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	insertQuery, insertArgs, err := query.New(s.db.Dialect()).
		Insert(tableName).
		ValuesMap(map[string]any{
			"id":         newID,
			"user_id":    token.UserID,
			"token_hash": hashToken(newPlaintext),
			"expires_at": expiresAt.Unix(),
		}).
		Build()
	if err != nil {
		return nil, "", fmt.Errorf("failed to build insert query: %w", err)
	}

	if _, err := tx.Exec(ctx, insertQuery, insertArgs...); err != nil {
		return nil, "", fmt.Errorf("failed to store rotated token: %w", err)
	}

	revokeQuery, revokeArgs, err := query.New(s.db.Dialect()).
		Update(tableName).
		Set("revoked_at", time.Now().Unix()).
		Set("replaced_by", newID).
		Where(query.Eq("id", token.ID)).
		Build()
	if err != nil {
		return nil, "", fmt.Errorf("failed to build revoke query: %w", err)
	}

	if _, err := tx.Exec(ctx, revokeQuery, revokeArgs...); err != nil {
		return nil, "", fmt.Errorf("failed to revoke rotated token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, "", fmt.Errorf("failed to commit rotation: %w", err)
	}

	return &Token{
		ID:        newID,
		UserID:    token.UserID,
		ExpiresAt: expiresAt,
	}, newPlaintext, nil
}

// Revoke marks a single token as revoked. Revoking an unknown or already-revoked
// token is not an error, so logout is idempotent.
func (s *Service) Revoke(ctx context.Context, plaintext string) error {
	queryStr, args, err := query.New(s.db.Dialect()).
		Update(tableName).
		Set("revoked_at", time.Now().Unix()).
		Where(query.Eq("token_hash", hashToken(plaintext))).
		And(query.IsNull("revoked_at")).
		Build()
	if err != nil {
		return fmt.Errorf("failed to build revoke query: %w", err)
	}

	if _, err := s.db.Exec(ctx, queryStr, args...); err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

// RevokeAllForUser ends every session a user has.
func (s *Service) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	queryStr, args, err := query.New(s.db.Dialect()).
		Update(tableName).
		Set("revoked_at", time.Now().Unix()).
		Where(query.Eq("user_id", userID)).
		And(query.IsNull("revoked_at")).
		Build()
	if err != nil {
		return fmt.Errorf("failed to build revoke query: %w", err)
	}

	if _, err := s.db.Exec(ctx, queryStr, args...); err != nil {
		return fmt.Errorf("failed to revoke user refresh tokens: %w", err)
	}

	return nil
}

// DeleteExpired removes tokens that expired before the cutoff and reports how
// many rows went away. Revoked tokens are kept until they expire so that reuse
// detection still has something to match against.
func (s *Service) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	queryStr, args, err := query.New(s.db.Dialect()).
		Delete(tableName).
		Where(query.Lt("expires_at", before.Unix())).
		Build()
	if err != nil {
		return 0, fmt.Errorf("failed to build cleanup query: %w", err)
	}

	result, err := s.db.Exec(ctx, queryStr, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired refresh tokens: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, nil
	}

	return affected, nil
}

func (s *Service) lookup(ctx context.Context, plaintext string) (*Token, error) {
	if plaintext == "" {
		return nil, ErrInvalidToken
	}

	queryStr, args, err := query.New(s.db.Dialect()).
		Select("id", "user_id", "expires_at", "revoked_at").
		From(tableName).
		Where(query.Eq("token_hash", hashToken(plaintext))).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var (
		id        uuid.UUID
		userID    uuid.UUID
		expiresAt int64
		revokedAt *int64
	)

	err = s.db.QueryRow(ctx, queryStr, args...).Scan(&id, &userID, &expiresAt, &revokedAt)
	if crud.IsNotFoundError(err) {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load refresh token: %w", err)
	}

	token := &Token{
		ID:        id,
		UserID:    userID,
		ExpiresAt: time.Unix(expiresAt, 0),
	}

	if revokedAt != nil {
		revoked := time.Unix(*revokedAt, 0)
		token.RevokedAt = &revoked
	}

	return token, nil
}

func generateToken() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// hashToken uses SHA-256 rather than bcrypt: the input is already 256 bits of
// entropy so stretching buys nothing, and lookup must hit the token_hash index
// instead of bcrypt-comparing every row.
func hashToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
