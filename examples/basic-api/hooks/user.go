package hooks

import (
	"context"

	"example.com/basic-api/generated/models"
	"github.com/nicolasbonnici/gorest/hooks"
	"golang.org/x/crypto/bcrypt"
)

type UserHooks struct {
	hooks.NoOpHooks[models.User]
}

func (h *UserHooks) StateProcessor(ctx context.Context, operation hooks.Operation, id any, user *models.User) error {
	if operation == hooks.OperationCreate || operation == hooks.OperationUpdate {
		if user.Password != nil && *user.Password != "" {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*user.Password), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			hashed := string(hashedPassword)
			user.Password = &hashed
		}
	}
	return nil
}

func (h *UserHooks) SerializeOne(ctx context.Context, operation hooks.Operation, user *models.User) error {
	user.Password = nil
	return nil
}

func (h *UserHooks) SerializeMany(ctx context.Context, operation hooks.Operation, users *[]models.User) error {
	for i := range *users {
		(*users)[i].Password = nil
	}
	return nil
}
