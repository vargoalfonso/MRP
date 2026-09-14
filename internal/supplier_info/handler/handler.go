package handler

import (
	"net/http"

	"github.com/ganasa18/go-template/internal/base/app"
	siModels "github.com/ganasa18/go-template/internal/supplier_info/models"
	siService "github.com/ganasa18/go-template/internal/supplier_info/service"
	"github.com/ganasa18/go-template/pkg/validator"
)

type HTTPHandler struct {
	svc siService.SupplierInfoService
}

func New(svc siService.SupplierInfoService) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

func (h *HTTPHandler) Create(appCtx *app.Context) *app.CostumeResponse {
	var req siModels.CreateSupplierInfoRequest
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	info, err := h.svc.Create(appCtx.Request.Context(), req)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusCreated, Message: "supplier info created", Data: info}
}

func (h *HTTPHandler) GetByID(appCtx *app.Context) *app.CostumeResponse {
	info, err := h.svc.GetByUUID(appCtx.Request.Context(), appCtx.Param("id"))
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: "OK", Data: info}
}

func (h *HTTPHandler) List(appCtx *app.Context) *app.CostumeResponse {
	var query siModels.ListSupplierInfoQuery
	if err := appCtx.ShouldBindQuery(&query); err != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusBadRequest, Message: "invalid query params"}
	}
	items, total, err := h.svc.List(appCtx.Request.Context(), query)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: "OK", Data: map[string]interface{}{"items": items, "total": total}}
}

func (h *HTTPHandler) Update(appCtx *app.Context) *app.CostumeResponse {
	var req siModels.UpdateSupplierInfoRequest
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	info, err := h.svc.Update(appCtx.Request.Context(), appCtx.Param("id"), req)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: "supplier info updated", Data: info}
}

func (h *HTTPHandler) Delete(appCtx *app.Context) *app.CostumeResponse {
	if err := h.svc.Delete(appCtx.Request.Context(), appCtx.Param("id")); err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: "supplier info deleted"}
}
