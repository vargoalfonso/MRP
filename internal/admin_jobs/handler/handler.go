package handler

import (
	"net/http"
	"time"

	"github.com/ganasa18/go-template/internal/admin_jobs/service"
	"github.com/ganasa18/go-template/internal/base/app"
)

// RecomputeSupplierPerformance handles POST /api/v1/admin/jobs/supplier-performance/recompute.
//
// Request body:
//
//	{ "snapshot_date": "2026-04-23" }
func (h *HTTPHandler) RecomputeSupplierPerformance(ctx *app.Context) *app.CostumeResponse {
	var req service.RecomputeSupplierPerformanceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body",
		}
	}
	n, err := h.svc.RecomputeSupplierPerformance(ctx.Request.Context(), req)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusInternalServerError,
			Message:   "failed to recompute supplier performance: " + err.Error(),
		}
	}

	snapshotDate := req.SnapshotDate
	if snapshotDate == "" {
		snapshotDate = time.Now().UTC().Format("2006-01-02")
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusAccepted,
		Message:   "Supplier performance recompute queued",
		Data: map[string]interface{}{
			"rows_upserted": n,
			"period_type":   "daily",
			"snapshot_date": snapshotDate,
		},
	}
}

type HTTPHandler struct {
	svc service.IService
}

func New(svc service.IService) *HTTPHandler { return &HTTPHandler{svc: svc} }

// RebuildPRLPeriodSummaries rebuilds inventory_demand_periode_summaries for today.
// The active_periode is resolved from global parameters — no query param is needed.
//
//	POST /api/v1/admin/jobs/rebuild-prl-period-summaries
func (h *HTTPHandler) RebuildPRLPeriodSummaries(ctx *app.Context) *app.CostumeResponse {
	n, activePeriode, err := h.svc.RebuildDemandPeriodeSummaries(ctx.Request.Context())
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusInternalServerError,
			Message:   "failed to rebuild demand periode summaries: " + err.Error(),
		}
	}

	if activePeriode == "" {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusOK,
			Message:   "no active working-days period found in global parameters; nothing rebuilt",
			Data: map[string]interface{}{
				"rows_upserted":  0,
				"active_periode": "",
				"snapshot_date":  "",
			},
		}
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data: map[string]interface{}{
			"rows_upserted":  n,
			"active_periode": activePeriode,
			"snapshot_date":  time.Now().Format("2006-01-02"),
		},
	}
}
