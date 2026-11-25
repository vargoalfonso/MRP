package handler

import (
	"net/http"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/supplier_item/models"
)

// BomUsage: GET /supplier-items/bom-usage?uniq_codes=BM4,BM6
func (h *HTTPHandler) BomUsage(appCtx *app.Context) *app.CostumeResponse {
	var query models.BomUsageQuery
	if err := appCtx.ShouldBindQuery(&query); err != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusBadRequest, Message: "invalid query params"}
	}
	rows, err := h.service.ListBomUsage(appCtx.Request.Context(), query)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: rows}
}

// BomMaterialGrades: GET /supplier-items/bom-material-grades?type=raw_material&search=SL
func (h *HTTPHandler) BomMaterialGrades(appCtx *app.Context) *app.CostumeResponse {
	var query models.BomMaterialGradeQuery
	if err := appCtx.ShouldBindQuery(&query); err != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusBadRequest, Message: "invalid query params"}
	}
	rows, err := h.service.ListBomMaterialGrades(appCtx.Request.Context(), query)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: rows}
}
