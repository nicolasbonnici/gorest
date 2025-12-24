package migrations

import (
	"context"
	"fmt"
	"time"

	"github.com/nicolasbonnici/gorest/database"
)

// Example migrations demonstrating Go-based migrations with DAL

// ExampleMigrations shows how to define migrations in Go code
func ExampleMigrations() []Migration {
	builder := NewMigrationBuilder("app")

	// Example 1: Create a table using dialect-aware helpers
	builder.Add(
		"20251222204530123",
		"create_users_table",
		func(ctx context.Context, db database.Database) error {
			// Use helper for cross-database compatibility
			columns := `
				id TEXT PRIMARY KEY,
				email TEXT UNIQUE NOT NULL,
				password TEXT NOT NULL,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			`

			// Adjust for database-specific types
			switch db.DriverName() {
			case "postgres":
				columns = `
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					email TEXT UNIQUE NOT NULL,
					password TEXT NOT NULL,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
				`
			case "mysql":
				columns = `
					id CHAR(36) PRIMARY KEY,
					email VARCHAR(255) UNIQUE NOT NULL,
					password VARCHAR(255) NOT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
				`
			}

			return CreateTableIfNotExists(ctx, db, "users", columns)
		},
		func(ctx context.Context, db database.Database) error {
			return DropTableIfExists(ctx, db, "users")
		},
	)

	// Example 2: Create an index
	builder.Add(
		"20251222204530124",
		"add_users_email_index",
		func(ctx context.Context, db database.Database) error {
			return CreateIndex(ctx, db, "idx_users_email", "users", "email")
		},
		func(ctx context.Context, db database.Database) error {
			return DropIndex(ctx, db, "idx_users_email", "users")
		},
	)

	// Example 3: Add a column
	builder.Add(
		"20251222204530125",
		"add_users_name_column",
		func(ctx context.Context, db database.Database) error {
			columnDef := "name TEXT"
			if db.DriverName() == "mysql" {
				columnDef = "name VARCHAR(255)"
			}
			return AddColumn(ctx, db, "users", columnDef)
		},
		func(ctx context.Context, db database.Database) error {
			return DropColumn(ctx, db, "users", "name")
		},
	)

	// Example 4: Complex migration with multiple operations
	builder.Add(
		"20251222204530126",
		"create_posts_table_with_foreign_key",
		func(ctx context.Context, db database.Database) error {
			return SQL(ctx, db, DialectSQL{
				Postgres: `CREATE TABLE IF NOT EXISTS posts (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					user_id UUID NOT NULL,
					title TEXT NOT NULL,
					content TEXT,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
				)`,
				MySQL: `CREATE TABLE IF NOT EXISTS posts (
					id CHAR(36) PRIMARY KEY,
					user_id CHAR(36) NOT NULL,
					title VARCHAR(255) NOT NULL,
					content TEXT,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
				SQLite: `CREATE TABLE IF NOT EXISTS posts (
					id TEXT PRIMARY KEY,
					user_id TEXT NOT NULL,
					title TEXT NOT NULL,
					content TEXT,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
				)`,
			})
		},
		func(ctx context.Context, db database.Database) error {
			return DropTableIfExists(ctx, db, "posts")
		},
	)

	// Example 5: Data migration
	builder.Add(
		"20251222204530127",
		"populate_default_admin_user",
		func(ctx context.Context, db database.Database) error {
			// Check if admin already exists
			var count int
			row := db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email = ?", "admin@example.com")
			if err := row.Scan(&count); err != nil {
				return fmt.Errorf("failed to check for existing admin: %w", err)
			}

			if count > 0 {
				return nil // Admin already exists
			}

			// Insert admin user using parameterized query
			query := `INSERT INTO users (id, email, password, name, created_at) VALUES (?, ?, ?, ?, ?)`
			args := []interface{}{
				"admin-uuid",
				"admin@example.com",
				"$2a$10$hashedpassword", // bcrypt hash
				"Administrator",
				time.Now(),
			}

			// Adjust for PostgreSQL numbered placeholders
			if db.DriverName() == "postgres" {
				query = `INSERT INTO users (id, email, password, name, created_at) VALUES ($1, $2, $3, $4, $5)`
			}

			_, err := db.Exec(ctx, query, args...)
			return err
		},
		func(ctx context.Context, db database.Database) error {
			// Remove the admin user
			query := "DELETE FROM users WHERE email = ?"
			if db.DriverName() == "postgres" {
				query = "DELETE FROM users WHERE email = $1"
			}
			_, err := db.Exec(ctx, query, "admin@example.com")
			return err
		},
	)

	// Example 6: Using raw SQL when needed (for very specific cases)
	builder.AddSQL(
		"20251222204530128",
		"create_custom_function",
		// This migration uses raw SQL for PostgreSQL-specific function
		`CREATE OR REPLACE FUNCTION update_modified_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = now();
			RETURN NEW;
		END;
		$$ language 'plpgsql'`,
		`DROP FUNCTION IF EXISTS update_modified_column()`,
	)

	return builder.Build().migrations
}

// ExamplePluginMigrations demonstrates plugin migrations with dependencies
func ExamplePluginMigrations() *GoMigrationSource {
	builder := NewMigrationBuilder("auth")

	builder.Add(
		"20251222204530200",
		"create_sessions_table",
		func(ctx context.Context, db database.Database) error {
			return SQL(ctx, db, DialectSQL{
				Postgres: `CREATE TABLE IF NOT EXISTS sessions (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					user_id UUID NOT NULL,
					token TEXT NOT NULL UNIQUE,
					expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
				)`,
				MySQL: `CREATE TABLE IF NOT EXISTS sessions (
					id CHAR(36) PRIMARY KEY,
					user_id CHAR(36) NOT NULL,
					token VARCHAR(255) NOT NULL UNIQUE,
					expires_at TIMESTAMP NOT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
				SQLite: `CREATE TABLE IF NOT EXISTS sessions (
					id TEXT PRIMARY KEY,
					user_id TEXT NOT NULL,
					token TEXT NOT NULL UNIQUE,
					expires_at TIMESTAMP NOT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
				)`,
			})
		},
		func(ctx context.Context, db database.Database) error {
			return DropTableIfExists(ctx, db, "sessions")
		},
	)

	builder.Add(
		"20251222204530201",
		"add_sessions_indexes",
		func(ctx context.Context, db database.Database) error {
			if err := CreateIndex(ctx, db, "idx_sessions_user_id", "sessions", "user_id"); err != nil {
				return err
			}
			return CreateIndex(ctx, db, "idx_sessions_token", "sessions", "token")
		},
		func(ctx context.Context, db database.Database) error {
			if err := DropIndex(ctx, db, "idx_sessions_token", "sessions"); err != nil {
				return err
			}
			return DropIndex(ctx, db, "idx_sessions_user_id", "sessions")
		},
	)

	return builder.Build()
}
