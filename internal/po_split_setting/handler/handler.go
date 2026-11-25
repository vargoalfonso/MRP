package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/po_split_setting/models"
	poSplitSettingService "github.com/ganasa18/go-template/internal/po_split_setting/service"
	"github.com/ganasa18/go-template/pkg/validator"
)

// HTTPHandler holds unit measurement endpoints
type HTTPHandler struct {
	service poSplitSettingService.IPOSplitSettingService
}

// New constructs handler
func New(service poSplitSettingService.IPOSplitSettingService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) GetAll(appCtx *app.Context) *app.CostumeResponse {
	data, err := h.service.GetAll(appCtx.Request.Context())
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		Status: http.StatusOK,
		Data:   data,
	}
}

func (h *HTTPHandler) GetByID(appCtx *app.Context) *app.CostumeResponse {
	id, _ := strconv.ParseInt(appCtx.Param("id"), 10, 64)

	data, err := h.service.GetByID(appCtx.Request.Context(), id)
	if err != nil {
		return &app.CostumeResponse{
			Status:  http.StatusNotFound,
			Message: "not found",
		}
	}

	return &app.CostumeResponse{
		Status: http.StatusOK,
		Data:   data,
	}
}

func (h *HTTPHandler) Create(appCtx *app.Context) *app.CostumeResponse {
	var req models.CreatePOSplitRequest

	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid request",
		}
	}

	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			Status:  http.StatusUnprocessableEntity,
			Message: "validation failed",
			Data:    errs,
		}
	}

	data, err := h.service.Create(appCtx.Request.Context(), req)
	if err != nil {
		return &app.CostumeResponse{
			Status:  http.StatusUnprocessableEntity,
			Message: "validation failed",
			Data:    map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		Status:  http.StatusCreated,
		Message: "created",
		Data:    data,
	}
}

func (h *HTTPHandler) Update(appCtx *app.Context) *app.CostumeResponse {
	id, _ := strconv.ParseInt(appCtx.Param("id"), 10, 64)

	var req models.UpdatePOSplitRequest
	appCtx.ShouldBindJSON(&req)

	data, err := h.service.Update(appCtx.Request.Context(), id, req)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		Status:  http.StatusOK,
		Message: "updated",
		Data:    data,
	}
}

func (h *HTTPHandler) Delete(appCtx *app.Context) *app.CostumeResponse {
	id, _ := strconv.ParseInt(appCtx.Param("id"), 10, 64)

	if err := h.service.Delete(appCtx.Request.Context(), id); err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		Status:  http.StatusOK,
		Message: "deleted",
	}
}
