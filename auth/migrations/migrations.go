package migrations

import (
	"context"
	"fmt"
	"regexp"

	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/migrations"
)

func GetMigrations() migrations.MigrationSource {
	// Plugins across the ecosystem (gorest-likeable, gorest-commentable) declare a
	// migration dependency on this exact source name, and unknown dependencies are
	// silently skipped by the resolver — renaming it would quietly lose ordering.
	builder := migrations.NewMigrationBuilder("gorest-core-auth")

	builder.Add(
		"20250121000001000",
		"create_users_table",
		func(ctx context.Context, db database.Database) error {
			if err := migrations.SQL(ctx, db, migrations.DialectSQL{
				Postgres: `CREATE TABLE IF NOT EXISTS users (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					firstname TEXT NOT NULL,
					lastname TEXT NOT NULL,
					email TEXT UNIQUE NOT NULL,
					password TEXT,
					updated_at TIMESTAMP(0) WITH TIME ZONE,
					created_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
				)`,
				MySQL: `CREATE TABLE IF NOT EXISTS users (
					id CHAR(36) PRIMARY KEY,
					firstname TEXT NOT NULL,
					lastname TEXT NOT NULL,
					email VARCHAR(255) UNIQUE NOT NULL,
					password TEXT,
					updated_at TIMESTAMP NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					INDEX idx_user_email (email)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
				SQLite: `CREATE TABLE IF NOT EXISTS users (
					id TEXT PRIMARY KEY,
					firstname TEXT NOT NULL,
					lastname TEXT NOT NULL,
					email TEXT UNIQUE NOT NULL,
					password TEXT,
					updated_at TEXT,
					created_at TEXT NOT NULL DEFAULT (datetime('now'))
				)`,
			}); err != nil {
				return err
			}

			if db.DriverName() == "postgres" {
				return migrations.CreateIndex(ctx, db, "idx_user_email", "users", "email")
			}

			if db.DriverName() == "sqlite" {
				return migrations.CreateIndex(ctx, db, "idx_user_email", "users", "email")
			}

			return nil
		},
		func(ctx context.Context, db database.Database) error {
			if db.DriverName() == "postgres" {
				_ = migrations.DropIndex(ctx, db, "idx_user_email", "users")
			}

			if db.DriverName() == "sqlite" {
				_ = migrations.DropIndex(ctx, db, "idx_user_email", "users")
			}

			return migrations.DropTableIfExists(ctx, db, "users")
		},
	)

	builder.Add(
		"20250121000002000",
		"create_refresh_tokens_table",
		func(ctx context.Context, db database.Database) error {
			// expires_at/revoked_at are epoch seconds rather than native timestamps:
			// the three engines disagree on timestamp storage and text formatting,
			// and integers compare and scan identically on all of them.
			if err := migrations.SQL(ctx, db, migrations.DialectSQL{
				// Postgres adds its foreign key afterwards too, in
				// addPostgresRefreshTokenForeignKey.
				Postgres: `CREATE TABLE IF NOT EXISTS refresh_tokens (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					user_id UUID NOT NULL,
					token_hash TEXT UNIQUE NOT NULL,
					expires_at BIGINT NOT NULL,
					revoked_at BIGINT,
					replaced_by UUID,
					created_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
				)`,
				// MySQL adds its foreign key afterwards, in
				// addMySQLRefreshTokenForeignKey.
				MySQL: `CREATE TABLE IF NOT EXISTS refresh_tokens (
					id CHAR(36) PRIMARY KEY,
					user_id CHAR(36) NOT NULL,
					token_hash VARCHAR(255) UNIQUE NOT NULL,
					expires_at BIGINT NOT NULL,
					revoked_at BIGINT NULL,
					replaced_by CHAR(36) NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					INDEX idx_refresh_token_user (user_id)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
				SQLite: `CREATE TABLE IF NOT EXISTS refresh_tokens (
					id TEXT PRIMARY KEY,
					user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					token_hash TEXT UNIQUE NOT NULL,
					expires_at INTEGER NOT NULL,
					revoked_at INTEGER,
					replaced_by TEXT,
					created_at TEXT NOT NULL DEFAULT (datetime('now'))
				)`,
			}); err != nil {
				return err
			}

			if db.DriverName() == "mysql" {
				return addMySQLRefreshTokenForeignKey(ctx, db)
			}

			// MySQL declared this index inline in its CREATE TABLE.
			if err := migrations.CreateIndex(ctx, db, "idx_refresh_token_user", "refresh_tokens", "user_id"); err != nil {
				return err
			}

			if db.DriverName() == "postgres" {
				return addPostgresRefreshTokenForeignKey(ctx, db)
			}

			return nil
		},
		func(ctx context.Context, db database.Database) error {
			if db.DriverName() == "postgres" || db.DriverName() == "sqlite" {
				_ = migrations.DropIndex(ctx, db, "idx_refresh_token_user", "refresh_tokens")
			}

			return migrations.DropTableIfExists(ctx, db, "refresh_tokens")
		},
	)

	return builder.Build()
}

// addPostgresRefreshTokenForeignKey aligns refresh_tokens.user_id with the
// actual type of users.id before adding the foreign key. Postgres requires the
// two sides to be of comparable types, and an application that brought its own
// users table (GoREST generates APIs from existing schemas) may well key it on
// TEXT or VARCHAR rather than UUID.
func addPostgresRefreshTokenForeignKey(ctx context.Context, db database.Database) error {
	var dataType string
	err := db.QueryRow(ctx, `SELECT data_type FROM information_schema.columns
		WHERE table_name = 'users' AND column_name = 'id'`).Scan(&dataType)
	if err != nil {
		return fmt.Errorf("failed to read users.id type: %w", err)
	}

	if !validPostgresType.MatchString(dataType) {
		return fmt.Errorf("unexpected type for users.id: %q", dataType)
	}

	if dataType != "uuid" {
		alter := fmt.Sprintf("ALTER TABLE refresh_tokens ALTER COLUMN user_id TYPE %s USING user_id::%s", dataType, dataType)
		if _, err := db.Exec(ctx, alter); err != nil {
			return fmt.Errorf("failed to align refresh_tokens.user_id type: %w", err)
		}
	}

	if _, err := db.Exec(ctx, `ALTER TABLE refresh_tokens
		ADD CONSTRAINT refresh_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE`); err != nil {
		return fmt.Errorf("failed to add refresh_tokens foreign key: %w", err)
	}

	return nil
}

// addMySQLRefreshTokenForeignKey aligns refresh_tokens.user_id with the actual
// collation of users.id before adding the foreign key. MySQL rejects a foreign
// key whose columns differ in collation, and users.id may carry either the
// server default or the collation declared by the users migration.
func addMySQLRefreshTokenForeignKey(ctx context.Context, db database.Database) error {
	var collation string
	err := db.QueryRow(ctx, `SELECT COLLATION_NAME FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND COLUMN_NAME = 'id'`).Scan(&collation)
	if err != nil {
		return fmt.Errorf("failed to read users.id collation: %w", err)
	}

	// Guard against interpolating anything unexpected into DDL, which cannot
	// use bound parameters for identifiers.
	if !validCollation.MatchString(collation) {
		return fmt.Errorf("unexpected collation name for users.id: %q", collation)
	}

	if _, err := db.Exec(ctx, "ALTER TABLE refresh_tokens MODIFY user_id CHAR(36) COLLATE "+collation+" NOT NULL"); err != nil {
		return fmt.Errorf("failed to align refresh_tokens.user_id collation: %w", err)
	}

	if _, err := db.Exec(ctx, `ALTER TABLE refresh_tokens
		ADD CONSTRAINT fk_refresh_token_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE`); err != nil {
		return fmt.Errorf("failed to add refresh_tokens foreign key: %w", err)
	}

	return nil
}

var (
	validCollation = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	// information_schema reports types like "character varying", so spaces are
	// allowed here where collation names never contain them.
	validPostgresType = regexp.MustCompile(`^[A-Za-z ]+$`)
)
