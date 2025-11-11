package hooks

import (
	"context"
	"log"

	"example.com/basic-api/generated/models"
	"github.com/nicolasbonnici/gorest/hooks"
)

type TodoHooks struct {
	hooks.NoOpHooks[models.Todo]
}

func (h *TodoHooks) StateProcessor(ctx context.Context, operation hooks.Operation, id any, todo *models.Todo) error {
	if operation == hooks.OperationCreate {
		if userID := ctx.Value("user_id"); userID != nil {
			if uid, ok := userID.(string); ok {
				todo.UserId = &uid
				log.Printf("StateProcessor: Set userId to %s", uid)
			}
		}
	}
	return nil
}

func (h *TodoHooks) SerializeOne(ctx context.Context, operation hooks.Operation, todo *models.Todo) error {
	if todo.UserId != nil {
		log.Printf("SerializeOne: Todo %s has userId: %s", todo.Id, *todo.UserId)
	} else {
		log.Printf("SerializeOne: Todo %s has NIL userId", todo.Id)
	}
	return nil
}

func (h *TodoHooks) SerializeMany(ctx context.Context, operation hooks.Operation, todos *[]models.Todo) error {
	for i := range *todos {
		h.SerializeOne(ctx, operation, &(*todos)[i])
	}
	return nil
}
