package app

import (
	"log/slog"
	"net/http"

	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/logger"
)

// CostumeResponse is the single JSON envelope for every API response.
//
//	{"request_id":"...", "status":200, "message":"OK", "data":{...}}
type CostumeResponse struct {
	RequestID string      `json:"request_id"`
	Status    int         `json:"status"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
}

// NewSuccess builds a success CostumeResponse.
func NewSuccess(ctx *Context, statusCode int, data interface{}) *CostumeResponse {
	return &CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    statusCode,
		Message:   http.StatusText(statusCode),
		Data:      data,
	}
}

// NewError builds an error CostumeResponse.
// If err is an *apperror.AppError its HTTPStatus and Message are used;
// otherwise a generic 500 is returned — internal details are never leaked.
func NewError(ctx *Context, err error) *CostumeResponse {
	if appErr, ok := apperror.As(err); ok {
		return &CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    appErr.HTTPStatus,
			Message:   appErr.Message,
		}
	}
	// Log the real cause server-side (never sent to the client) so a bare
	// "an unexpected error occurred" can still be traced by request_id.
	logger.FromContext(ctx.Request.Context()).Error("unhandled error",
		slog.String("request_id", ctx.APIReqID),
		slog.String("path", ctx.Request.URL.Path),
		slog.Any("error", err),
	)
	return &CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusInternalServerError,
		Message:   "an unexpected error occurred",
	}
}
