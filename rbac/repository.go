package rbac

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/query"
)

// Repository handles database operations for RBAC management.
type Repository struct {
	db database.Database
}

func NewRepository(db database.Database) *Repository {
	return &Repository{db: db}
}

// ListUsers returns all users with their assigned roles.
// Users without roles are included with an empty Roles slice.
func (r *Repository) ListUsers(ctx context.Context) ([]UserRoles, error) {
	listSQL, args, err := query.New(r.db.Dialect()).
		Select("u.id", "u.email", "r.name", "ur.assigned_at").
		From("users").As("u").
		LeftJoinAs("user_roles", "ur", query.ColEq("u.id", "ur.user_id")).
		LeftJoinAs("roles", "r", query.ColEq("ur.role_id", "r.id")).
		OrderBy("u.id", query.ASC).
		OrderBy("r.name", query.ASC).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.db.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	// Maintain insertion order while aggregating per user.
	type entry = UserRoles
	index := map[string]int{}
	var users []entry

	for rows.Next() {
		var userID, email string
		var roleName sql.NullString
		var assignedAt sql.NullTime

		if err := rows.Scan(&userID, &email, &roleName, &assignedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}

		i, seen := index[userID]
		if !seen {
			i = len(users)
			index[userID] = i
			users = append(users, entry{UserID: userID, Email: email})
		}

		if roleName.Valid {
			users[i].Roles = append(users[i].Roles, roleName.String)
		}

		if assignedAt.Valid && assignedAt.Time.After(users[i].UpdatedAt) {
			users[i].UpdatedAt = assignedAt.Time
		}
	}

	return users, rows.Err()
}

func (r *Repository) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	qb := query.New(r.db.Dialect()).
		Select("r.name").
		From("user_roles").As("ur").
		JoinAs("roles", "r", query.ColEq("ur.role_id", "r.id")).
		Where(query.Eq("ur.user_id", userID)).
		OrderBy("r.name", query.ASC)

	queryStr, args, err := qb.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.db.Query(ctx, queryStr, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query user roles: %w", err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, rows.Err()
}

func (r *Repository) AssignRole(ctx context.Context, userID, roleName, assignedBy string) error {
	if _, err := uuid.Parse(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	var roleID string
	selectQb := query.New(r.db.Dialect()).
		Select("id").From("roles").Where(query.Eq("name", roleName))

	selectSQL, selectArgs, err := selectQb.Build()
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	if err := r.db.QueryRow(ctx, selectSQL, selectArgs...).Scan(&roleID); err != nil {
		if err == sql.ErrNoRows {
			_ = r.logAudit(ctx, userID, "promote", roleName, assignedBy, false, "role not found")
			return ErrRoleNotFound
		}
		return fmt.Errorf("failed to find role: %w", err)
	}

	insertSQL, insertArgs, err := query.New(r.db.Dialect()).
		Insert("user_roles").
		Columns("user_id", "role_id", "assigned_by", "assigned_at").
		Values(userID, roleID, assignedBy, time.Now()).
		OnConflictDoNothing("user_id", "role_id").
		Build()
	if err != nil {
		return fmt.Errorf("failed to build insert: %w", err)
	}

	if _, err := r.db.Exec(ctx, insertSQL, insertArgs...); err != nil {
		_ = r.logAudit(ctx, userID, "promote", roleName, assignedBy, false, err.Error())
		return fmt.Errorf("failed to assign role: %w", err)
	}

	return r.logAudit(ctx, userID, "promote", roleName, assignedBy, true, "")
}

func (r *Repository) RemoveRole(ctx context.Context, userID, roleName, removedBy string) error {
	if _, err := uuid.Parse(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	var roleID string
	selectQb := query.New(r.db.Dialect()).
		Select("id").From("roles").Where(query.Eq("name", roleName))

	selectSQL, selectArgs, err := selectQb.Build()
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	if err := r.db.QueryRow(ctx, selectSQL, selectArgs...).Scan(&roleID); err != nil {
		if err == sql.ErrNoRows {
			_ = r.logAudit(ctx, userID, "demote", roleName, removedBy, false, "role not found")
			return ErrRoleNotFound
		}
		return fmt.Errorf("failed to find role: %w", err)
	}

	deleteQb := query.New(r.db.Dialect()).
		Delete("user_roles").
		Where(query.And(query.Eq("user_id", userID), query.Eq("role_id", roleID)))

	deleteSQL, deleteArgs, err := deleteQb.Build()
	if err != nil {
		return fmt.Errorf("failed to build delete: %w", err)
	}

	result, err := r.db.Exec(ctx, deleteSQL, deleteArgs...)
	if err != nil {
		_ = r.logAudit(ctx, userID, "demote", roleName, removedBy, false, err.Error())
		return fmt.Errorf("failed to remove role: %w", err)
	}

	if n, _ := result.RowsAffected(); n == 0 {
		_ = r.logAudit(ctx, userID, "demote", roleName, removedBy, false, "user does not have role")
		return fmt.Errorf("user does not have role %q", roleName)
	}

	return r.logAudit(ctx, userID, "demote", roleName, removedBy, true, "")
}

func (r *Repository) ListRoles(ctx context.Context) ([]Role, error) {
	qb := query.New(r.db.Dialect()).
		Select("id", "name", "description", "parent", "created_at").
		From("roles").
		OrderBy("name", query.ASC)

	queryStr, args, err := qb.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.db.Query(ctx, queryStr, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query roles: %w", err)
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		var parent sql.NullString
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &parent, &role.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		role.Parent = parent.String
		roles = append(roles, role)
	}

	return roles, rows.Err()
}

func (r *Repository) GetRoleHierarchy(ctx context.Context) (map[string][]string, error) {
	qb := query.New(r.db.Dialect()).
		Select("parent_role", "child_role").
		From("role_hierarchy").
		OrderBy("parent_role", query.ASC).
		OrderBy("child_role", query.ASC)

	queryStr, args, err := qb.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.db.Query(ctx, queryStr, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query hierarchy: %w", err)
	}
	defer rows.Close()

	hierarchy := make(map[string][]string)
	for rows.Next() {
		var parent, child string
		if err := rows.Scan(&parent, &child); err != nil {
			return nil, fmt.Errorf("failed to scan hierarchy: %w", err)
		}
		hierarchy[parent] = append(hierarchy[parent], child)
	}

	return hierarchy, rows.Err()
}

func (r *Repository) logAudit(ctx context.Context, userID, action, role, actor string, success bool, errMsg string) error {
	var errValue any
	if errMsg != "" {
		errValue = errMsg
	}

	qb := query.New(r.db.Dialect()).
		Insert("rbac_audit_log").
		Columns("user_id", "action", "role", "actor", "success", "error").
		Values(userID, action, role, actor, success, errValue)

	insertSQL, args, err := qb.Build()
	if err != nil {
		return fmt.Errorf("failed to build audit insert: %w", err)
	}

	_, err = r.db.Exec(ctx, insertSQL, args...)
	return err
}
