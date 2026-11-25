package handler

import (
	"bytes"
	"io"
	"net/http"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/supplier_item/models"
	supplierItemService "github.com/ganasa18/go-template/internal/supplier_item/service"
	"github.com/ganasa18/go-template/pkg/validator"
)

type HTTPHandler struct {
	service supplierItemService.SupplierItemService
}

func New(service supplierItemService.SupplierItemService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) Create(appCtx *app.Context) *app.CostumeResponse {
	rawPayload, err := readAndRestoreRequestBody(appCtx)
	if err != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}

	var req models.CreateSupplierItemRequest
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	item, err := h.service.Create(appCtx.Request.Context(), req, rawPayload)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusCreated, Message: "supplier item created successfully", Data: item}
}

func (h *HTTPHandler) GetByID(appCtx *app.Context) *app.CostumeResponse {
	item, err := h.service.GetByUUID(appCtx.Request.Context(), appCtx.Param("id"))
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: item}
}

func (h *HTTPHandler) List(appCtx *app.Context) *app.CostumeResponse {
	var query models.ListSupplierItemQuery
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
	rawPayload, err := readAndRestoreRequestBody(appCtx)
	if err != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}

	var req models.UpdateSupplierItemRequest
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	item, err := h.service.Update(appCtx.Request.Context(), appCtx.Param("id"), req, rawPayload)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: "supplier item updated successfully", Data: item}
}

func (h *HTTPHandler) Delete(appCtx *app.Context) *app.CostumeResponse {
	if err := h.service.Delete(appCtx.Request.Context(), appCtx.Param("id")); err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: "supplier item deleted successfully"}
}

func readAndRestoreRequestBody(appCtx *app.Context) ([]byte, error) {
	rawPayload, err := io.ReadAll(appCtx.Request.Body)
	if err != nil {
		return nil, err
	}
	appCtx.Request.Body = io.NopCloser(bytes.NewReader(rawPayload))
	return rawPayload, nil
}
