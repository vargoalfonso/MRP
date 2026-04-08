package app

import (
	"net/http"

	"github.com/ganasa18/go-template/pkg/apperror"
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
	return &CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusInternalServerError,
		Message:   "an unexpected error occurred",
	}
}
