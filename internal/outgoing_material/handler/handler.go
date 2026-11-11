package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	outModels "github.com/ganasa18/go-template/internal/outgoing_material/models"
	outService "github.com/ganasa18/go-template/internal/outgoing_material/service"
	userPkg "github.com/ganasa18/go-template/pkg/auth"
	"github.com/ganasa18/go-template/pkg/pagination"
	"github.com/ganasa18/go-template/pkg/validator"
)

type HTTPHandler struct {
	svc outService.IService
}

func New(svc outService.IService) *HTTPHandler { return &HTTPHandler{svc: svc} }

func parseID(ctx *app.Context) (int64, bool) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// GetFormOptions returns raw material options for the create outgoing form autocomplete.
// FE hits this when user types in the RM Code / uniq field to pre-fill rm_name, uom, and current stock.
//
//	GET /api/v1/outgoing-raw-materials/form-options?q=LV3&limit=20
func (h *HTTPHandler) GetFormOptions(ctx *app.Context) *app.CostumeResponse {
	q := ctx.Query("q")
	limit := 20
	if v := ctx.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	data, err := h.svc.SearchRawMaterials(ctx.Request.Context(), q, limit)
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

// ListOutgoingRM returns paginated outgoing RM transactions.
//
//	GET /api/v1/outgoing-raw-materials?date_from=2024-01-01&date_to=2024-12-31&reason=Production+Use&uniq=RM-PL-689795&limit=20&page=1
func (h *HTTPHandler) ListOutgoingRM(ctx *app.Context) *app.CostumeResponse {
	p := pagination.OutgoingRMPagination(ctx)
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

// GetOutgoingRMByID returns a single outgoing RM transaction detail.
//
//	GET /api/v1/outgoing-raw-materials/:id
func (h *HTTPHandler) GetOutgoingRMByID(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}
	data, err := h.svc.GetByID(ctx.Request.Context(), id)
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

// CreateOutgoingRM processes an outgoing RM transaction.
// Atomically deducts stock from raw_materials, records the transaction, and writes audit log.
//
//	POST /api/v1/outgoing-raw-materials
func (h *HTTPHandler) CreateOutgoingRM(ctx *app.Context) *app.CostumeResponse {
	var req outModels.CreateOutgoingRMRequest
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
		Message:   http.StatusText(http.StatusCreated),
		Data:      data,
	}
}

// UpdateOutgoingRM updates an existing outgoing RM transaction.
// When quantity_out (and/or uniq) changes, stock in raw_materials is
// re-calculated automatically inside a single DB transaction.
//
//	PUT /api/v1/outgoing-raw-materials/:id
func (h *HTTPHandler) UpdateOutgoingRM(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}
	var req outModels.UpdateOutgoingRMRequest
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
	data, err := h.svc.Update(ctx.Request.Context(), id, req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "Updated",
		Data:      data,
	}
}

// DeleteOutgoingRM soft-deletes an outgoing RM transaction.
// NOTE: stock is intentionally NOT restored here. Use POST /:id/restore-stock
// to return the quantity back to raw_materials.
//
//	DELETE /api/v1/outgoing-raw-materials/:id
func (h *HTTPHandler) DeleteOutgoingRM(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	if err := h.svc.Delete(ctx.Request.Context(), id, userCtx.UserID); err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "Deleted",
	}
}

// RestoreStock returns the transaction's quantity back into raw_materials stock.
// It is a manual, one-time action (guarded against double restore) meant to be
// used after a transaction was deleted or created by mistake.
//
//	POST /api/v1/outgoing-raw-materials/:id/restore-stock
func (h *HTTPHandler) RestoreStock(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	data, err := h.svc.RestoreStock(ctx.Request.Context(), id, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "Stock restored",
		Data:      data,
	}
}
