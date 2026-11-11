package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/machine_pattern/models"
	"github.com/ganasa18/go-template/internal/machine_pattern/service"
	"github.com/ganasa18/go-template/pkg/auth"
	"github.com/ganasa18/go-template/pkg/validator"
)

type HTTPHandler struct {
	svc service.IService
}

func New(svc service.IService) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

// GetMachinePatterns handles GET /machine-patterns
func (h *HTTPHandler) GetMachinePatterns(c *app.Context) *app.CostumeResponse {
	page := 1
	limit := 20
	if p := c.Query("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			limit = n
		}
	}

	var machineID int64
	if mid := c.Query("machine_id"); mid != "" {
		if n, err := strconv.ParseInt(mid, 10, 64); err == nil && n > 0 {
			machineID = n
		}
	}

	q := service.ListQuery{
		Page:       page,
		Limit:      limit,
		Search:     c.Query("search"),
		MachineID:  machineID,
		MovingType: c.Query("moving_type"),
		Status:     c.Query("status"),
		UniqCode:   c.Query("uniq_code"),
	}

	resp, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		return app.NewError(c, err)
	}

	return &app.CostumeResponse{
		RequestID: c.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      resp,
	}
}

// GetMachinePatternByID handles GET /machine-patterns/:id
func (h *HTTPHandler) GetMachinePatternByID(c *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: c.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}

	resp, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		return app.NewError(c, err)
	}

	return &app.CostumeResponse{
		RequestID: c.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      resp,
	}
}

// GetMachinePatternSummary handles GET /machine-patterns/summary
func (h *HTTPHandler) GetMachinePatternSummary(c *app.Context) *app.CostumeResponse {
	resp, err := h.svc.GetSummary(c.Request.Context())
	if err != nil {
		return app.NewError(c, err)
	}

	return &app.CostumeResponse{
		RequestID: c.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      resp,
	}
}

// CreateMachinePattern handles POST /machine-patterns
func (h *HTTPHandler) CreateMachinePattern(c *app.Context) *app.CostumeResponse {
	var req models.CreateMachinePatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: c.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body",
		}
	}

	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: c.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": errs},
		}
	}

	userCtx := auth.MustExtractUserContext(c)

	resp, err := h.svc.Create(c.Request.Context(), req, userCtx.UserID)
	if err != nil {
		return app.NewError(c, err)
	}

	return &app.CostumeResponse{
		RequestID: c.APIReqID,
		Status:    http.StatusCreated,
		Message:   "machine pattern created successfully",
		Data:      resp,
	}
}

// UpdateMachinePattern handles PUT /machine-patterns/:id
func (h *HTTPHandler) UpdateMachinePattern(c *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: c.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}

	var req models.UpdateMachinePatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: c.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body",
		}
	}

	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: c.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": errs},
		}
	}

	resp, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		return app.NewError(c, err)
	}

	return &app.CostumeResponse{
		RequestID: c.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      resp,
	}
}

// DeleteMachinePattern handles DELETE /machine-patterns/:id
func (h *HTTPHandler) DeleteMachinePattern(c *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: c.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		return app.NewError(c, err)
	}

	return &app.CostumeResponse{
		RequestID: c.APIReqID,
		Status:    http.StatusOK,
		Message:   "machine pattern deleted successfully",
	}
}

// BulkCreateMachinePattern handles POST /machine-patterns/bulk
func (h *HTTPHandler) BulkCreateMachinePattern(c *app.Context) *app.CostumeResponse {
	var req models.BulkMachinePatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: c.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body",
		}
	}

	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: c.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": errs},
		}
	}

	userCtx := auth.MustExtractUserContext(c)

	resp, err := h.svc.BulkCreate(c.Request.Context(), req, userCtx.UserID)
	if err != nil {
		return app.NewError(c, err)
	}

	return &app.CostumeResponse{
		RequestID: c.APIReqID,
		Status:    http.StatusCreated,
		Message:   "bulk machine patterns processed successfully",
		Data:      resp,
	}
}
