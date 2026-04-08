// Package response provides a single, consistent JSON envelope for all API responses.
//
// Success shape:
//
//	{"success":true,  "message":"...", "data":{...},  "request_id":"uuid"}
//
// Error shape:
//
//	{"success":false, "message":"...", "error":{"code":"...", "details":...}, "request_id":"uuid"}
package response

import (
	"net/http"

	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/logger"
	"github.com/gin-gonic/gin"
)

// envelope is the top-level JSON wrapper for every response.
type envelope struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Error     *errDetail  `json:"error,omitempty"`
	RequestID string      `json:"request_id"`
}

type errDetail struct {
	Code    apperror.Code `json:"code"`
	Details interface{}   `json:"details,omitempty"`
}

// requestID extracts the request ID stored by the request-id middleware.
func requestID(c *gin.Context) string {
	return logger.RequestIDFromContext(c.Request.Context())
}

// OK writes a 200 success response.
func OK(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, envelope{
		Success:   true,
		Message:   message,
		Data:      data,
		RequestID: requestID(c),
	})
}

// Created writes a 201 success response.
func Created(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusCreated, envelope{
		Success:   true,
		Message:   message,
		Data:      data,
		RequestID: requestID(c),
	})
}

// Error writes an error response. It checks if err is an *apperror.AppError and
// uses its HTTP status + code. Unknown errors become 500 Internal Server Error.
// The internal Cause is never exposed to the client.
func Error(c *gin.Context, err error) {
	if appErr, ok := apperror.As(err); ok {
		c.JSON(appErr.HTTPStatus, envelope{
			Success:   false,
			Message:   appErr.Message,
			Error:     &errDetail{Code: appErr.Code},
			RequestID: requestID(c),
		})
		return
	}
	// Unknown error — do not leak details.
	c.JSON(http.StatusInternalServerError, envelope{
		Success:   false,
		Message:   "an unexpected error occurred",
		Error:     &errDetail{Code: apperror.CodeInternalError},
		RequestID: requestID(c),
	})
}

// ValidationError writes a 422 response with field-level validation details.
func ValidationError(c *gin.Context, details interface{}) {
	c.JSON(http.StatusUnprocessableEntity, envelope{
		Success:   false,
		Message:   "validation failed",
		Error:     &errDetail{Code: apperror.CodeUnprocessable, Details: details},
		RequestID: requestID(c),
	})
}

// WrapSuccess builds the success envelope map used by RunAction.
// Exposed so RunAction can call c.JSON directly with a typed status code.
func WrapSuccess(httpCode int, data interface{}, requestID string) envelope {
	return envelope{
		Success:   true,
		Message:   "ok",
		Data:      data,
		RequestID: requestID,
	}
}

// Abort is like Error but also aborts the gin handler chain.
func Abort(c *gin.Context, err error) {
	if appErr, ok := apperror.As(err); ok {
		c.AbortWithStatusJSON(appErr.HTTPStatus, envelope{
			Success:   false,
			Message:   appErr.Message,
			Error:     &errDetail{Code: appErr.Code},
			RequestID: requestID(c),
		})
		return
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, envelope{
		Success:   false,
		Message:   "an unexpected error occurred",
		Error:     &errDetail{Code: apperror.CodeInternalError},
		RequestID: requestID(c),
	})
}
