package migrations

import (
	"context"

	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/migrations"
)

func addCreateRolesTable(builder *migrations.MigrationBuilder) {
	builder.Add(
		"20260101000001000",
		"create_roles_table",
		func(ctx context.Context, db database.Database) error {
			return migrations.SQL(ctx, db, migrations.DialectSQL{
				Postgres: `
CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    parent VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_parent FOREIGN KEY (parent) REFERENCES roles(name) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_roles_name ON roles(name);
CREATE INDEX IF NOT EXISTS idx_roles_parent ON roles(parent);
`,
				MySQL: `
CREATE TABLE IF NOT EXISTS roles (
    id CHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    parent VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_parent FOREIGN KEY (parent) REFERENCES roles(name) ON DELETE SET NULL,
    INDEX idx_roles_name (name),
    INDEX idx_roles_parent (parent)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`,
				SQLite: `
CREATE TABLE IF NOT EXISTS roles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    parent TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    CONSTRAINT fk_parent FOREIGN KEY (parent) REFERENCES roles(name) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_roles_name ON roles(name);
CREATE INDEX IF NOT EXISTS idx_roles_parent ON roles(parent);
`,
			})
		},
		func(ctx context.Context, db database.Database) error {
			return migrations.DropTableIfExists(ctx, db, "roles")
		},
	)
}
