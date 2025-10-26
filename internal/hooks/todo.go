package hooks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nicolasbonnici/gorest/internal/models"
)

type TodoHooks struct {
	NoOpHooks[models.Todo]
}

func (h *TodoHooks) StateProcessor(ctx context.Context, operation Operation, id any, todo *models.Todo) error {
	switch operation {
	case OperationCreate:
		return h.processCreate(ctx, todo)
	case OperationUpdate:
		return h.processUpdate(ctx, id, todo)
	case OperationDelete:
		return h.processDelete(ctx, id)
	default:
		return nil
	}
}

func (h *TodoHooks) processCreate(ctx context.Context, todo *models.Todo) error {
	if strings.TrimSpace(todo.Title) == "" {
		return fmt.Errorf("title is required and cannot be empty")
	}

	if len(todo.Title) > 200 {
		return fmt.Errorf("title cannot exceed 200 characters")
	}

	if len(todo.Content) > 5000 {
		return fmt.Errorf("content cannot exceed 5000 characters")
	}

	todo.Title = strings.TrimSpace(todo.Title)
	todo.Content = strings.TrimSpace(todo.Content)

	if len(todo.Title) > 0 {
		todo.Title = strings.ToUpper(string(todo.Title[0])) + todo.Title[1:]
	}

	if userID := ctx.Value("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			todo.UserId = &uid
		}
	}

	return nil
}

func (h *TodoHooks) processUpdate(ctx context.Context, id any, todo *models.Todo) error {
	if strings.TrimSpace(todo.Title) == "" {
		return fmt.Errorf("title is required and cannot be empty")
	}

	if len(todo.Title) > 200 {
		return fmt.Errorf("title cannot exceed 200 characters")
	}

	if len(todo.Content) > 5000 {
		return fmt.Errorf("content cannot exceed 5000 characters")
	}

	todo.Title = strings.TrimSpace(todo.Title)
	todo.Content = strings.TrimSpace(todo.Content)

	if len(todo.Title) > 0 {
		todo.Title = strings.ToUpper(string(todo.Title[0])) + todo.Title[1:]
	}

	now := time.Now()
	todo.UpdatedAt = &now

	return nil
}

func (h *TodoHooks) processDelete(ctx context.Context, id any) error {
	return nil
}

func (h *TodoHooks) BeforeQuery(ctx context.Context, operation Operation, query string, args []any) (string, []any, error) {
	return query, args, nil
}

func (h *TodoHooks) AfterQuery(ctx context.Context, operation Operation, query string, args []any, result any, err error) error {
	return nil
}

func (h *TodoHooks) OverrideQuery(ctx context.Context, operation Operation, id any, todo *models.Todo) (query string, args []any, skip bool) {
	switch operation {
	case OperationGetAll:
		return h.overrideGetAll(ctx)
	case OperationGetByID:
		return h.overrideGetByID(ctx, id)
	case OperationUpdate:
		return h.overrideUpdate(ctx, id, todo)
	case OperationDelete:
		return h.overrideDelete(ctx, id)
	default:
		return "", nil, false
	}
}

func (h *TodoHooks) overrideGetAll(ctx context.Context) (query string, args []any, skip bool) {
	query = `
		SELECT id, user_id, title, content, updated_at, created_at
		FROM todo
		WHERE 1=1
	`
	args = []any{}

	if userID := ctx.Value("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			query += " AND user_id = $1"
			args = append(args, uid)
		}
	}

	query += " ORDER BY created_at DESC"

	return query, args, true
}

func (h *TodoHooks) overrideGetByID(ctx context.Context, id any) (query string, args []any, skip bool) {
	query = `
		SELECT id, user_id, title, content, updated_at, created_at
		FROM todo
		WHERE id = $1
	`
	args = []any{id}

	if userID := ctx.Value("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			query += " AND user_id = $2"
			args = append(args, uid)
		}
	}

	return query, args, true
}

func (h *TodoHooks) overrideUpdate(ctx context.Context, id any, todo *models.Todo) (query string, args []any, skip bool) {
	query = `
		UPDATE todo
		SET title = $1, content = $2, updated_at = $3
		WHERE id = $4
	`

	now := time.Now()
	args = []any{todo.Title, todo.Content, now, id}

	if userID := ctx.Value("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			query += " AND user_id = $5"
			args = append(args, uid)
		}
	}

	return query, args, true
}

func (h *TodoHooks) overrideDelete(ctx context.Context, id any) (query string, args []any, skip bool) {
	query = "DELETE FROM todo WHERE id = $1"
	args = []any{id}

	if userID := ctx.Value("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			query += " AND user_id = $2"
			args = append(args, uid)
		}
	}

	return query, args, true
}

func (h *TodoHooks) SerializeOne(ctx context.Context, operation Operation, todo *models.Todo) error {
	return nil
}

func (h *TodoHooks) SerializeMany(ctx context.Context, operation Operation, todos *[]models.Todo) error {
	if todos == nil || len(*todos) == 0 {
		return nil
	}

	for i := range *todos {
		if err := h.SerializeOne(ctx, operation, &(*todos)[i]); err != nil {
			return err
		}
	}

	return nil
}
