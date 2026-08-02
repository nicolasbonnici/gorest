package core

import (
	authmigrations "github.com/nicolasbonnici/gorest/auth/migrations"
	"github.com/nicolasbonnici/gorest/migrations"
)

// GetAuthMigrations returns migrations for the authentication system. It is
// registered automatically by gorest.Start when auth.enabled is true, and
// delegates to auth/migrations so there is a single definition of the schema.
func GetAuthMigrations() migrations.MigrationSource {
	return authmigrations.GetMigrations()
}
