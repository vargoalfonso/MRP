package handler

import (
	"net/http"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/production/models"
	productionService "github.com/ganasa18/go-template/internal/production/service"
	"github.com/ganasa18/go-template/pkg/validator"
)

type HTTPHandler struct {
	service productionService.IProductionService
}

func New(service productionService.IProductionService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) ScanQR(appCtx *app.Context) *app.CostumeResponse {
	var req models.ScanQRRequest

	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body",
		}
	}

	if req.QR == "" {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "QR tidak boleh kosong",
		}
	}

	data, err := h.service.GetScanData(appCtx.Request.Context(), req.QR)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "Error",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "success",
		Data:      data,
	}
}

func (h *HTTPHandler) ScanIn(appCtx *app.Context) *app.CostumeResponse {
	var req models.ProductionScanLog

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

	err := h.service.ScanIn(appCtx.Request.Context(), req)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "scan in success",
	}
}

func (h *HTTPHandler) ScanOut(appCtx *app.Context) *app.CostumeResponse {
	var req models.ProductionScanLog

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

	err := h.service.ScanOut(appCtx.Request.Context(), req)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "scan out success",
	}
}
