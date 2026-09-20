package processor

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/response"
)

type ErrorHandler interface {
	HandleError(c fiber.Ctx, err error, operation string) error
}

type DefaultErrorHandler struct{}

func (h *DefaultErrorHandler) HandleError(c fiber.Ctx, err error, operation string) error {
	if operation == "parse" || operation == "parseFilters" || operation == "parseOrdering" {
		return response.SendError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if ferr, ok := err.(*fiber.Error); ok {
		msg := ferr.Message
		if ferr.Code >= 500 {
			msg = "Internal server error"
		}
		return response.SendError(c, ferr.Code, msg)
	}

	// Checked before IsInvalidIDError because crud.ErrInvalidID wraps
	// sql.ErrNoRows: an identifier that cannot address a row has not found one,
	// and answering 404 keeps a probe from learning the key's type.
	if crud.IsNotFoundError(err) {
		return response.SendError(c, fiber.StatusNotFound, "Not found")
	}

	// Reached when the driver, not checkID, was the thing that rejected the
	// value. checkID only sees models whose key field is typed uuid.UUID;
	// several plugins declare `ID string` over a uuid column, and for those the
	// driver error is the only signal available.
	//
	// Which answer is right depends on where the value came from. A malformed
	// id in the path addresses no row, so it is answered 404 exactly as
	// checkID's own rejection would be, and the two paths stay
	// indistinguishable from outside. A malformed value in a body is a request
	// the caller can fix, so it is answered 400. Neither repeats the driver's
	// wording.
	if crud.IsInvalidIDError(err) {
		switch operation {
		case "getById", "update", "delete":
			return response.SendError(c, fiber.StatusNotFound, "Not found")
		default:
			return response.SendError(c, fiber.StatusBadRequest, "Invalid identifier in request")
		}
	}

	if operation == "validate" {
		return response.SendError(c, fiber.StatusBadRequest, response.SafeMessage(err, "Validation failed"))
	}

	return response.SendError(c, fiber.StatusInternalServerError, "Internal server error")
}
