package hooks

import (
	"context"
	"fmt"

	"github.com/nicolasbonnici/gorest/internal/models"
)

type TodoHooks struct {
	NoOpHooks[models.Todo]
}

func (h *TodoHooks) StateProcessor(ctx context.Context, operation Operation, id any, todo *models.Todo) error {
	authenticatedUserID := ctx.Value("user_id")
	if authenticatedUserID == nil {
		return fmt.Errorf("authentication required")
	}

	authUserID, ok := authenticatedUserID.(string)
	if !ok {
		return fmt.Errorf("invalid authentication context")
	}

	switch operation {
	case OperationCreate:
		todo.UserId = &authUserID

		if todo.Title == "" {
			return fmt.Errorf("title is required")
		}
		if len(todo.Title) < 3 {
			return fmt.Errorf("title must be at least 3 characters")
		}

	case OperationUpdate, OperationDelete:
		if todo.UserId != nil && *todo.UserId != authUserID {
			return fmt.Errorf("forbidden: cannot modify other users' todos")
		}
	}
	return nil
}

func (h *TodoHooks) OverrideQuery(ctx context.Context, operation Operation, id any, todo *models.Todo) (query string, args []any, skip bool) {
	authenticatedUserID := ctx.Value("user_id")
	if authenticatedUserID == nil {
		return "", nil, false
	}

	authUserID, ok := authenticatedUserID.(string)
	if !ok {
		return "", nil, false
	}

	if operation == OperationGetAll {
		query := "SELECT id, user_id, title, content, updated_at, created_at FROM todo WHERE user_id = $1"
		return query, []any{authUserID}, true
	}

	if operation == OperationGetByID {
		query := "SELECT id, user_id, title, content, updated_at, created_at FROM todo WHERE id = $1 AND user_id = $2"
		return query, []any{id, authUserID}, true
	}

	return "", nil, false
}
