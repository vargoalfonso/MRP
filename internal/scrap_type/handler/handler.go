package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/scrap_type/models"
	"github.com/ganasa18/go-template/internal/scrap_type/service"
	"github.com/ganasa18/go-template/pkg/auth"
	"github.com/ganasa18/go-template/pkg/validator"
)

// HTTPHandler holds scrap type endpoints
type HTTPHandler struct {
	svc service.IService
}

// New constructs handler
func New(svc service.IService) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

// GetScrapTypes handles GET /scrap-types
func (h *HTTPHandler) GetScrapTypes(c *app.Context) *app.CostumeResponse {
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

	q := service.ListQuery{
		Page:   page,
		Limit:  limit,
		Search: c.Query("search"),
		Status: c.Query("status"),
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

// GetScrapTypeByID handles GET /scrap-types/:id
func (h *HTTPHandler) GetScrapTypeByID(c *app.Context) *app.CostumeResponse {
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

// CreateScrapType handles POST /scrap-types
func (h *HTTPHandler) CreateScrapType(c *app.Context) *app.CostumeResponse {
	var req models.CreateScrapTypeRequest
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
		Message:   "scrap type created successfully",
		Data:      resp,
	}
}

// UpdateScrapType handles PUT /scrap-types/:id
func (h *HTTPHandler) UpdateScrapType(c *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: c.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}

	var req models.UpdateScrapTypeRequest
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

// DeleteScrapType handles DELETE /scrap-types/:id
func (h *HTTPHandler) DeleteScrapType(c *app.Context) *app.CostumeResponse {
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
		Message:   "scrap type deleted successfully",
	}
}
