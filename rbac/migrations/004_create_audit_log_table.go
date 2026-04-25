package migrations

import (
	"context"

	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/migrations"
)

func addCreateAuditLogTable(builder *migrations.MigrationBuilder) {
	builder.Add(
		"20260101000004000",
		"create_audit_log_table",
		func(ctx context.Context, db database.Database) error {
			return migrations.SQL(ctx, db, migrations.DialectSQL{
				Postgres: `
CREATE TABLE IF NOT EXISTS rbac_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    user_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    role VARCHAR(100) NOT NULL,
    actor VARCHAR(255) NOT NULL,
    success BOOLEAN NOT NULL DEFAULT TRUE,
    error TEXT
);

CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON rbac_audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp ON rbac_audit_log(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON rbac_audit_log(action);
`,
				MySQL: `
CREATE TABLE IF NOT EXISTS rbac_audit_log (
    id CHAR(36) PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    user_id CHAR(36) NOT NULL,
    action VARCHAR(50) NOT NULL,
    role VARCHAR(100) NOT NULL,
    actor VARCHAR(255) NOT NULL,
    success BOOLEAN NOT NULL DEFAULT TRUE,
    error TEXT,
    INDEX idx_audit_log_user_id (user_id),
    INDEX idx_audit_log_timestamp (timestamp DESC),
    INDEX idx_audit_log_action (action)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`,
				SQLite: `
CREATE TABLE IF NOT EXISTS rbac_audit_log (
    id TEXT PRIMARY KEY,
    timestamp TEXT NOT NULL DEFAULT (datetime('now')),
    user_id TEXT NOT NULL,
    action TEXT NOT NULL,
    role TEXT NOT NULL,
    actor TEXT NOT NULL,
    success INTEGER NOT NULL DEFAULT 1,
    error TEXT
);

CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON rbac_audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp ON rbac_audit_log(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON rbac_audit_log(action);
`,
			})
		},
		func(ctx context.Context, db database.Database) error {
			return migrations.DropTableIfExists(ctx, db, "rbac_audit_log")
		},
	)
}
