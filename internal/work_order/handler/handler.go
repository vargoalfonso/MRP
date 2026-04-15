package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ganasa18/go-template/internal/base/app"
	woModels "github.com/ganasa18/go-template/internal/work_order/models"
	woService "github.com/ganasa18/go-template/internal/work_order/service"
	userPkg "github.com/ganasa18/go-template/pkg/auth"
	"github.com/ganasa18/go-template/pkg/pagination"
	"github.com/ganasa18/go-template/pkg/validator"
)

type HTTPHandler struct {
	svc woService.IService
}

func New(svc woService.IService) *HTTPHandler { return &HTTPHandler{svc: svc} }

// ListWorkOrders returns paginated work order list.
// GET /api/v1/working-order/work-orders
func (h *HTTPHandler) ListWorkOrders(ctx *app.Context) *app.CostumeResponse {
	p := pagination.WorkOrderPagination(ctx)
	data, err := h.svc.List(ctx.Request.Context(), p)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// CreateWorkOrder creates a work order header and items (kanbans).
// POST /api/v1/working-order/work-orders
func (h *HTTPHandler) CreateWorkOrder(ctx *app.Context) *app.CostumeResponse {
	var req woModels.CreateWorkOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
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

	userCtx := userPkg.MustExtractUserContext(ctx)
	data, err := h.svc.Create(ctx.Request.Context(), req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "Created",
		Data:      data,
	}
}

// PreviewWorkOrder returns computed wo_number + kanban lines without inserting.
// POST /api/v1/working-order/work-orders/preview
func (h *HTTPHandler) PreviewWorkOrder(ctx *app.Context) *app.CostumeResponse {
	var req woModels.CreateWorkOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
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

	data, err := h.svc.Preview(ctx.Request.Context(), req)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// GetWorkOrderSummary returns board summary counters.
// GET /api/v1/working-order/work-orders/summary
func (h *HTTPHandler) GetWorkOrderSummary(ctx *app.Context) *app.CostumeResponse {
	data, err := h.svc.GetSummary(ctx.Request.Context())
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// GetWorkOrderDetail returns WO header + items.
// GET /api/v1/working-order/work-orders/:id
func (h *HTTPHandler) GetWorkOrderDetail(ctx *app.Context) *app.CostumeResponse {
	woUUID := ctx.Param("id")
	data, err := h.svc.GetDetail(ctx.Request.Context(), woUUID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// Approval approves or rejects a work order.
// POST /api/v1/working-order/work-orders/:id/approval
func (h *HTTPHandler) Approval(ctx *app.Context) *app.CostumeResponse {
	woUUID := ctx.Param("id")

	var req woModels.WorkOrderApprovalRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
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

	userCtx := userPkg.MustExtractUserContext(ctx)
	data, err := h.svc.Approval(ctx.Request.Context(), woUUID, req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// BulkApproval approves or rejects multiple work orders by wo_number.
// POST /api/v1/working-order/work-orders/bulk-approval
func (h *HTTPHandler) BulkApproval(ctx *app.Context) *app.CostumeResponse {
	var req woModels.BulkWorkOrderApprovalRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
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

	userCtx := userPkg.MustExtractUserContext(ctx)
	data, err := h.svc.BulkApproval(ctx.Request.Context(), req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// GetWorkOrderQR returns (and caches) QR base64 for WO header.
// GET /api/v1/working-order/work-orders/:id/qr
func (h *HTTPHandler) GetWorkOrderQR(ctx *app.Context) *app.CostumeResponse {
	woUUID := ctx.Param("id")
	refresh := strings.EqualFold(ctx.Query("refresh"), "1") || strings.EqualFold(ctx.Query("refresh"), "true")

	data, err := h.svc.GetWorkOrderQR(ctx.Request.Context(), woUUID, refresh)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// GetWorkOrderItemQR returns (and caches) QR base64 for WO item/kanban.
// GET /api/v1/working-order/work-order-items/:id/qr
func (h *HTTPHandler) GetWorkOrderItemQR(ctx *app.Context) *app.CostumeResponse {
	itemUUID := ctx.Param("id")
	refresh := strings.EqualFold(ctx.Query("refresh"), "1") || strings.EqualFold(ctx.Query("refresh"), "true")

	data, err := h.svc.GetWorkOrderItemQR(ctx.Request.Context(), itemUUID, refresh)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// UniqFormOptions returns union uniq_code options across items + inventory tables.
// GET /api/v1/working-order/work-orders/form-options/uniq?q=...&limit=20&sources=items,raw_material,indirect,subcon
func (h *HTTPHandler) UniqFormOptions(ctx *app.Context) *app.CostumeResponse {
	q := ctx.Query("q")
	limit := 20
	if v := strings.TrimSpace(ctx.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	var sources []string
	if s := strings.TrimSpace(ctx.Query("sources")); s != "" {
		sources = strings.Split(s, ",")
	}

	data, err := h.svc.ListUniqOptions(ctx.Request.Context(), q, limit, sources)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// ProcessFormOptions returns active process options for Work Order create form.
// GET /api/v1/working-order/work-orders/form-options/processes
func (h *HTTPHandler) ProcessFormOptions(ctx *app.Context) *app.CostumeResponse {
	data, err := h.svc.ListProcessOptions(ctx.Request.Context())
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}
