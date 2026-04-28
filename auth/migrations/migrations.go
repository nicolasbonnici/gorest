package migrations

import (
	"context"

	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/migrations"
)

func GetMigrations() migrations.MigrationSource {
	builder := migrations.NewMigrationBuilder("gorest-auth")

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

	return builder.Build()
}
