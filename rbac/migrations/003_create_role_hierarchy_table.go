package migrations

import (
	"context"

	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/migrations"
)

func addCreateRoleHierarchyTable(builder *migrations.MigrationBuilder) {
	builder.Add(
		"20260101000003000",
		"create_role_hierarchy_table",
		func(ctx context.Context, db database.Database) error {
			return migrations.SQL(ctx, db, migrations.DialectSQL{
				Postgres: `
CREATE TABLE IF NOT EXISTS role_hierarchy (
    parent_role VARCHAR(100) NOT NULL,
    child_role VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (parent_role, child_role),
    CONSTRAINT fk_parent_role FOREIGN KEY (parent_role) REFERENCES roles(name) ON DELETE CASCADE,
    CONSTRAINT fk_child_role FOREIGN KEY (child_role) REFERENCES roles(name) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_role_hierarchy_parent ON role_hierarchy(parent_role);
CREATE INDEX IF NOT EXISTS idx_role_hierarchy_child ON role_hierarchy(child_role);
`,
				MySQL: `
CREATE TABLE IF NOT EXISTS role_hierarchy (
    parent_role VARCHAR(100) NOT NULL,
    child_role VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (parent_role, child_role),
    CONSTRAINT fk_parent_role FOREIGN KEY (parent_role) REFERENCES roles(name) ON DELETE CASCADE,
    CONSTRAINT fk_child_role FOREIGN KEY (child_role) REFERENCES roles(name) ON DELETE CASCADE,
    INDEX idx_role_hierarchy_parent (parent_role),
    INDEX idx_role_hierarchy_child (child_role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`,
				SQLite: `
CREATE TABLE IF NOT EXISTS role_hierarchy (
    parent_role TEXT NOT NULL,
    child_role TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (parent_role, child_role),
    CONSTRAINT fk_parent_role FOREIGN KEY (parent_role) REFERENCES roles(name) ON DELETE CASCADE,
    CONSTRAINT fk_child_role FOREIGN KEY (child_role) REFERENCES roles(name) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_role_hierarchy_parent ON role_hierarchy(parent_role);
CREATE INDEX IF NOT EXISTS idx_role_hierarchy_child ON role_hierarchy(child_role);
`,
			})
		},
		func(ctx context.Context, db database.Database) error {
			return migrations.DropTableIfExists(ctx, db, "role_hierarchy")
		},
	)
}
