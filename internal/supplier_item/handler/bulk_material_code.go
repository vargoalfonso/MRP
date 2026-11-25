package handler

import (
	"net/http"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/supplier_item/models"
	"github.com/ganasa18/go-template/pkg/validator"
)

// PreviewByMaterialCode: GET /supplier-items/by-material-code?material_code=BR50&supplier_uuid=...&type=raw_material
func (h *HTTPHandler) PreviewByMaterialCode(appCtx *app.Context) *app.CostumeResponse {
	var query models.MaterialCodePreviewQuery
	if err := appCtx.ShouldBindQuery(&query); err != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusBadRequest, Message: "invalid query params"}
	}
	rows, err := h.service.PreviewByMaterialCode(appCtx.Request.Context(), query)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: rows}
}

// CreateByMaterialCode: POST /supplier-items/bulk-by-material-code
func (h *HTTPHandler) CreateByMaterialCode(appCtx *app.Context) *app.CostumeResponse {
	var req models.BulkByMaterialCodeRequest
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	result, err := h.service.CreateByMaterialCode(appCtx.Request.Context(), req)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusCreated, Message: "supplier items created by material code", Data: result}
}
