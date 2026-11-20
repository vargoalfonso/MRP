package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	mainDashboardService "github.com/ganasa18/go-template/internal/main_dashboard/service"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/pagination"
)

type HTTPHandler struct {
	svc mainDashboardService.IService
}

func New(svc mainDashboardService.IService) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

func (h *HTTPHandler) GetSummary(ctx *app.Context) *app.CostumeResponse {
	q := pagination.MainDashboardPagination(ctx)
	resp, err := h.svc.GetSummary(ctx.Request.Context(), mainDashboardService.SummaryParams{
		Period:    q.Period,
		StartDate: q.StartDate,
		EndDate:   q.EndDate,
	})
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

func (h *HTTPHandler) GetRawMaterialSummary(ctx *app.Context) *app.CostumeResponse {
	q := pagination.MainDashboardPagination(ctx)
	resp, err := h.svc.GetRawMaterialSummary(ctx.Request.Context(), mainDashboardService.SummaryParams{
		Period:    q.Period,
		StartDate: q.StartDate,
		EndDate:   q.EndDate,
	})
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

func (h *HTTPHandler) ListTables(ctx *app.Context) *app.CostumeResponse {
	limit := 200
	if raw := ctx.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			return app.NewError(ctx, apperror.BadRequest("limit must be a positive integer"))
		}
		limit = v
	}

	resp, err := h.svc.GetListTables(ctx.Request.Context(), ctx.Query("schema"), limit)
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
