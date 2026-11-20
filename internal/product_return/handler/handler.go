// product_return.go
package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/product_return/models"
	productReturnService "github.com/ganasa18/go-template/internal/product_return/service"
	"github.com/ganasa18/go-template/pkg/validator"
)

type HTTPHandler struct {
	service productReturnService.IProductReturn
}

func New(service productReturnService.IProductReturn) *HTTPHandler {
	return &HTTPHandler{service: service}
}

// ======================================
// GET ALL WITH PAGINATION
// ======================================
func (h *HTTPHandler) GetProductReturns(appCtx *app.Context) *app.CostumeResponse {
	page, _ := strconv.Atoi(appCtx.Query("page"))
	limit, _ := strconv.Atoi(appCtx.Query("limit"))

	if page <= 0 {
		page = 1
	}

	if limit <= 0 {
		limit = 10
	}

	data, total, err := h.service.GetAll(appCtx.Request.Context(), page, limit)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data: map[string]interface{}{
			"data":  data,
			"total": total,
			"page":  page,
			"limit": limit,
		},
	}
}

// ======================================
// CREATE
// ======================================
func (h *HTTPHandler) CreateProductReturn(appCtx *app.Context) *app.CostumeResponse {
	var req models.CreateProductReturnRequest

	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body",
		}
	}

	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data: map[string]interface{}{
				"errors": errs,
			},
		}
	}

	data, err := h.service.Create(appCtx.Request.Context(), req)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "product return created successfully",
		Data:      data,
	}
}

// ======================================
// GET BY ID
// ======================================
func (h *HTTPHandler) GetProductReturnByID(appCtx *app.Context) *app.CostumeResponse {
	idParam := appCtx.Param("id")

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}

	data, err := h.service.GetByID(appCtx.Request.Context(), id)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusNotFound,
			Message:   "Not Found. Product Return does not exist",
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// ======================================
// UPDATE
// ======================================
func (h *HTTPHandler) UpdateProductReturn(appCtx *app.Context) *app.CostumeResponse {
	idParam := appCtx.Param("id")

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}

	_, err = h.service.GetByID(appCtx.Request.Context(), id)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusNotFound,
			Message:   "Not Found. Product Return does not exist",
		}
	}

	var req models.UpdateProductReturnRequest

	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body",
		}
	}

	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data: map[string]interface{}{
				"errors": errs,
			},
		}
	}

	data, err := h.service.Update(appCtx.Request.Context(), id, req)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "product return updated successfully",
		Data:      data,
	}
}

// ======================================
// DELETE
// ======================================
func (h *HTTPHandler) DeleteProductReturn(appCtx *app.Context) *app.CostumeResponse {
	idParam := appCtx.Param("id")

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}

	_, err = h.service.GetByID(appCtx.Request.Context(), id)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusNotFound,
			Message:   "Not Found. Product Return does not exist",
		}
	}

	err = h.service.Delete(appCtx.Request.Context(), id)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "product return deleted successfully",
	}
}
