package handler

import (
	"net/http"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/customer_order_logs/models"
	"github.com/ganasa18/go-template/internal/customer_order_logs/service"
	"github.com/ganasa18/go-template/pkg/validator"
)

type HTTPHandler struct {
	svc service.IService
}

func New(svc service.IService) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

// List returns customer-order automation failure logs, optionally filtered by
// document_number or a free-text search.
func (h *HTTPHandler) List(ctx *app.Context) *app.CostumeResponse {
	var q models.ListLogQuery
	if err := ctx.ShouldBindQuery(&q); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid query params",
		}
	}

	resp, err := h.svc.List(ctx.Request.Context(), q)
	if err != nil {
		return app.NewError(ctx, err)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      resp,
	}
}

// Create stores a single failed-automation log row.
func (h *HTTPHandler) Create(ctx *app.Context) *app.CostumeResponse {
	var req models.CreateLogRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body",
		}
	}

	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": errs},
		}
	}

	resp, err := h.svc.Create(ctx.Request.Context(), req)
	if err != nil {
		return app.NewError(ctx, err)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusCreated,
		Message:   http.StatusText(http.StatusCreated),
		Data:      resp,
	}
}
