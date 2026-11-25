package handler

import (
	"net/http"

	"github.com/ganasa18/go-template/internal/base/app"
	productionDashboardService "github.com/ganasa18/go-template/internal/production_dashboard/service"
)

type HTTPHandler struct {
	svc productionDashboardService.IService
}

// New builds the Production Dashboard HTTP handler.
func New(svc productionDashboardService.IService) *HTTPHandler { return &HTTPHandler{svc: svc} }

// FGDashboard serves GET /production-dashboard/fg-dashboard.
func (h *HTTPHandler) FGDashboard(ctx *app.Context) *app.CostumeResponse {
	data, err := h.svc.FinishedGoods(ctx.Request.Context())
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, data)
}

// WIPDashboard serves GET /production-dashboard/wip-dashboard.
func (h *HTTPHandler) WIPDashboard(ctx *app.Context) *app.CostumeResponse {
	data, err := h.svc.Wip(ctx.Request.Context())
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, data)
}

// OutputMachineDashboard serves GET /production-dashboard/output-machine-dashboard.
func (h *HTTPHandler) OutputMachineDashboard(ctx *app.Context) *app.CostumeResponse {
	data, err := h.svc.OutputPerMachine(ctx.Request.Context())
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, data)
}

// SummaryStrokeDashboard serves GET /production-dashboard/summary-stroke-dashboard.
func (h *HTTPHandler) SummaryStrokeDashboard(ctx *app.Context) *app.CostumeResponse {
	data, err := h.svc.SummaryStroke(ctx.Request.Context())
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, data)
}

// Runtime serves GET /production-dashboard/runtime.
func (h *HTTPHandler) Runtime(ctx *app.Context) *app.CostumeResponse {
	data, err := h.svc.Runtime(ctx.Request.Context())
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, data)
}
