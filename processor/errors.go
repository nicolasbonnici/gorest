package processor

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/response"
)

// ErrorHandler defines the interface for handling errors in processor operations.
type ErrorHandler interface {
	HandleError(c *fiber.Ctx, err error, operation string) error
}

// DefaultErrorHandler provides standard error responses for common error types.
type DefaultErrorHandler struct{}

// HandleError handles errors with standard HTTP status codes and messages.
func (h *DefaultErrorHandler) HandleError(c *fiber.Ctx, err error, operation string) error {
	// Parse errors (usually from BodyParser)
	if operation == "parse" || operation == "parseFilters" || operation == "parseOrdering" {
		return response.SendError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Parse errors from Fiber
	if _, ok := err.(*fiber.Error); ok {
		return response.SendError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Invalid ID errors
	if crud.IsInvalidIDError(err) {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	// Not found errors
	if crud.IsNotFoundError(err) {
		return response.SendError(c, fiber.StatusNotFound, "Not found")
	}

	// Validation errors (can be extended with custom validation error types)
	if operation == "validate" {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	// Default to internal server error for database and other errors
	return response.SendError(c, fiber.StatusInternalServerError, err.Error())
}
