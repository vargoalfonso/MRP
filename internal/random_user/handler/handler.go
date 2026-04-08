package handler

import (
	"net/http"

	"github.com/ganasa18/go-template/internal/base/app"
	randomUserService "github.com/ganasa18/go-template/internal/random_user/service"
)

// HTTPHandler exposes random_user endpoints.
type HTTPHandler struct {
	RandomUserService randomUserService.Service
}

// NewHTTPHandler constructs an HTTPHandler.
func NewHTTPHandler(svc randomUserService.Service) *HTTPHandler {
	return &HTTPHandler{RandomUserService: svc}
}

func (h *HTTPHandler) GetRandomDataUser(appCtx *app.Context) *app.CostumeResponse {
	resp, statusCode, err := h.RandomUserService.GetRandomDataUser(appCtx.Request.Context())
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    statusCode,
		Message:   http.StatusText(statusCode),
		Data:      resp,
	}
}
