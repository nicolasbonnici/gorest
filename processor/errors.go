package processor

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/response"
)

type ErrorHandler interface {
	HandleError(c *fiber.Ctx, err error, operation string) error
}

type DefaultErrorHandler struct{}

func (h *DefaultErrorHandler) HandleError(c *fiber.Ctx, err error, operation string) error {
	if operation == "parse" || operation == "parseFilters" || operation == "parseOrdering" {
		return response.SendError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if _, ok := err.(*fiber.Error); ok {
		return response.SendError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if crud.IsInvalidIDError(err) {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if crud.IsNotFoundError(err) {
		return response.SendError(c, fiber.StatusNotFound, "Not found")
	}

	if operation == "validate" {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	return response.SendError(c, fiber.StatusInternalServerError, err.Error())
}
