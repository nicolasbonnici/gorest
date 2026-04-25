package migrations

import (
	"github.com/nicolasbonnici/gorest/migrations"
)

func GetMigrations() migrations.MigrationSource {
	builder := migrations.NewMigrationBuilder("gorest-rbac")

	addCreateRolesTable(builder)
	addCreateUserRolesTable(builder)
	addCreateRoleHierarchyTable(builder)
	addCreateAuditLogTable(builder)
	addInsertDefaultRoles(builder)

	return builder.Build()
}
