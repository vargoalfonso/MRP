package handler

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	registerService "github.com/ganasa18/go-template/internal/auth/service"
	"github.com/ganasa18/go-template/internal/base/app"
	importService "github.com/ganasa18/go-template/internal/import_file/service"
)

// HTTPHandler holds access control matrix endpoints
type HTTPHandler struct {
	service importService.ImportService
	auth    registerService.Authenticator
}

// New constructs handler
func New(service importService.ImportService, auth registerService.Authenticator) *HTTPHandler {
	return &HTTPHandler{service: service, auth: auth}
}

func (h *HTTPHandler) ImportExcel(appCtx *app.Context) *app.CostumeResponse {
	ctx := appCtx.Request.Context()

	file, err := appCtx.FormFile("file")
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "file wajib diisi",
		}
	}

	if !strings.HasSuffix(file.Filename, ".xlsx") {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "file harus format .xlsx",
		}
	}

	filePath := fmt.Sprintf("./tmp/%d_%s", time.Now().Unix(), file.Filename)

	if err := appCtx.SaveUploadedFile(file, filePath); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusInternalServerError,
			Message:   "gagal menyimpan file",
			Data:      map[string]interface{}{"error": err.Error()},
		}
	}

	defer os.Remove(filePath)

	data, err := h.service.ImportExcel(ctx, filePath)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusInternalServerError,
			Message:   "gagal import excel",
			Data:      map[string]interface{}{"error": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "import excel berhasil",
		Data:      data,
	}
}
