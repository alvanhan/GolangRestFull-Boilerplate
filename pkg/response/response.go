package response

import "github.com/gofiber/fiber/v2"

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

type ErrorInfo struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type Meta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

func Success(c *fiber.Ctx, statusCode int, message string, data interface{}) error {
	return c.Status(statusCode).JSON(Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func SuccessWithMeta(c *fiber.Ctx, statusCode int, message string, data interface{}, meta *Meta) error {
	return c.Status(statusCode).JSON(Response{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

func Error(c *fiber.Ctx, statusCode int, message string, details interface{}) error {
	return c.Status(statusCode).JSON(Response{
		Success: false,
		Message: message,
		Error: &ErrorInfo{
			Code:    statusCode,
			Message: message,
			Details: details,
		},
	})
}

func Created(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func OK(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

func BadRequest(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusBadRequest, message, nil)
}

func Unauthorized(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusUnauthorized, message, nil)
}

func Forbidden(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusForbidden, message, nil)
}

func NotFound(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusNotFound, message, nil)
}

func InternalError(c *fiber.Ctx) error {
	return Error(c, fiber.StatusInternalServerError, "internal server error", nil)
}
