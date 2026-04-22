package handler

import (
	"net/http"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/supplier_performance/models"
	"github.com/ganasa18/go-template/internal/supplier_performance/service"
	userPkg "github.com/ganasa18/go-template/pkg/auth"
	"github.com/ganasa18/go-template/pkg/pagination"
	"github.com/ganasa18/go-template/pkg/validator"
)

type HTTPHandler struct {
	svc service.IService
}

func New(svc service.IService) *HTTPHandler { return &HTTPHandler{svc: svc} }

// List handles GET /api/v1/suppliers/performance
func (h *HTTPHandler) List(ctx *app.Context) *app.CostumeResponse {
	p := pagination.SupplierPerformancePagination(ctx)
	result, err := h.svc.List(ctx.Request.Context(), p)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      result,
	}
}

// Summary handles GET /api/v1/suppliers/performance/summary
func (h *HTTPHandler) Summary(ctx *app.Context) *app.CostumeResponse {
	periodType := ctx.Query("period_type")
	periodValue := ctx.Query("period_value")
	result, err := h.svc.Summary(ctx.Request.Context(), periodType, periodValue)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      result,
	}
}

// Charts handles GET /api/v1/suppliers/performance/charts
func (h *HTTPHandler) Charts(ctx *app.Context) *app.CostumeResponse {
	periodType := ctx.Query("period_type")
	periodValue := ctx.Query("period_value")
	result, err := h.svc.Charts(ctx.Request.Context(), periodType, periodValue)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      result,
	}
}

// Export handles GET /api/v1/suppliers/performance/export
func (h *HTTPHandler) Export(ctx *app.Context) *app.CostumeResponse {
	p := pagination.SupplierPerformancePagination(ctx)
	result, err := h.svc.Export(ctx.Request.Context(), p)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      result,
	}
}

// Override handles POST /api/v1/suppliers/performance/:supplier_id/override
func (h *HTTPHandler) Override(ctx *app.Context) *app.CostumeResponse {
	supplierUUID := ctx.Param("supplier_id")

	var req models.OverrideRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body",
		}
	}
	req.SupplierUUID = supplierUUID

	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": errs},
		}
	}

	userCtx := userPkg.MustExtractUserContext(ctx)
	actor := userCtx.UserID

	if err := h.svc.Override(ctx.Request.Context(), req, actor); err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
	}
}

// AuditLogs handles GET /api/v1/suppliers/performance/:supplier_id/audit-logs
func (h *HTTPHandler) AuditLogs(ctx *app.Context) *app.CostumeResponse {
	supplierUUID := ctx.Param("supplier_id")
	periodType := ctx.Query("period_type")
	periodValue := ctx.Query("period_value")

	logs, err := h.svc.AuditLogs(ctx.Request.Context(), supplierUUID, periodType, periodValue)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      map[string]interface{}{"items": logs},
	}
}
