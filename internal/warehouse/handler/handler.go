package handler

import (
	"net/http"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/warehouse/models"
	warehouseService "github.com/ganasa18/go-template/internal/warehouse/service"
	"github.com/ganasa18/go-template/pkg/validator"
)

type HTTPHandler struct {
	service warehouseService.WarehouseService
}

func New(service warehouseService.WarehouseService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) Create(appCtx *app.Context) *app.CostumeResponse {
	var req models.CreateWarehouseRequest
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	warehouse, err := h.service.Create(appCtx.Request.Context(), req)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusCreated, Message: "warehouse created successfully", Data: warehouse}
}

func (h *HTTPHandler) GetByID(appCtx *app.Context) *app.CostumeResponse {
	warehouse, err := h.service.GetByUUID(appCtx.Request.Context(), appCtx.Param("id"))
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: warehouse}
}

func (h *HTTPHandler) List(appCtx *app.Context) *app.CostumeResponse {
	var query models.ListWarehouseQuery
	if err := appCtx.ShouldBindQuery(&query); err != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusBadRequest, Message: "invalid query params"}
	}
	result, err := h.service.List(appCtx.Request.Context(), query)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: result}
}

func (h *HTTPHandler) Update(appCtx *app.Context) *app.CostumeResponse {
	var req models.UpdateWarehouseRequest
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	warehouse, err := h.service.Update(appCtx.Request.Context(), appCtx.Param("id"), req)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: "warehouse updated successfully", Data: warehouse}
}

func (h *HTTPHandler) Delete(appCtx *app.Context) *app.CostumeResponse {
	if err := h.service.Delete(appCtx.Request.Context(), appCtx.Param("id")); err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: "warehouse deleted successfully"}
}
