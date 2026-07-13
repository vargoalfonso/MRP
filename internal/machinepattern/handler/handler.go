// Package handler exposes Machine Pattern HTTP endpoints.
//
// GET    /api/v1/machine-patterns            — list
// POST   /api/v1/machine-patterns            — create single
// POST   /api/v1/machine-patterns/bulk       — bulk create
// GET    /api/v1/machine-patterns/calculate  — preview calculation
// GET    /api/v1/machine-patterns/params     — get global params
// PUT    /api/v1/machine-patterns/params     — update global params
// GET    /api/v1/machine-patterns/safety-stock — safety stock output
// GET    /api/v1/machine-patterns/:id        — detail
// PUT    /api/v1/machine-patterns/:id        — update
// DELETE /api/v1/machine-patterns/:id        — delete
package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/machinepattern/models"
	"github.com/ganasa18/go-template/internal/machinepattern/service"
	"github.com/ganasa18/go-template/pkg/validator"
)

type HTTPHandler struct {
	svc service.IService
}

func New(svc service.IService) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

// ---------------------------------------------------------------------------
// List  GET /api/v1/machine-patterns
// Query: page, limit, uniq_code, machine_id, moving_type, status, search
// ---------------------------------------------------------------------------

func (h *HTTPHandler) List(ctx *app.Context) *app.CostumeResponse {
	q := models.ListQuery{
		UniqCode:   ctx.Query("uniq_code"),
		MovingType: ctx.Query("moving_type"),
		Status:     ctx.Query("status"),
		Search:     ctx.Query("search"),
	}
	if mid, err := strconv.ParseInt(ctx.Query("machine_id"), 10, 64); err == nil {
		q.MachineID = mid
	}
	if p, err := strconv.Atoi(ctx.Query("page")); err == nil && p > 0 {
		q.Page = p
	}
	if l, err := strconv.Atoi(ctx.Query("limit")); err == nil && l > 0 {
		q.Limit = l
	}

	resp, err := h.svc.List(ctx.Request.Context(), q)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Create  POST /api/v1/machine-patterns
// ---------------------------------------------------------------------------

func (h *HTTPHandler) Create(ctx *app.Context) *app.CostumeResponse {
	var req models.CreateMachinePatternRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed",
			Data: map[string]interface{}{"errors": errs}}
	}

	resp, err := h.svc.Create(ctx.Request.Context(), req)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusCreated, resp)
}

// ---------------------------------------------------------------------------
// BulkCreate  POST /api/v1/machine-patterns/bulk
// ---------------------------------------------------------------------------

func (h *HTTPHandler) BulkCreate(ctx *app.Context) *app.CostumeResponse {
	var req models.BulkCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed",
			Data: map[string]interface{}{"errors": errs}}
	}

	resp, err := h.svc.BulkCreate(ctx.Request.Context(), req)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusCreated, resp)
}

// ---------------------------------------------------------------------------
// GetByID  GET /api/v1/machine-patterns/:id
// ---------------------------------------------------------------------------

func (h *HTTPHandler) GetByID(ctx *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	resp, err := h.svc.GetByID(ctx.Request.Context(), id)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Update  PUT /api/v1/machine-patterns/:id
// ---------------------------------------------------------------------------

func (h *HTTPHandler) Update(ctx *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	var req models.UpdateMachinePatternRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	resp, err := h.svc.Update(ctx.Request.Context(), id, req)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Delete  DELETE /api/v1/machine-patterns/:id
// ---------------------------------------------------------------------------

func (h *HTTPHandler) Delete(ctx *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	if err := h.svc.Delete(ctx.Request.Context(), id); err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, map[string]interface{}{"deleted": true})
}

// ---------------------------------------------------------------------------
// Calculate (preview)  GET /api/v1/machine-patterns/calculate
// Query: uniq_code, machine_id, cycle_time_sec, prl_reference, working_days
// ---------------------------------------------------------------------------

func (h *HTTPHandler) Calculate(ctx *app.Context) *app.CostumeResponse {
	req := models.CalculateRequest{
		UniqCode: ctx.Query("uniq_code"),
	}
	if v, err := strconv.ParseInt(ctx.Query("machine_id"), 10, 64); err == nil {
		req.MachineID = v
	}
	if v, err := strconv.ParseFloat(ctx.Query("cycle_time_sec"), 64); err == nil {
		req.CycleTimeSec = v
	}
	if v, err := strconv.ParseFloat(ctx.Query("prl_reference"), 64); err == nil {
		req.PrlReference = v
	}
	if v, err := strconv.Atoi(ctx.Query("working_days")); err == nil {
		req.WorkingDays = v
	}

	resp, err := h.svc.Calculate(ctx.Request.Context(), req)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// GetParams  GET /api/v1/machine-patterns/params
// ---------------------------------------------------------------------------

func (h *HTTPHandler) GetParams(ctx *app.Context) *app.CostumeResponse {
	resp, err := h.svc.GetParams(ctx.Request.Context())
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// UpdateParams  PUT /api/v1/machine-patterns/params
// ---------------------------------------------------------------------------

func (h *HTTPHandler) UpdateParams(ctx *app.Context) *app.CostumeResponse {
	var req models.UpdateParamRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	resp, err := h.svc.UpdateParams(ctx.Request.Context(), req)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// SafetyStock  GET /api/v1/machine-patterns/safety-stock
// ---------------------------------------------------------------------------

func (h *HTTPHandler) SafetyStock(ctx *app.Context) *app.CostumeResponse {
	resp, err := h.svc.ListSafetyStock(ctx.Request.Context())
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, resp)
}
