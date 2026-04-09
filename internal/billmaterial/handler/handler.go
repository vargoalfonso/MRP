// Package handler exposes BOM HTTP endpoints.
//
// GET  /api/v1/products/bom        — list (expandable tree)
// POST /api/v1/products/bom        — create (wizard: parent + children in one call)
// GET  /api/v1/products/bom/:id    — detail (full tree with routing + material spec)
package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/billmaterial/models"
	"github.com/ganasa18/go-template/internal/billmaterial/service"
	"github.com/ganasa18/go-template/pkg/pagination"
	"github.com/ganasa18/go-template/pkg/validator"
)

type HTTPHandler struct {
	svc service.IService
}

func New(svc service.IService) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

// ListBom  GET /api/v1/products/bom
//
// Query params:
//
//	limit=20&page=1&search=LV7&uniq_code=LV7&status=Active
//	orderBy=created_at&orderDirection=desc
//	filter=status:eq:Active
func (h *HTTPHandler) ListBom(ctx *app.Context) *app.CostumeResponse {
	p := pagination.BomPagination(ctx)

	resp, err := h.svc.ListBom(ctx.Request.Context(), models.ListBomQuery{
		UniqCode:       p.UniqCode,
		Status:         p.Status,
		Search:         p.Search,
		Page:           p.Page,
		Limit:          p.Limit,
		OrderBy:        p.OrderBy,
		OrderDirection: p.OrderDirection,
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

// CreateBom  POST /api/v1/products/bom
func (h *HTTPHandler) CreateBom(ctx *app.Context) *app.CostumeResponse {
	var req models.CreateBomRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}

	result, err := h.svc.CreateBom(ctx.Request.Context(), req)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusCreated,
		Message:   http.StatusText(http.StatusCreated),
		Data:      result,
	}
}

// GetBomDetail  GET /api/v1/products/bom/:id
func (h *HTTPHandler) GetBomDetail(ctx *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	result, err := h.svc.GetBomDetail(ctx.Request.Context(), id)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      result,
	}
}
