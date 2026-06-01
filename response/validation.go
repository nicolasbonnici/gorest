package response

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

type ValidationExample struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Age      int    `json:"age" validate:"gte=0,lte=130"`
}

func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

func ValidateAndRespond(c fiber.Ctx, s interface{}) error {
	if err := validate.Struct(s); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return SendError(c, fiber.StatusBadRequest, validationErrors.Error())
		}
		return SendError(c, fiber.StatusBadRequest, "Invalid request data")
	}
	return nil
}
