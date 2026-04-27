package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	shopFloorService "github.com/ganasa18/go-template/internal/shop_floor/service"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/pagination"
)

type HTTPHandler struct {
	svc shopFloorService.IService
}

func New(svc shopFloorService.IService) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

func (h *HTTPHandler) GetLiveProductionSummary(ctx *app.Context) *app.CostumeResponse {
	limit, err := positiveIntQuery(ctx.Query("limit"), 10)
	if err != nil {
		return app.NewError(ctx, err)
	}
	staleMinutes, err := positiveIntQuery(ctx.Query("stale_minutes"), 30)
	if err != nil {
		return app.NewError(ctx, err)
	}

	result, err := h.svc.GetLiveProductionSummary(ctx.Request.Context(), limit, staleMinutes)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, result)
}

func (h *HTTPHandler) GetDeliveryReadinessSummary(ctx *app.Context) *app.CostumeResponse {
	limit, err := positiveIntQuery(ctx.Query("limit"), 10)
	if err != nil {
		return app.NewError(ctx, err)
	}

	result, err := h.svc.GetDeliveryReadinessSummary(ctx.Request.Context(), limit)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, result)
}

func (h *HTTPHandler) GetProductionIssuesSummary(ctx *app.Context) *app.CostumeResponse {
	limit, err := positiveIntQuery(ctx.Query("limit"), 10)
	if err != nil {
		return app.NewError(ctx, err)
	}
	windowHours, err := positiveIntQuery(ctx.Query("window_hours"), 168)
	if err != nil {
		return app.NewError(ctx, err)
	}

	result, err := h.svc.GetProductionIssuesSummary(ctx.Request.Context(), limit, windowHours)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, result)
}

func (h *HTTPHandler) GetScanEventsSummary(ctx *app.Context) *app.CostumeResponse {
	pag := pagination.Pagination(ctx)
	windowHours, err := positiveIntQuery(ctx.Query("window_hours"), 24)
	if err != nil {
		return app.NewError(ctx, err)
	}

	result, err := h.svc.GetScanEventsSummary(ctx.Request.Context(), pag.Limit, pag.Page, pag.Offset(), windowHours)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, result)
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
