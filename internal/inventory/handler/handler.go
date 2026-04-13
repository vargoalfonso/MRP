package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	invModels "github.com/ganasa18/go-template/internal/inventory/models"
	invService "github.com/ganasa18/go-template/internal/inventory/service"
	userPkg "github.com/ganasa18/go-template/pkg/auth"
	"github.com/ganasa18/go-template/pkg/pagination"
	"github.com/ganasa18/go-template/pkg/validator"
)

type HTTPHandler struct {
	svc invService.IService
}

func New(svc invService.IService) *HTTPHandler { return &HTTPHandler{svc: svc} }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func parseID(ctx *app.Context) (int64, bool) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// ---------------------------------------------------------------------------
// Raw Material — GET /api/v1/inventory/raw-materials
// ---------------------------------------------------------------------------

// ListRawMaterials returns the RM database list with stats cards.
//
//	GET /api/v1/inventory/raw-materials?search=LV7&rm_type=sheet_plate&status=low_on_stock&buy_not_buy=buy&limit=20&page=1
func (h *HTTPHandler) ListRawMaterials(ctx *app.Context) *app.CostumeResponse {
	p := pagination.InventoryRMPagination(ctx)
	data, err := h.svc.ListRawMaterials(ctx.Request.Context(), p)
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

// CreateRawMaterial creates a new RM record.
//
//	POST /api/v1/inventory/raw-materials
func (h *HTTPHandler) CreateRawMaterial(ctx *app.Context) *app.CostumeResponse {
	var req invModels.CreateRawMaterialRequest
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
	data, err := h.svc.CreateRawMaterial(ctx.Request.Context(), req, userCtx.UserID)
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

// BulkCreateRawMaterials creates multiple RM records in one request.
//
//	POST /api/v1/inventory/raw-materials/bulk
func (h *HTTPHandler) BulkCreateRawMaterials(ctx *app.Context) *app.CostumeResponse {
	var req invModels.BulkCreateRawMaterialRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}
	if len(req.Items) == 0 {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "items cannot be empty",
		}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	count, err := h.svc.BulkCreateRawMaterials(ctx.Request.Context(), req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "Created",
		Data:      map[string]interface{}{"created": count},
	}
}

// GetRawMaterialByID returns a single RM record.
//
//	GET /api/v1/inventory/raw-materials/:id
func (h *HTTPHandler) GetRawMaterialByID(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	data, err := h.svc.GetRawMaterialByID(ctx.Request.Context(), id)
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

// UpdateRawMaterial updates a RM record (partial update).
//
//	PUT /api/v1/inventory/raw-materials/:id
func (h *HTTPHandler) UpdateRawMaterial(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	var req invModels.UpdateRawMaterialRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	data, err := h.svc.UpdateRawMaterial(ctx.Request.Context(), id, req, userCtx.UserID)
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

// DeleteRawMaterial soft-deletes a RM record.
//
//	DELETE /api/v1/inventory/raw-materials/:id
func (h *HTTPHandler) DeleteRawMaterial(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	if err := h.svc.DeleteRawMaterial(ctx.Request.Context(), id, userCtx.UserID); err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "deleted",
	}
}

// GetRawMaterialHistory returns movement log for a single RM record.
//
//	GET /api/v1/inventory/raw-materials/:id/history?limit=20&page=1
func (h *HTTPHandler) GetRawMaterialHistory(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	p := pagination.Pagination(ctx)
	data, err := h.svc.GetRawMaterialHistory(ctx.Request.Context(), id, p)
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

// ListIncomingRM returns incoming scan list filtered by RM type.
//
//	GET /api/v1/inventory/raw-materials/incoming?status=pending&po_number=PO-2026-001&limit=20&page=1
func (h *HTTPHandler) ListIncomingRM(ctx *app.Context) *app.CostumeResponse {
	p := pagination.InventoryIncomingPagination(ctx)
	data, err := h.svc.ListIncoming(ctx.Request.Context(), "RM", p)
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

// ---------------------------------------------------------------------------
// Indirect Raw Material — GET /api/v1/inventory/indirect-materials
// ---------------------------------------------------------------------------

// ListIndirectMaterials returns the indirect RM list with stats.
//
//	GET /api/v1/inventory/indirect-materials?search=NBR&status=normal&limit=20&page=1
func (h *HTTPHandler) ListIndirectMaterials(ctx *app.Context) *app.CostumeResponse {
	p := pagination.InventoryIndirectPagination(ctx)
	data, err := h.svc.ListIndirectMaterials(ctx.Request.Context(), p)
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

// CreateIndirectMaterial creates a new indirect RM record.
//
//	POST /api/v1/inventory/indirect-materials
func (h *HTTPHandler) CreateIndirectMaterial(ctx *app.Context) *app.CostumeResponse {
	var req invModels.CreateIndirectMaterialRequest
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
	data, err := h.svc.CreateIndirectMaterial(ctx.Request.Context(), req, userCtx.UserID)
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

// BulkCreateIndirectMaterials creates multiple indirect RM records.
//
//	POST /api/v1/inventory/indirect-materials/bulk
func (h *HTTPHandler) BulkCreateIndirectMaterials(ctx *app.Context) *app.CostumeResponse {
	var req invModels.BulkCreateIndirectMaterialRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}
	if len(req.Items) == 0 {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "items cannot be empty",
		}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	count, err := h.svc.BulkCreateIndirectMaterials(ctx.Request.Context(), req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "Created",
		Data:      map[string]interface{}{"created": count},
	}
}

// GetIndirectByID returns a single indirect RM record.
//
//	GET /api/v1/inventory/indirect-materials/:id
func (h *HTTPHandler) GetIndirectByID(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	data, err := h.svc.GetIndirectByID(ctx.Request.Context(), id)
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

// UpdateIndirectMaterial updates an indirect RM record.
//
//	PUT /api/v1/inventory/indirect-materials/:id
func (h *HTTPHandler) UpdateIndirectMaterial(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	var req invModels.UpdateIndirectMaterialRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	data, err := h.svc.UpdateIndirectMaterial(ctx.Request.Context(), id, req, userCtx.UserID)
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

// DeleteIndirectMaterial soft-deletes an indirect RM record.
//
//	DELETE /api/v1/inventory/indirect-materials/:id
func (h *HTTPHandler) DeleteIndirectMaterial(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	if err := h.svc.DeleteIndirectMaterial(ctx.Request.Context(), id, userCtx.UserID); err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "deleted",
	}
}

// GetIndirectHistory returns movement log for a single indirect RM record.
//
//	GET /api/v1/inventory/indirect-materials/:id/history?limit=20&page=1
func (h *HTTPHandler) GetIndirectHistory(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	p := pagination.Pagination(ctx)
	data, err := h.svc.GetIndirectHistory(ctx.Request.Context(), id, p)
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

// ListIncomingIndirect returns incoming scan list filtered by INDIRECT type.
//
//	GET /api/v1/inventory/indirect-materials/incoming?status=pending&limit=20&page=1
func (h *HTTPHandler) ListIncomingIndirect(ctx *app.Context) *app.CostumeResponse {
	p := pagination.InventoryIncomingPagination(ctx)
	data, err := h.svc.ListIncoming(ctx.Request.Context(), "INDIRECT", p)
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

// ---------------------------------------------------------------------------
// Subcon Inventory — GET /api/v1/inventory/subcon
// ---------------------------------------------------------------------------

// ListSubconInventory returns the stock in vendor list.
//
//	GET /api/v1/inventory/subcon?search=EMA&po_number=PO-2026-001&period=2026-04&limit=20&page=1
func (h *HTTPHandler) ListSubconInventory(ctx *app.Context) *app.CostumeResponse {
	p := pagination.InventorySubconPagination(ctx)
	data, err := h.svc.ListSubconInventory(ctx.Request.Context(), p)
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

// CreateSubconInventory creates a new subcon inventory record.
//
//	POST /api/v1/inventory/subcon
func (h *HTTPHandler) CreateSubconInventory(ctx *app.Context) *app.CostumeResponse {
	var req invModels.CreateSubconInventoryRequest
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
	data, err := h.svc.CreateSubconInventory(ctx.Request.Context(), req, userCtx.UserID)
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

// GetSubconByID returns a single subcon inventory record.
//
//	GET /api/v1/inventory/subcon/:id
func (h *HTTPHandler) GetSubconByID(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	data, err := h.svc.GetSubconByID(ctx.Request.Context(), id)
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

// UpdateSubconInventory updates a subcon inventory record.
//
//	PUT /api/v1/inventory/subcon/:id
func (h *HTTPHandler) UpdateSubconInventory(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	var req invModels.UpdateSubconInventoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	data, err := h.svc.UpdateSubconInventory(ctx.Request.Context(), id, req, userCtx.UserID)
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

// DeleteSubconInventory soft-deletes a subcon inventory record.
//
//	DELETE /api/v1/inventory/subcon/:id
func (h *HTTPHandler) DeleteSubconInventory(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	if err := h.svc.DeleteSubconInventory(ctx.Request.Context(), id, userCtx.UserID); err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "deleted",
	}
}

// GetSubconHistory returns movement log for a subcon inventory record.
//
//	GET /api/v1/inventory/subcon/:id/history?limit=20&page=1
func (h *HTTPHandler) GetSubconHistory(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	p := pagination.Pagination(ctx)
	data, err := h.svc.GetSubconHistory(ctx.Request.Context(), id, p)
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

// ListSubconReceived returns stock received from vendor (movement type: incoming on subcon).
//
//	GET /api/v1/inventory/subcon/received?po_number=PO-2026-001&limit=20&page=1
func (h *HTTPHandler) ListSubconReceived(ctx *app.Context) *app.CostumeResponse {
	p := pagination.InventoryIncomingPagination(ctx)
	data, err := h.svc.ListIncoming(ctx.Request.Context(), "SUBCON", p)
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

// GetKanbanSummary returns kanban totals and incomplete stock for a given item_uniq_code.
// Frontend calls this asynchronously per row in the DN list to show kanban progress.
//
//	GET /api/v1/inventory/kanban-summary?uniq_code=EMA-LV7-001
func (h *HTTPHandler) GetKanbanSummary(ctx *app.Context) *app.CostumeResponse {
	uniqCode := ctx.Query("uniq_code")
	if uniqCode == "" {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "uniq_code is required",
		}
	}
	data, err := h.svc.GetKanbanSummary(ctx.Request.Context(), uniqCode)
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
