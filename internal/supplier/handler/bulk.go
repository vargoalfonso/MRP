package handler

import (
	"context"
	"net/http"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/supplier/models"
)

// supplierBulkCreator is satisfied by the concrete supplier service. Using a
// local interface lets the handler call BulkCreate without changing the shared
// SupplierService interface.
type supplierBulkCreator interface {
	BulkCreate(ctx context.Context, items []models.CreateSupplierRequest) (*models.BulkImportResult, error)
}

// BulkCreate handles POST /api/v1/suppliers/bulk. It accepts a JSON array of
// supplier payloads and imports them in one request, returning a per-row report.
func (h *HTTPHandler) BulkCreate(appCtx *app.Context) *app.CostumeResponse {
	var req models.BulkCreateSupplierRequest
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body",
		}
	}

	if len(req.Items) == 0 {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "items wajib diisi",
		}
	}

	creator, ok := h.service.(supplierBulkCreator)
	if !ok {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusInternalServerError,
			Message:   "bulk import belum tersedia",
		}
	}

	result, err := creator.BulkCreate(appCtx.Request.Context(), req.Items)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	status := http.StatusOK
	message := "import selesai"
	switch {
	case result.SuccessCount == 0:
		status = http.StatusUnprocessableEntity
		message = "import gagal"
	case result.FailedCount > 0:
		status = http.StatusMultiStatus
		message = "import sebagian berhasil"
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    status,
		Message:   message,
		Data:      result,
	}
}
