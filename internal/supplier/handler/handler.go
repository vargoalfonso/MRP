package handler

import (
	"net/http"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/supplier/models"
	supplierService "github.com/ganasa18/go-template/internal/supplier/service"
	"github.com/ganasa18/go-template/pkg/validator"
)

type HTTPHandler struct {
	service supplierService.SupplierService
}

func New(service supplierService.SupplierService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) Create(appCtx *app.Context) *app.CostumeResponse {
	var req models.CreateSupplierRequest
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
			Data:      map[string]interface{}{"errors": errs},
		}
	}

	supplier, err := h.service.Create(appCtx.Request.Context(), req)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "supplier created successfully",
		Data:      supplier,
	}
}

func (h *HTTPHandler) GetByID(appCtx *app.Context) *app.CostumeResponse {
	supplier, err := h.service.GetByUUID(appCtx.Request.Context(), appCtx.Param("id"))
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      supplier,
	}
}

func (h *HTTPHandler) List(appCtx *app.Context) *app.CostumeResponse {
	var query models.ListSupplierQuery
	if err := appCtx.ShouldBindQuery(&query); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid query params",
		}
	}

	result, err := h.service.List(appCtx.Request.Context(), query)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      result,
	}
}

func (h *HTTPHandler) Update(appCtx *app.Context) *app.CostumeResponse {
	var req models.UpdateSupplierRequest
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
			Data:      map[string]interface{}{"errors": errs},
		}
	}

	supplier, err := h.service.Update(appCtx.Request.Context(), appCtx.Param("id"), req)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "supplier updated successfully",
		Data:      supplier,
	}
}

func (h *HTTPHandler) Delete(appCtx *app.Context) *app.CostumeResponse {
	if err := h.service.Delete(appCtx.Request.Context(), appCtx.Param("id")); err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "supplier deleted successfully",
	}
}
