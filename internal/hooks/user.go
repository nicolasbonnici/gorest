package hooks

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/nicolasbonnici/gorest/internal/models"
)

type UserHooks struct {
	NoOpHooks[models.User]
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func (h *UserHooks) StateProcessor(ctx context.Context, operation Operation, id any, user *models.User) error {
	switch operation {
	case OperationCreate:
		return h.processCreate(ctx, user)
	case OperationUpdate:
		return h.processUpdate(ctx, id, user)
	case OperationDelete:
		return h.processDelete(ctx, id)
	default:
		return nil
	}
}

func (h *UserHooks) processCreate(ctx context.Context, user *models.User) error {
	if user.Email == "" {
		return fmt.Errorf("email is required")
	}

	user.Email = strings.ToLower(strings.TrimSpace(user.Email))

	if !emailRegex.MatchString(user.Email) {
		return fmt.Errorf("invalid email format: %s", user.Email)
	}

	if strings.TrimSpace(user.Firstname) == "" {
		return fmt.Errorf("firstname is required")
	}

	if strings.TrimSpace(user.Lastname) == "" {
		return fmt.Errorf("lastname is required")
	}

	user.Firstname = strings.TrimSpace(user.Firstname)
	user.Lastname = strings.TrimSpace(user.Lastname)
	user.Firstname = capitalizeFirst(user.Firstname)
	user.Lastname = capitalizeFirst(user.Lastname)

	return nil
}

func (h *UserHooks) processUpdate(ctx context.Context, id any, user *models.User) error {
	if user.Email != "" {
		user.Email = strings.ToLower(strings.TrimSpace(user.Email))
		if !emailRegex.MatchString(user.Email) {
			return fmt.Errorf("invalid email format: %s", user.Email)
		}
	}

	if user.Firstname != "" {
		user.Firstname = capitalizeFirst(strings.TrimSpace(user.Firstname))
	}
	if user.Lastname != "" {
		user.Lastname = capitalizeFirst(strings.TrimSpace(user.Lastname))
	}

	return nil
}

func (h *UserHooks) processDelete(ctx context.Context, id any) error {
	return nil
}

func (h *UserHooks) BeforeQuery(ctx context.Context, operation Operation, query string, args []any) (string, []any, error) {
	return query, args, nil
}

func (h *UserHooks) AfterQuery(ctx context.Context, operation Operation, query string, args []any, result any, err error) error {
	return nil
}

func (h *UserHooks) OverrideQuery(ctx context.Context, operation Operation, id any, user *models.User) (query string, args []any, skip bool) {
	switch operation {
	case OperationGetAll:
		return h.overrideGetAll(ctx)
	case OperationGetByID:
		return h.overrideGetByID(ctx, id)
	case OperationDelete:
		return h.overrideDelete(ctx, id)
	default:
		return "", nil, false
	}
}

func (h *UserHooks) overrideGetAll(ctx context.Context) (query string, args []any, skip bool) {
	query = `
		SELECT id, email, firstname, lastname, created_at, updated_at
		FROM users
		WHERE 1=1
	`
	args = []any{}

	query += " ORDER BY created_at DESC"

	return query, args, true
}

func (h *UserHooks) overrideGetByID(ctx context.Context, id any) (query string, args []any, skip bool) {
	query = `
		SELECT id, email, firstname, lastname, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	args = []any{id}

	return query, args, true
}

func (h *UserHooks) overrideDelete(ctx context.Context, id any) (query string, args []any, skip bool) {
	query = "DELETE FROM users WHERE id = $1"
	args = []any{id}

	if userID := ctx.Value("user_id"); userID != nil {
		if !isAdmin(ctx) {
			query += " AND id = $2"
			args = append(args, userID)
		}
	}

	return query, args, true
}

func (h *UserHooks) SerializeOne(ctx context.Context, operation Operation, user *models.User) error {
	return nil
}

func (h *UserHooks) SerializeMany(ctx context.Context, operation Operation, users *[]models.User) error {
	if users == nil || len(*users) == 0 {
		return nil
	}

	for i := range *users {
		if err := h.SerializeOne(ctx, operation, &(*users)[i]); err != nil {
			return err
		}
	}

	return nil
}

func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

func isAdmin(ctx context.Context) bool {
	return false
}
