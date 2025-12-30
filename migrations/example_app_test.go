package migrations

import (
	"context"
	"testing"

	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
	"github.com/nicolasbonnici/gorest/internal/testhelpers"
)

// This example demonstrates a complete migration workflow for a blog application
func TestExampleBlogApplication(t *testing.T) {
	// Setup database
	db := testhelpers.SetupTestDB(t)
	

	ctx := context.Background()

	// Build migrations for blog application
	builder := NewMigrationBuilder("blog")

	// Migration 1: Create users table
	builder.Add(
		"20251222204530123",
		"create_users_table",
		func(ctx context.Context, db database.Database) error {
			return CreateTableIfNotExists(ctx, db, "users",
				"id TEXT PRIMARY KEY, username TEXT UNIQUE NOT NULL, email TEXT UNIQUE NOT NULL, created_at TEXT DEFAULT CURRENT_TIMESTAMP")
		},
		func(ctx context.Context, db database.Database) error {
			return DropTableIfExists(ctx, db, "users")
		},
	)

	// Migration 2: Create posts table
	builder.Add(
		"20251222204530124",
		"create_posts_table",
		func(ctx context.Context, db database.Database) error {
			return SQL(ctx, db, DialectSQL{
				Postgres: `CREATE TABLE posts (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					user_id UUID NOT NULL,
					title TEXT NOT NULL,
					content TEXT,
					published BOOLEAN DEFAULT FALSE,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
				)`,
				MySQL: `CREATE TABLE posts (
					id CHAR(36) PRIMARY KEY,
					user_id CHAR(36) NOT NULL,
					title VARCHAR(255) NOT NULL,
					content TEXT,
					published BOOLEAN DEFAULT FALSE,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
				SQLite: `CREATE TABLE posts (
					id TEXT PRIMARY KEY,
					user_id TEXT NOT NULL,
					title TEXT NOT NULL,
					content TEXT,
					published INTEGER DEFAULT 0,
					created_at TEXT DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
				)`,
			})
		},
		func(ctx context.Context, db database.Database) error {
			return DropTableIfExists(ctx, db, "posts")
		},
	)

	// Migration 3: Add indexes for performance
	builder.Add(
		"20251222204530125",
		"add_performance_indexes",
		func(ctx context.Context, db database.Database) error {
			if err := CreateIndex(ctx, db, "idx_users_username", "users", "username"); err != nil {
				return err
			}
			if err := CreateIndex(ctx, db, "idx_users_email", "users", "email"); err != nil {
				return err
			}
			if err := CreateIndex(ctx, db, "idx_posts_user_id", "posts", "user_id"); err != nil {
				return err
			}
			return CreateIndex(ctx, db, "idx_posts_published", "posts", "published")
		},
		func(ctx context.Context, db database.Database) error {
			DropIndex(ctx, db, "idx_posts_published", "posts")
			DropIndex(ctx, db, "idx_posts_user_id", "posts")
			DropIndex(ctx, db, "idx_users_email", "users")
			DropIndex(ctx, db, "idx_users_username", "users")
			return nil
		},
	)

	// Migration 4: Add comments table
	builder.Add(
		"20251222204530126",
		"create_comments_table",
		func(ctx context.Context, db database.Database) error {
			return CreateTableIfNotExists(ctx, db, "comments",
				`id TEXT PRIMARY KEY,
				post_id TEXT NOT NULL,
				user_id TEXT NOT NULL,
				content TEXT NOT NULL,
				created_at TEXT DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE`)
		},
		func(ctx context.Context, db database.Database) error {
			return DropTableIfExists(ctx, db, "comments")
		},
	)

	// Migration 5: Seed initial admin user
	builder.Add(
		"20251222204530127",
		"seed_admin_user",
		func(ctx context.Context, db database.Database) error {
			query := "INSERT INTO users (id, username, email) VALUES (?, ?, ?)"
			_, err := db.Exec(ctx, query, "admin-1", "admin", "admin@example.com")
			return err
		},
		func(ctx context.Context, db database.Database) error {
			query := "DELETE FROM users WHERE id = ?"
			_, err := db.Exec(ctx, query, "admin-1")
			return err
		},
	)

	// Create migration source
	source := builder.Build()

	// Create migrator
	migrator := NewMigrator(db, source)

	// Check initial status
	statuses, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	t.Logf("Initial state: %d migrations pending", len(statuses))

	// Run all migrations
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	t.Log("✓ All migrations applied")

	// Verify database state
	statuses, err = migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	appliedCount := 0
	for _, status := range statuses {
		if status.Applied {
			appliedCount++
			t.Logf("  ✓ Applied: %s", status.Migration.FullName())
		}
	}

	if appliedCount != 5 {
		t.Errorf("Expected 5 applied migrations, got %d", appliedCount)
	}

	// Verify tables exist
	tables := []string{"users", "posts", "comments"}
	for _, table := range tables {
		_, err := db.Query(ctx, "SELECT * FROM "+table+" LIMIT 1")
		if err != nil {
			t.Errorf("Table %s should exist: %v", table, err)
		} else {
			t.Logf("  ✓ Table exists: %s", table)
		}
	}

	// Verify admin user was seeded
	var count int
	row := db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE username = ?", "admin")
	if err := row.Scan(&count); err != nil {
		t.Errorf("Failed to query admin user: %v", err)
	} else if count != 1 {
		t.Errorf("Expected 1 admin user, got %d", count)
	} else {
		t.Log("  ✓ Admin user seeded")
	}

	t.Log("\n✓ Blog application migration test completed successfully")
}

// This example shows how to use the migration system with custom logic
func TestExampleCustomMigration(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	

	ctx := context.Background()

	builder := NewMigrationBuilder("custom")

	// Migration with complex business logic
	builder.Add(
		"20251222204530200",
		"migrate_user_data",
		func(ctx context.Context, db database.Database) error {
			// Create new table
			if err := CreateTableIfNotExists(ctx, db, "users_new",
				"id TEXT PRIMARY KEY, email TEXT, status TEXT DEFAULT 'active'"); err != nil {
				return err
			}

			// Create old table for demonstration
			if err := CreateTableIfNotExists(ctx, db, "users_old",
				"id TEXT PRIMARY KEY, email TEXT"); err != nil {
				return err
			}

			// Insert test data
			_, _ = db.Exec(ctx, "INSERT INTO users_old (id, email) VALUES (?, ?)", "1", "user1@example.com")
			_, _ = db.Exec(ctx, "INSERT INTO users_old (id, email) VALUES (?, ?)", "2", "user2@example.com")

			// Migrate data with transformation
			// Note: Read all data first before writing (SQLite limitation)
			rows, err := db.Query(ctx, "SELECT id, email FROM users_old")
			if err != nil {
				return err
			}

			// Collect all rows first
			type userData struct {
				id    string
				email string
			}
			var users []userData
			for rows.Next() {
				var u userData
				if err := rows.Scan(&u.id, &u.email); err != nil {
					rows.Close()
					return err
				}
				users = append(users, u)
			}
			rows.Close()

			// Now insert the migrated data
			for _, u := range users {
				_, err := db.Exec(ctx,
					"INSERT INTO users_new (id, email, status) VALUES (?, ?, ?)",
					u.id, u.email, "migrated")
				if err != nil {
					return err
				}
			}

			// Drop old table
			return DropTableIfExists(ctx, db, "users_old")
		},
		func(ctx context.Context, db database.Database) error {
			return DropTableIfExists(ctx, db, "users_new")
		},
	)

	source := builder.Build()
	migrator := NewMigrator(db, source)

	// Apply migration
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("Failed to apply migration: %v", err)
	}

	// Verify data was migrated
	var count int
	row := db.QueryRow(ctx, "SELECT COUNT(*) FROM users_new WHERE status = ?", "migrated")
	if err := row.Scan(&count); err != nil {
		t.Errorf("Failed to query migrated users: %v", err)
	} else if count != 2 {
		t.Errorf("Expected 2 migrated users, got %d", count)
	} else {
		t.Log("✓ Data migration successful")
	}
}
