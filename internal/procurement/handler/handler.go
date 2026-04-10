// Package handler implements HTTP handlers for the Procurement module.
package handler

import (
	"net/http"
	"strconv"
	"strings"

	authModels "github.com/ganasa18/go-template/internal/auth/models"
	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/procurement/models"
	"github.com/ganasa18/go-template/internal/procurement/service"
	"github.com/ganasa18/go-template/pkg/pagination"
)

// HTTPHandler wires HTTP requests to service methods.
type HTTPHandler struct {
	svc service.IService
}

// New creates a new HTTPHandler.
func New(svc service.IService) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

// ---------------------------------------------------------------------------
// A) Summary KPI cards
// GET /api/v1/procurement/purchase-orders:summary
// ---------------------------------------------------------------------------

func (h *HTTPHandler) GetSummary(ctx *app.Context) *app.CostumeResponse {
	poType := ctx.Query("po_type")
	period := ctx.Query("period")

	result, err := h.svc.GetSummary(ctx.Request.Context(), poType, period)
	if err != nil {
		return app.NewError(ctx, err)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

// ---------------------------------------------------------------------------
// B) PO Board list
// GET /api/v1/procurement/purchase-orders
// ---------------------------------------------------------------------------

func (h *HTTPHandler) ListPOBoard(ctx *app.Context) *app.CostumeResponse {
	p := pagination.POBoardPagination(ctx)

	result, err := h.svc.ListPOBoard(ctx.Request.Context(), p)
	if err != nil {
		return app.NewError(ctx, err)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

// ---------------------------------------------------------------------------
// C / H) PO Detail
// GET /api/v1/procurement/purchase-orders/:po_id
// ---------------------------------------------------------------------------

func (h *HTTPHandler) GetPODetail(ctx *app.Context) *app.CostumeResponse {
	poID, err := strconv.ParseInt(ctx.Param("po_id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid po_id; must be a numeric integer",
		}
	}

	result, err := h.svc.GetPODetail(ctx.Request.Context(), poID)
	if err != nil {
		return app.NewError(ctx, err)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

// ---------------------------------------------------------------------------
// D) DN list (filter by po_number)
// GET /api/v1/procurement/incoming-dns
// ---------------------------------------------------------------------------

func (h *HTTPHandler) ListDNs(ctx *app.Context) *app.CostumeResponse {
	limit := 20
	page := 1
	if v := ctx.Query("limit"); v != "" {
		if n, e := strconv.Atoi(v); e == nil && n > 0 {
			limit = n
		}
	}
	if v := ctx.Query("page"); v != "" {
		if n, e := strconv.Atoi(v); e == nil && n > 0 {
			page = n
		}
	}

	f := models.DNListFilter{
		PoNumber: ctx.Query("po_number"),
		Period:   ctx.Query("period"),
		Status:   ctx.Query("status"),
		Page:     page,
		Limit:    limit,
	}

	result, err := h.svc.ListDNs(ctx.Request.Context(), f)
	if err != nil {
		return app.NewError(ctx, err)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

// ---------------------------------------------------------------------------
// E) DN detail
// GET /api/v1/procurement/incoming-dns/:dn_id
// ---------------------------------------------------------------------------

func (h *HTTPHandler) GetDNDetail(ctx *app.Context) *app.CostumeResponse {
	dnID := ctx.Param("dn_id")
	if dnID == "" {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "dn_id is required",
		}
	}

	result, err := h.svc.GetDNDetail(ctx.Request.Context(), dnID)
	if err != nil {
		return app.NewError(ctx, err)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

// ---------------------------------------------------------------------------
// F) Form options (wizard dropdown data)
// GET /api/v1/procurement/purchase-orders:form_options
// ---------------------------------------------------------------------------

func (h *HTTPHandler) GetFormOptions(ctx *app.Context) *app.CostumeResponse {
	poType := ctx.Query("po_type")
	period := ctx.Query("period")

	if poType == "" || period == "" {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "po_type and period are required query params",
		}
	}

	result, err := h.svc.GetFormOptions(ctx.Request.Context(), poType, period)
	if err != nil {
		return app.NewError(ctx, err)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

// ---------------------------------------------------------------------------
// G) Generate PO
// POST /api/v1/procurement/purchase-orders:generate
// ---------------------------------------------------------------------------

func (h *HTTPHandler) GeneratePO(ctx *app.Context) *app.CostumeResponse {
	var req models.GeneratePORequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}

	// Validate generate_mode
	switch req.GenerateMode {
	case "", "both_stages", "stage_only", "bulk_all_suppliers":
		// OK
	default:
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "generate_mode must be one of: both_stages, stage_only, bulk_all_suppliers",
		}
	}
	if req.GenerateMode == "" {
		req.GenerateMode = "stage_only"
		if req.Stage == 0 {
			req.Stage = 1
		}
	}

	// Validate line_strategy
	switch req.LineStrategy {
	case "", "keep_granular", "aggregate_by_uniq":
		// OK
	default:
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "line_strategy must be one of: keep_granular, aggregate_by_uniq",
		}
	}

	// Extract caller identity from JWT
	createdBy := mustUsername(ctx)

	result, err := h.svc.GeneratePO(ctx.Request.Context(), req, createdBy)
	if err != nil {
		// Only business-logic errors (type mismatch, no entries found) → 422.
		// Everything else (DB error, etc.) → use app.NewError which returns 500.
		msg := err.Error()
		if isBusinessError(msg) {
			return &app.CostumeResponse{
				RequestID: ctx.APIReqID,
				Status:    http.StatusUnprocessableEntity,
				Message:   msg,
			}
		}
		return app.NewError(ctx, err)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "Created",
		Data:      result,
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isBusinessError returns true for known domain/validation errors that should
// return 422, as opposed to infrastructure errors (DB, network) that are 500.
func isBusinessError(msg string) bool {
	return strings.Contains(msg, "type mismatch") ||
		strings.Contains(msg, "no approved budget entries found") ||
		strings.Contains(msg, "po_split_settings") ||
		strings.Contains(msg, "min_order_qty") ||
		strings.Contains(msg, "max_split_lines") ||
		strings.Contains(msg, "generate_mode") ||
		strings.Contains(msg, "stage=1 or stage=2")
}

// mustUsername extracts the userID from JWT claims.
// Falls back to "system" if claims not present (should not happen with auth middleware).
func mustUsername(ctx *app.Context) string {
	raw, exists := ctx.Get("claims")
	if !exists {
		return "system"
	}
	claims, ok := raw.(*authModels.Claims)
	if !ok || claims == nil {
		return "system"
	}
	if claims.UserID != "" {
		return claims.UserID
	}
	return "system"
}
