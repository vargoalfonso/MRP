// Package handler provides HTTP handlers for the Finished Goods module.
package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ganasa18/go-template/internal/base/app"
	fgModels "github.com/ganasa18/go-template/internal/finished_goods/models"
	"github.com/ganasa18/go-template/internal/finished_goods/repository"
	fgService "github.com/ganasa18/go-template/internal/finished_goods/service"
	userPkg "github.com/ganasa18/go-template/pkg/auth"
	"github.com/ganasa18/go-template/pkg/pagination"
	"github.com/ganasa18/go-template/pkg/validator"
)

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

type HTTPHandler struct {
	svc fgService.IService
}

func New(svc fgService.IService) *HTTPHandler { return &HTTPHandler{svc: svc} }

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

func buildFilter(ctx *app.Context) repository.FinishedGoodsFilter {
	p := pagination.FinishedGoodsPagination(ctx)
	return repository.FinishedGoodsFilter{
		Search:            p.Search,
		Model:             p.Model,
		Status:            p.Status,
		WarehouseLocation: p.WarehouseLocation,
		Page:              p.Page,
		Limit:             p.Limit,
		Offset:            p.Offset(),
	}
}

func buildStatusMonitoringFilter(ctx *app.Context) repository.StatusMonitoringFilter {
	p := pagination.StatusMonitoringPagination(ctx)
	return repository.StatusMonitoringFilter{
		AlertType: p.AlertType,
		Page:      p.Page,
		Limit:     p.Limit,
		Offset:    p.Offset(),
	}
}

// ---------------------------------------------------------------------------
// GET /api/v1/finished-goods/summary
// ---------------------------------------------------------------------------

func (h *HTTPHandler) GetSummary(ctx *app.Context) *app.CostumeResponse {
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

// ---------------------------------------------------------------------------
// GET /api/v1/finished-goods/status-monitoring
// ---------------------------------------------------------------------------

func (h *HTTPHandler) GetStatusMonitoring(ctx *app.Context) *app.CostumeResponse {
	f := buildStatusMonitoringFilter(ctx)
	data, err := h.svc.GetStatusMonitoring(ctx.Request.Context(), f)
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
// GET /api/v1/finished-goods
// ---------------------------------------------------------------------------

func (h *HTTPHandler) ListFinishedGoods(ctx *app.Context) *app.CostumeResponse {
	f := buildFilter(ctx)
	data, err := h.svc.ListFinishedGoods(ctx.Request.Context(), f)
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
// GET /api/v1/finished-goods/parameterized-summary?uniq_code=...
// ---------------------------------------------------------------------------

func (h *HTTPHandler) GetParameterizedSummary(ctx *app.Context) *app.CostumeResponse {
	uniqCode := strings.TrimSpace(ctx.Query("uniq_code"))
	if uniqCode == "" {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "uniq_code is required",
		}
	}

	data, err := h.svc.GetParameterizedSummary(ctx.Request.Context(), uniqCode)
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
// GET /api/v1/finished-goods/form-options/uniq
// ---------------------------------------------------------------------------

func (h *HTTPHandler) CreateFormUniqOptions(ctx *app.Context) *app.CostumeResponse {
	q := strings.TrimSpace(ctx.Query("q"))
	limit := 20
	if v := strings.TrimSpace(ctx.Query("limit")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return &app.CostumeResponse{
				RequestID: ctx.APIReqID,
				Status:    http.StatusBadRequest,
				Message:   "invalid limit",
			}
		}
		limit = n
	}

	data, err := h.svc.ListCreateUniqOptions(ctx.Request.Context(), q, limit)
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
// GET /api/v1/finished-goods/:id
// ---------------------------------------------------------------------------

func (h *HTTPHandler) GetFinishedGoodsByID(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}
	data, err := h.svc.GetFinishedGoodsByID(ctx.Request.Context(), id)
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
// POST /api/v1/finished-goods
// ---------------------------------------------------------------------------

func (h *HTTPHandler) CreateFinishedGoods(ctx *app.Context) *app.CostumeResponse {
	var req fgModels.CreateFinishedGoodsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   err.Error(),
		}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      errs,
		}
	}

	user := userPkg.MustExtractUserContext(ctx)
	data, err := h.svc.CreateFinishedGoods(ctx.Request.Context(), req, user.UserID)
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

// ---------------------------------------------------------------------------
// PUT /api/v1/finished-goods/:id
// ---------------------------------------------------------------------------

func (h *HTTPHandler) UpdateFinishedGoods(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}

	var req fgModels.UpdateFinishedGoodsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   err.Error(),
		}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      errs,
		}
	}

	user := userPkg.MustExtractUserContext(ctx)
	data, err := h.svc.UpdateFinishedGoods(ctx.Request.Context(), id, req, user.UserID)
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
