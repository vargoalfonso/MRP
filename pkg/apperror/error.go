// Package apperror defines the canonical error type used across all layers.
// Handlers map AppError to HTTP responses; services & repositories return AppError
// (or plain errors that handlers wrap).
package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

// Code is a machine-readable error identifier sent to clients.
type Code string

const (
	CodeBadRequest     Code = "BAD_REQUEST"
	CodeUnauthorized   Code = "UNAUTHORIZED"
	CodeForbidden      Code = "FORBIDDEN"
	CodeNotFound       Code = "NOT_FOUND"
	CodeConflict       Code = "CONFLICT"
	CodeUnprocessable  Code = "UNPROCESSABLE_ENTITY"
	CodeInternalError  Code = "INTERNAL_SERVER_ERROR"
	CodeServiceUnavail Code = "SERVICE_UNAVAILABLE"
	CodeTokenExpired   Code = "TOKEN_EXPIRED"
	CodeTokenInvalid   Code = "TOKEN_INVALID"
)

// AppError is the standard application error. It carries an HTTP status code,
// a machine-readable Code, a human-readable Message, and an optional Cause.
//
// The Cause MUST NOT be surfaced in API responses to avoid leaking internals.
type AppError struct {
	HTTPStatus int    `json:"-"`
	Code       Code   `json:"code"`
	Message    string `json:"message"`
	Cause      error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Cause }

// New creates a new AppError.
func New(httpStatus int, code Code, message string) *AppError {
	return &AppError{HTTPStatus: httpStatus, Code: code, Message: message}
}

// Wrap wraps a cause error inside an AppError.
func Wrap(httpStatus int, code Code, message string, cause error) *AppError {
	return &AppError{HTTPStatus: httpStatus, Code: code, Message: message, Cause: cause}
}

// As extracts an *AppError from err if one exists in the chain.
func As(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// --- Convenience constructors ---

func BadRequest(msg string) *AppError {
	return New(http.StatusBadRequest, CodeBadRequest, msg)
}

func Unauthorized(msg string) *AppError {
	return New(http.StatusUnauthorized, CodeUnauthorized, msg)
}

func Forbidden(msg string) *AppError {
	return New(http.StatusForbidden, CodeForbidden, msg)
}

func NotFound(msg string) *AppError {
	return New(http.StatusNotFound, CodeNotFound, msg)
}

func Conflict(msg string) *AppError {
	return New(http.StatusConflict, CodeConflict, msg)
}

func UnprocessableEntity(msg string) *AppError {
	return New(http.StatusUnprocessableEntity, CodeUnprocessable, msg)
}

func Internal(msg string) *AppError {
	return New(http.StatusInternalServerError, CodeInternalError, msg)
}

func InternalWrap(msg string, cause error) *AppError {
	return Wrap(http.StatusInternalServerError, CodeInternalError, msg, cause)
}

func TokenExpired() *AppError {
	return New(http.StatusUnauthorized, CodeTokenExpired, "token has expired")
}

func TokenInvalid() *AppError {
	return New(http.StatusUnauthorized, CodeTokenInvalid, "token is invalid")
}
