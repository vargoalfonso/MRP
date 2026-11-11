package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ganasa18/go-template/internal/base/app"
	siModels "github.com/ganasa18/go-template/internal/supplier_info/models"
	siService "github.com/ganasa18/go-template/internal/supplier_info/service"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/approval"
	"github.com/ganasa18/go-template/pkg/validator"
	"github.com/gin-gonic/gin"
)

const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

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

// ImportExcel POST /api/v1/supplier-info/import (multipart, field "file", .xlsx)
func (h *HTTPHandler) ImportExcel(appCtx *app.Context) *app.CostumeResponse {
	file, err := appCtx.FormFile("file")
	if err != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusBadRequest, Message: "file wajib diisi"}
	}
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".xlsx") {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusBadRequest, Message: "file harus format .xlsx"}
	}
	src, err := file.Open()
	if err != nil {
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusInternalServerError, Message: "gagal membuka file"}
	}
	defer src.Close()

	result, err := h.svc.ImportFromExcel(appCtx.Request.Context(), src, file.Filename)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	downloadURL := ""
	if result.ErrorToken != "" {
		scheme := "http"
		if proto := appCtx.GetHeader("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else if appCtx.Request.TLS != nil {
			scheme = "https"
		}
		downloadURL = fmt.Sprintf("%s://%s/api/v1/supplier-info/import/errors/%s", scheme, appCtx.Request.Host, result.ErrorToken)
	}

	status, message, data := approval.BuildBulkImportResponse(result, downloadURL)
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: status, Message: message, Data: data}
}

// DownloadImportTemplateRaw GET /api/v1/supplier-info/import/template
func (h *HTTPHandler) DownloadImportTemplateRaw(c *gin.Context) {
	data, err := h.svc.DownloadImportTemplate(c.Request.Context())
	if err != nil {
		writeRawError(c, err, "gagal generate template")
		return
	}
	c.Header("Content-Disposition", "attachment; filename=supplier_info_template.xlsx")
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.Data(http.StatusOK, xlsxContentType, data)
}

// DownloadImportErrorsRaw GET /api/v1/supplier-info/import/errors/:token
func (h *HTTPHandler) DownloadImportErrorsRaw(c *gin.Context) {
	data, err := h.svc.DownloadImportErrors(c.Request.Context(), c.Param("token"))
	if err != nil {
		writeRawError(c, err, "gagal download file error")
		return
	}
	c.Header("Content-Disposition", "attachment; filename=supplier_info_import_errors.xlsx")
	c.Data(http.StatusOK, xlsxContentType, data)
}

func writeRawError(c *gin.Context, err error, fallback string) {
	status := http.StatusInternalServerError
	msg := fallback
	if appErr, ok := apperror.As(err); ok {
		status = appErr.HTTPStatus
		msg = appErr.Message
	}
	c.JSON(status, gin.H{"status": status, "message": msg})
}
