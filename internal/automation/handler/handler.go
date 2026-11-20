package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/automation/models"
	"github.com/ganasa18/go-template/internal/automation/service"
	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/pkg/raigineclient"
	"github.com/ganasa18/go-template/pkg/validator"
)

// HTTPHandler exposes the MRP -> Raigine automation endpoints.
type HTTPHandler struct {
	svc service.IService
}

// New constructs the automation HTTP handler.
func New(svc service.IService) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func (h *HTTPHandler) listParams(ctx *app.Context) raigineclient.ListParams {
	return raigineclient.ListParams{
		Page:       atoiDefault(ctx.Query("page"), 1),
		Limit:      atoiDefault(ctx.Query("limit"), 10),
		Pagination: ctx.DefaultQuery("pagination", "true"),
		FolderID:   ctx.Query("folder_id"),
		MachineID:  ctx.Query("machine_id"),
		ProcessID:  ctx.Query("process_id"),
	}
}

// Run triggers an automation process on the Raigine platform.
func (h *HTTPHandler) Run(ctx *app.Context) *app.CostumeResponse {
	var req models.RunRequest
	_ = ctx.ShouldBindJSON(&req) // body is optional

	resp, err := h.svc.RunProcess(ctx.Request.Context(), ctx.Param("id"), req)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "automation process triggered",
		Data:      resp,
	}
}

// Stop stops a running automation process.
func (h *HTTPHandler) Stop(ctx *app.Context) *app.CostumeResponse {
	resp, err := h.svc.StopProcess(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "automation process stopped",
		Data:      resp,
	}
}

// ListProcesses returns available automation processes.
func (h *HTTPHandler) ListProcesses(ctx *app.Context) *app.CostumeResponse {
	resp, err := h.svc.ListProcesses(ctx.Request.Context(), h.listParams(ctx))
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

// ListJobs returns automation job history.
func (h *HTTPHandler) ListJobs(ctx *app.Context) *app.CostumeResponse {
	resp, err := h.svc.ListJobs(ctx.Request.Context(), h.listParams(ctx))
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

// CreateSchedule registers a cron schedule for an automation process.
func (h *HTTPHandler) CreateSchedule(ctx *app.Context) *app.CostumeResponse {
	var req models.ScheduleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body",
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

	resp, err := h.svc.CreateSchedule(ctx.Request.Context(), req)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusCreated,
		Message:   http.StatusText(http.StatusCreated),
		Data:      resp,
	}
}
