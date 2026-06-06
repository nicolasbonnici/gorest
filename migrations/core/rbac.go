package core

import (
	"github.com/nicolasbonnici/gorest/migrations"
	rbacmigrations "github.com/nicolasbonnici/gorest/rbac/migrations"
)

// GetRBACMigrations returns migrations for the RBAC system. It is registered
// automatically by gorest.Start when rbac.enabled is true, mirroring the
// auth core wiring — callers don't need to register the rbac plugin manually.
func GetRBACMigrations() migrations.MigrationSource {
	return rbacmigrations.GetMigrations()
}
