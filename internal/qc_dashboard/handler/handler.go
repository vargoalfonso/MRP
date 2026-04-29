package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	qcDashboardModels "github.com/ganasa18/go-template/internal/qc_dashboard/models"
	qcDashboardRepo "github.com/ganasa18/go-template/internal/qc_dashboard/repository"
	qcDashboardService "github.com/ganasa18/go-template/internal/qc_dashboard/service"
	"github.com/ganasa18/go-template/pkg/apperror"
	userPkg "github.com/ganasa18/go-template/pkg/auth"
	"github.com/ganasa18/go-template/pkg/pagination"
)

type HTTPHandler struct{ svc qcDashboardService.IService }

func New(svc qcDashboardService.IService) *HTTPHandler { return &HTTPHandler{svc: svc} }

func (h *HTTPHandler) GetOverview(ctx *app.Context) *app.CostumeResponse {
	windowHours, err := positiveIntQuery(ctx.Query("window_hours"), 168)
	if err != nil {
		return app.NewError(ctx, err)
	}
	data, err := h.svc.GetOverview(ctx.Request.Context(), qcDashboardRepo.Filter{
		WindowHours: windowHours,
		DateFrom:    ctx.Query("date_from"),
		DateTo:      ctx.Query("date_to"),
	})
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, data)
}

func (h *HTTPHandler) ListProductionQC(ctx *app.Context) *app.CostumeResponse {
	p := pagination.QCDashboardPagination(ctx)
	data, err := h.svc.ListProductionQC(ctx.Request.Context(), qcDashboardRepo.Filter{
		Limit:    p.Limit,
		Page:     p.Page,
		Offset:   p.Offset(),
		Search:   p.Search,
		DateFrom: p.DateFrom,
		DateTo:   p.DateTo,
		Status:   p.Status,
		WONumber: p.WONumber,
		UniqCode: p.UniqCode,
	})
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, data)
}

func (h *HTTPHandler) ListIncomingQC(ctx *app.Context) *app.CostumeResponse {
	p := pagination.QCDashboardPagination(ctx)
	data, err := h.svc.ListIncomingQC(ctx.Request.Context(), qcDashboardRepo.Filter{
		Limit:      p.Limit,
		Page:       p.Page,
		Offset:     p.Offset(),
		Search:     p.Search,
		DateFrom:   p.DateFrom,
		DateTo:     p.DateTo,
		Status:     p.Status,
		SupplierID: p.SupplierID,
		PONumber:   p.PONumber,
		UniqCode:   p.UniqCode,
	})
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, data)
}

func (h *HTTPHandler) ListProductReturnQC(ctx *app.Context) *app.CostumeResponse {
	p := pagination.QCDashboardPagination(ctx)
	data, err := h.svc.ListProductReturnQC(ctx.Request.Context(), qcDashboardRepo.Filter{
		Limit:    p.Limit,
		Page:     p.Page,
		Offset:   p.Offset(),
		Search:   p.Search,
		DateFrom: p.DateFrom,
		DateTo:   p.DateTo,
		Status:   p.Status,
		UniqCode: p.UniqCode,
	})
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, data)
}

func (h *HTTPHandler) ListDefects(ctx *app.Context) *app.CostumeResponse {
	p := pagination.QCDashboardPagination(ctx)
	data, err := h.svc.ListDefects(ctx.Request.Context(), qcDashboardRepo.Filter{
		Limit:        p.Limit,
		Page:         p.Page,
		Offset:       p.Offset(),
		Search:       p.Search,
		DateFrom:     p.DateFrom,
		DateTo:       p.DateTo,
		UniqCode:     p.UniqCode,
		ReasonCode:   p.ReasonCode,
		DefectSource: p.DefectSource,
	})
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, data)
}

func (h *HTTPHandler) ListIssueTypes(ctx *app.Context) *app.CostumeResponse {
	return app.NewSuccess(ctx, http.StatusOK, map[string]interface{}{"items": h.svc.ListIssueTypes()})
}

func (h *HTTPHandler) ManualReferenceFormOptions(ctx *app.Context) *app.CostumeResponse {
	limit, err := positiveIntQuery(ctx.Query("limit"), 20)
	if err != nil {
		return app.NewError(ctx, err)
	}
	data, err := h.svc.ListManualReferenceOptions(ctx.Request.Context(), ctx.Query("qc_type"), ctx.Query("q"), limit)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, data)
}

func (h *HTTPHandler) CreateManualQCReport(ctx *app.Context) *app.CostumeResponse {
	var req qcDashboardModels.CreateManualQCReportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return app.NewError(ctx, apperror.BadRequest("invalid request body: "+err.Error()))
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	if err := h.svc.CreateManualQCReport(ctx.Request.Context(), req, userCtx.UserID); err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusCreated, map[string]interface{}{"status": "created"})
}

func (h *HTTPHandler) CreateReworkTask(ctx *app.Context) *app.CostumeResponse {
	defectID, err := strconv.ParseInt(ctx.Param("defect_id"), 10, 64)
	if err != nil || defectID <= 0 {
		return app.NewError(ctx, apperror.BadRequest("invalid defect_id"))
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	if err := h.svc.CreateReworkTask(ctx.Request.Context(), defectID, userCtx.UserID); err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusCreated, map[string]interface{}{"defect_id": defectID, "status": "pending"})
}

func positiveIntQuery(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return 0, apperror.BadRequest("query parameter must be a positive integer")
	}
	return v, nil
}
