package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	registerService "github.com/ganasa18/go-template/internal/auth/service"
	"github.com/ganasa18/go-template/internal/base/app"
	importService "github.com/ganasa18/go-template/internal/import_file/service"
)

type HTTPHandler struct {
	service importService.ImportService
	auth    registerService.Authenticator
}

func New(service importService.ImportService, auth registerService.Authenticator) *HTTPHandler {
	return &HTTPHandler{service: service, auth: auth}
}

func (h *HTTPHandler) DownloadTemplate(appCtx *app.Context) *app.CostumeResponse {
	file, err := h.service.GenerateTemplatePrls()
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusInternalServerError,
			Message:   "gagal generate template",
			Data:      map[string]interface{}{"error": err.Error()},
		}
	}

	appCtx.Writer.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	appCtx.Writer.Header().Set("Content-Disposition", "attachment; filename=template_import_prls.xlsx")

	appCtx.Writer.WriteHeader(http.StatusOK)
	_, _ = appCtx.Writer.Write(file.Bytes())

	return nil
}

func (h *HTTPHandler) BulkImportPRL(appCtx *app.Context) *app.CostumeResponse {
	file, err := appCtx.FormFile("file")
	if err != nil {
		return &app.CostumeResponse{Status: http.StatusBadRequest, Message: "file wajib diisi"}
	}

	if !strings.HasSuffix(file.Filename, ".xlsx") {
		return &app.CostumeResponse{Status: http.StatusBadRequest, Message: "file harus .xlsx"}
	}

	tmpDir := "./tmp"
	_ = os.MkdirAll(tmpDir, os.ModePerm)

	timestamp := time.Now().Unix()
	cleanFileName := strings.ReplaceAll(file.Filename, " ", "_")
	fileName := fmt.Sprintf("FAILED_IMPORT_%d_%s", timestamp, cleanFileName)
	filePath := filepath.Join(tmpDir, fileName)

	if err := appCtx.SaveUploadedFile(file, filePath); err != nil {
		return &app.CostumeResponse{Status: http.StatusInternalServerError, Message: "gagal simpan file"}
	}

	data, err := h.service.ParsingPRL(appCtx.Request.Context(), filePath)
	if err != nil {
		os.Remove(filePath)
		return &app.CostumeResponse{Status: http.StatusBadRequest, Message: "gagal parsing excel", Data: err.Error()}
	}

	if len(data) == 0 {
		os.Remove(filePath)
		return &app.CostumeResponse{Status: http.StatusBadRequest, Message: "data kosong"}
	}

	result, err := h.service.BulkInsertPRL(appCtx.Request.Context(), data, filePath)
	if err != nil {
		os.Remove(filePath)
		return &app.CostumeResponse{Status: http.StatusInternalServerError, Message: "gagal insert data"}
	}

	response := map[string]interface{}{
		"total_data": len(data),
		"success":    result.Success,
		"failed":     result.Failed,
	}

	if result.Failed > 0 {
		response["failed_file"] = fileName
	} else {
		response["failed_file"] = nil
		os.Remove(filePath)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "import selesai",
		Data:      response,
	}
}

func (h *HTTPHandler) DownloadFailedFile(appCtx *app.Context) *app.CostumeResponse {
	fileName := filepath.Base(appCtx.Param("filename"))
	filePath := "./tmp/" + fileName

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return &app.CostumeResponse{
			Status:  http.StatusNotFound,
			Message: "file sudah kadaluarsa atau tidak ditemukan",
		}
	}

	appCtx.Writer.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	appCtx.Writer.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	http.ServeFile(appCtx.Writer, appCtx.Request, filePath)

	return nil
}
