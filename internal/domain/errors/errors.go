package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError is the standard application error type carrying an HTTP status code,
// a human-readable message, optional structured details, and an optional wrapped cause.
type AppError struct {
	Code    int
	Message string
	Details interface{}
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

func (e *AppError) HTTPStatus() int { return e.Code }

// New creates an AppError with the given HTTP status code and message.
func New(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// Wrap creates an AppError that wraps an existing error.
func Wrap(code int, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

// NotFound returns a 404 error for a missing resource.
func NotFound(resource string) *AppError {
	return &AppError{
		Code:    http.StatusNotFound,
		Message: fmt.Sprintf("%s not found", resource),
		Err:     ErrNotFound,
	}
}

// Unauthorized returns a 401 error.
func Unauthorized(message string) *AppError {
	return &AppError{
		Code:    http.StatusUnauthorized,
		Message: message,
		Err:     ErrUnauthorized,
	}
}

// Forbidden returns a 403 error for a disallowed action.
func Forbidden(action string) *AppError {
	return &AppError{
		Code:    http.StatusForbidden,
		Message: fmt.Sprintf("forbidden: %s", action),
		Err:     ErrForbidden,
	}
}

// BadRequest returns a 400 error.
func BadRequest(message string) *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: message,
		Err:     ErrInvalidInput,
	}
}

// Conflict returns a 409 error.
func Conflict(message string) *AppError {
	return &AppError{
		Code:    http.StatusConflict,
		Message: message,
		Err:     ErrConflict,
	}
}

// InternalServer returns a 500 error wrapping the original cause.
func InternalServer(err error) *AppError {
	return &AppError{
		Code:    http.StatusInternalServerError,
		Message: "internal server error",
		Err:     err,
	}
}

// UnprocessableEntity returns a 422 error.
func UnprocessableEntity(message string) *AppError {
	return &AppError{
		Code:    http.StatusUnprocessableEntity,
		Message: message,
		Err:     ErrInvalidInput,
	}
}

// TooManyRequests returns a 429 error.
func TooManyRequests() *AppError {
	return &AppError{
		Code:    http.StatusTooManyRequests,
		Message: "too many requests, please slow down",
	}
}

// StorageQuotaExceeded returns a 413 error when the user's quota is full.
func StorageQuotaExceeded() *AppError {
	return &AppError{
		Code:    http.StatusRequestEntityTooLarge,
		Message: "storage quota exceeded",
		Err:     ErrStorageQuota,
	}
}

// FileTooBig returns a 413 error when an uploaded file exceeds the allowed size.
func FileTooBig(maxSize int64) *AppError {
	return &AppError{
		Code:    http.StatusRequestEntityTooLarge,
		Message: fmt.Sprintf("file exceeds maximum allowed size of %d bytes", maxSize),
		Err:     ErrFileTooBig,
	}
}

// Sentinel errors — use with errors.Is for targeted handling.
var (
	ErrNotFound         = errors.New("resource not found")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrForbidden        = errors.New("forbidden")
	ErrConflict         = errors.New("resource already exists")
	ErrInvalidInput     = errors.New("invalid input")
	ErrTokenExpired     = errors.New("token expired")
	ErrTokenInvalid     = errors.New("token invalid")
	ErrStorageQuota     = errors.New("storage quota exceeded")
	ErrFileTooBig       = errors.New("file too big")
	ErrInvalidFileType  = errors.New("invalid file type")
	ErrPermissionDenied = errors.New("permission denied")
	ErrShareLinkExpired = errors.New("share link expired or invalid")
)

// Is delegates to the standard library so callers need not import "errors" separately.
func Is(err, target error) bool { return errors.Is(err, target) }

// As delegates to the standard library so callers need not import "errors" separately.
func As(err error, target interface{}) bool { return errors.As(err, &target) }

// IsAppError checks whether err (or any error in its chain) is an *AppError and
// returns it along with a boolean indicating success.
func IsAppError(err error) (*AppError, bool) {
	var e *AppError
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}
