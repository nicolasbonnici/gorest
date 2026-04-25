package migrations

import (
	"context"

	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/migrations"
)

func addInsertDefaultRoles(builder *migrations.MigrationBuilder) {
	builder.Add(
		"20260101000005000",
		"insert_default_roles",
		func(ctx context.Context, db database.Database) error {
			return migrations.SQL(ctx, db, migrations.DialectSQL{
				Postgres: `
INSERT INTO roles (name, description) VALUES ('admin', 'System administrator with full access')
ON CONFLICT (name) DO NOTHING;

INSERT INTO roles (name, description) VALUES ('user', 'Basic authenticated user')
ON CONFLICT (name) DO NOTHING;
`,
				MySQL: `
INSERT IGNORE INTO roles (name, description) VALUES ('admin', 'System administrator with full access');
INSERT IGNORE INTO roles (name, description) VALUES ('user', 'Basic authenticated user');
`,
				SQLite: `
INSERT OR IGNORE INTO roles (name, description) VALUES ('admin', 'System administrator with full access');
INSERT OR IGNORE INTO roles (name, description) VALUES ('user', 'Basic authenticated user');
`,
			})
		},
		func(ctx context.Context, db database.Database) error {
			return migrations.SQL(ctx, db, migrations.DialectSQL{
				Postgres: `DELETE FROM roles WHERE name IN ('admin', 'user');`,
				MySQL:    `DELETE FROM roles WHERE name IN ('admin', 'user');`,
				SQLite:   `DELETE FROM roles WHERE name IN ('admin', 'user');`,
			})
		},
	)
}
