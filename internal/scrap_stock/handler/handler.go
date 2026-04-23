// Package handler provides HTTP handlers for the Scrap Stock module.
package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	scrapModels "github.com/ganasa18/go-template/internal/scrap_stock/models"
	"github.com/ganasa18/go-template/internal/scrap_stock/repository"
	scrapService "github.com/ganasa18/go-template/internal/scrap_stock/service"
	userPkg "github.com/ganasa18/go-template/pkg/auth"
	"github.com/ganasa18/go-template/pkg/pagination"
	"github.com/ganasa18/go-template/pkg/validator"
)

type HTTPHandler struct {
	svc scrapService.IService
}

func New(svc scrapService.IService) *HTTPHandler { return &HTTPHandler{svc: svc} }

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

func buildStockFilter(p pagination.ScrapStockPaginationInput) repository.ScrapStockFilter {
	return repository.ScrapStockFilter{
		ScrapType:     p.ScrapType,
		UniqCode:      p.UniqCode,
		PackingNumber: p.PackingNumber,
		WONumber:      p.WONumber,
		Status:        p.Status,
		DateFrom:      p.DateFrom,
		DateTo:        p.DateTo,
		Page:          p.Page,
		Limit:         p.Limit,
		Offset:        p.Offset(),
	}
}

func buildReleaseFilter(p pagination.ScrapReleasePaginationInput) repository.ScrapReleaseFilter {
	return repository.ScrapReleaseFilter{
		ReleaseType:    p.ReleaseType,
		ApprovalStatus: p.ApprovalStatus,
		ScrapStockID:   p.ScrapStockID,
		Page:           p.Page,
		Limit:          p.Limit,
		Offset:         p.Offset(),
	}
}

// ---------------------------------------------------------------------------
// Scrap Stock — /api/v1/scrap-stocks
// ---------------------------------------------------------------------------

// GetStats returns the 4 summary cards for the Scrap Stock dashboard.
//
//	GET /api/v1/scrap-stocks/stats
func (h *HTTPHandler) GetStats(ctx *app.Context) *app.CostumeResponse {
	data, err := h.svc.GetStats(ctx.Request.Context())
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

// ListScrapStocks returns a paginated list of scrap stock records.
//
//	GET /api/v1/scrap-stocks?scrap_type=process_scrap&uniq=EMA-LV7&page=1&limit=20
func (h *HTTPHandler) ListScrapStocks(ctx *app.Context) *app.CostumeResponse {
	f := buildStockFilter(pagination.ScrapStockPagination(ctx))
	data, err := h.svc.ListScrapStocks(ctx.Request.Context(), f)
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

// GetScrapStockByID returns the detail of a single scrap stock record.
//
//	GET /api/v1/scrap-stocks/:id
func (h *HTTPHandler) GetScrapStockByID(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}
	data, err := h.svc.GetScrapStockByID(ctx.Request.Context(), id)
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

// CreateScrapStock creates a new scrap stock record (manual entry).
//
//	POST /api/v1/scrap-stocks
func (h *HTTPHandler) CreateScrapStock(ctx *app.Context) *app.CostumeResponse {
	var req scrapModels.CreateScrapStockRequest
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
	data, err := h.svc.CreateScrapStock(ctx.Request.Context(), req, userCtx.UserID)
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

// ---------------------------------------------------------------------------
// Incoming Scrap — /api/v1/action-ui/scrap/incoming
// ---------------------------------------------------------------------------

// CreateIncomingScrap records a new scrap from the Action UI scan flow.
//
//	POST /api/v1/action-ui/scrap/incoming
func (h *HTTPHandler) CreateIncomingScrap(ctx *app.Context) *app.CostumeResponse {
	var req scrapModels.IncomingScrapRequest
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
	data, err := h.svc.CreateIncomingScrap(ctx.Request.Context(), req, userCtx.UserID)
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

// ListIncomingScrap lists incoming scrap records (Action UI view).
//
//	GET /api/v1/action-ui/scrap/incoming
func (h *HTTPHandler) ListIncomingScrap(ctx *app.Context) *app.CostumeResponse {
	f := buildStockFilter(pagination.ScrapStockPagination(ctx))
	data, err := h.svc.ListScrapStocks(ctx.Request.Context(), f)
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
// Scrap Release — /api/v1/scrap-releases
// ---------------------------------------------------------------------------

// ListScrapReleases returns a paginated list of scrap release records.
//
//	GET /api/v1/scrap-releases?release_type=Sell&approval_status=Pending&page=1&limit=20
func (h *HTTPHandler) ListScrapReleases(ctx *app.Context) *app.CostumeResponse {
	f := buildReleaseFilter(pagination.ScrapReleasePagination(ctx))
	data, err := h.svc.ListScrapReleases(ctx.Request.Context(), f)
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

// GetScrapReleaseByID returns the detail of a single scrap release record.
//
//	GET /api/v1/inventory/scrap-releases/:id
func (h *HTTPHandler) GetScrapReleaseByID(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}
	data, err := h.svc.GetScrapReleaseByID(ctx.Request.Context(), id)
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

// ListScrapMovements returns the in/out history log for a scrap stock record.
//
//	GET /api/v1/scrap-stocks/:id/history-logs?page=1&limit=20
func (h *HTTPHandler) ListScrapMovements(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}
	p := pagination.Pagination(ctx)
	data, err := h.svc.ListScrapMovements(ctx.Request.Context(), id, p.Page, p.Limit)
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

// CreateScrapRelease creates a new release request (Sell or Dump). Starts as Pending.
//
//	POST /api/v1/scrap-releases
func (h *HTTPHandler) CreateScrapRelease(ctx *app.Context) *app.CostumeResponse {
	var req scrapModels.CreateScrapReleaseRequest
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
	data, err := h.svc.CreateScrapRelease(ctx.Request.Context(), req, userCtx.UserID)
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

// ApproveScrapRelease approves or rejects a scrap release.
// When approved, release_qty is atomically deducted from scrap stock.
//
//	PUT /api/v1/scrap-releases/:id/approve
func (h *HTTPHandler) ApproveScrapRelease(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}
	var req scrapModels.ApproveScrapReleaseRequest
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
	if err := h.svc.ApproveScrapRelease(ctx.Request.Context(), id, req, userCtx.UserID); err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
	}
}
