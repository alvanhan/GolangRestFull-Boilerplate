package handler

import (
	"github.com/gofiber/fiber/v2"

	"file-management-service/internal/domain/errors"
	"file-management-service/pkg/response"
)

// handleError maps a domain error to the appropriate HTTP response.
// AppErrors are mapped using their Code and Message. All other errors produce a 500.
func handleError(c *fiber.Ctx, err error) error {
	if appErr, ok := errors.IsAppError(err); ok {
		return response.Error(c, appErr.HTTPStatus(), appErr.Message, appErr.Details)
	}
	return response.InternalError(c)
}
