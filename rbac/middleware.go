package rbac

import (
	"github.com/gofiber/fiber/v3"

	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/query"
)

// deny emits the same body shape as response.SendError. It is inlined rather
// than imported: response depends on serializer, which depends back on this
// package through crud and hooks, and the cycle breaks the build.
func deny(c fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{"error": message})
}

// The guards below used to be copied into each plugin that needed them, which
// is how two plugins ended up registering their mutating routes with no guard
// at all. Keeping one implementation here means a plugin opts out of
// authorisation explicitly rather than by forgetting to write it.

// RoleLoader reads the caller's roles into the request context so the Require*
// guards can see them. It is deliberately permissive: an anonymous request or a
// failed lookup simply continues with no roles, and the guard that follows is
// what rejects. Mount it before any Require* guard.
func RoleLoader(db database.Database, hierarchy map[string][]string) fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, ok := c.Locals("user_id").(string)
		if !ok || userID == "" {
			return c.Next()
		}

		sql, args, err := query.New(db.Dialect()).
			Select("r.name").
			From("user_roles").
			As("ur").
			JoinAs("roles", "r", query.ColEq("ur.role_id", "r.id")).
			Where(query.Eq("ur.user_id", userID)).
			Build()
		if err != nil {
			return c.Next()
		}

		rows, err := db.Query(c.Context(), sql, args...)
		if err != nil {
			return c.Next()
		}
		defer func() { _ = rows.Close() }()

		var roles []string
		for rows.Next() {
			var role string
			if err := rows.Scan(&role); err != nil {
				continue
			}
			roles = append(roles, role)
		}

		if len(roles) > 0 {
			c.Locals("user_roles", roles)
			c.SetContext(WithRoles(c.Context(), roles))
		}

		return c.Next()
	}
}

// RequireAuthenticated rejects anonymous requests.
func RequireAuthenticated() fiber.Handler {
	return func(c fiber.Ctx) error {
		if uid, ok := c.Locals("user_id").(string); !ok || uid == "" {
			return deny(c, fiber.StatusUnauthorized, "authentication required")
		}
		return c.Next()
	}
}

// RequireRole admits the superuser role, the named role, or anything the
// hierarchy says inherits it.
func RequireRole(hierarchy map[string][]string, superuserRole, requiredRole string) fiber.Handler {
	return RequireAnyRole(hierarchy, superuserRole, requiredRole)
}

// RequireAnyRole admits the superuser role or any one of the named roles.
func RequireAnyRole(hierarchy map[string][]string, superuserRole string, requiredRoles ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		roles, ok := GetRoles(c.Context())
		if !ok || len(roles) == 0 {
			return deny(c, fiber.StatusForbidden, "insufficient permissions")
		}

		for _, role := range roles {
			if superuserRole != "" && role == superuserRole {
				return c.Next()
			}
		}

		if HasAnyRole(roles, requiredRoles, hierarchy) {
			return c.Next()
		}

		return deny(c, fiber.StatusForbidden, "insufficient permissions")
	}
}
