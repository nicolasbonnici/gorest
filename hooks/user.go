package hooks

import (
	"context"
	"fmt"

	"github.com/nicolasbonnici/gorest/internal/models"
)

type UserHooks struct {
	NoOpHooks[models.User]
}

func (h *UserHooks) StateProcessor(ctx context.Context, operation Operation, id any, user *models.User) error {
	authenticatedUserID := ctx.Value("user_id")
	if authenticatedUserID == nil {
		return fmt.Errorf("authentication required")
	}

	authUserID, ok := authenticatedUserID.(string)
	if !ok {
		return fmt.Errorf("invalid authentication context")
	}

	switch operation {
	case OperationUpdate, OperationDelete:
		if id != nil && id.(string) != authUserID {
			return fmt.Errorf("forbidden: cannot modify other users' data")
		}
	case OperationGetByID:
		if id != nil && id.(string) != authUserID {
			return fmt.Errorf("forbidden: cannot view other users' data")
		}
	}
	return nil
}
