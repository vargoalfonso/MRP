package handler

import (
	"net/http"

	"github.com/ganasa18/go-template/internal/admin_jobs/service"
	"github.com/ganasa18/go-template/internal/base/app"
)

type HTTPHandler struct {
	svc service.IService
}

func New(svc service.IService) *HTTPHandler { return &HTTPHandler{svc: svc} }

// RebuildPRLPeriodSummaries rebuilds the prl_item_period_summaries cache for the given forecast period.
//
//	POST /api/v1/admin/jobs/rebuild-prl-period-summaries?forecast_period=April+2026
func (h *HTTPHandler) RebuildPRLPeriodSummaries(ctx *app.Context) *app.CostumeResponse {
	forecastPeriod := ctx.Query("forecast_period")

	n, usedPeriod, err := h.svc.RebuildPRLPeriodSummaries(ctx.Request.Context(), forecastPeriod)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusInternalServerError,
			Message:   "failed to rebuild prl period summaries: " + err.Error(),
		}
	}

	if usedPeriod == "" {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusOK,
			Message:   "no approved PRL period found; nothing rebuilt",
			Data: map[string]interface{}{
				"rows_upserted":   0,
				"forecast_period": "",
			},
		}
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data: map[string]interface{}{
			"rows_upserted":   n,
			"forecast_period": usedPeriod,
		},
	}
}
