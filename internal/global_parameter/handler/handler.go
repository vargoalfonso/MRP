package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/global_parameter/models"
	globalParameterService "github.com/ganasa18/go-template/internal/global_parameter/service"
	"github.com/ganasa18/go-template/pkg/validator"
)

// HTTPHandler holds global parameter endpoints
type HTTPHandler struct {
	service globalParameterService.IGlobalParameterService
}

// New constructs handler
func New(service globalParameterService.IGlobalParameterService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) GetGlobalParameters(appCtx *app.Context) *app.CostumeResponse {
	data, err := h.service.GetAll(appCtx.Request.Context())
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

func (h *HTTPHandler) CreateGlobalParameter(appCtx *app.Context) *app.CostumeResponse {
	var req models.CreateGlobalParameterRequest

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

	data, err := h.service.Create(appCtx.Request.Context(), req)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "global parameter created successfully",
		Data:      data,
	}
}

func (h *HTTPHandler) GetGlobalParameterByID(appCtx *app.Context) *app.CostumeResponse {
	idParam := appCtx.Param("id")

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}

	data, err := h.service.GetByID(appCtx.Request.Context(), id)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusNotFound,
			Message:   "Not Found. Global parameter does not exist",
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

func (h *HTTPHandler) UpdateGlobalParameter(appCtx *app.Context) *app.CostumeResponse {
	idParam := appCtx.Param("id")

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}

	_, err = h.service.GetByID(appCtx.Request.Context(), id)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusNotFound,
			Message:   "Not Found. Global parameter does not exist",
		}
	}

	var req models.UpdateGlobalParameterRequest
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

	data, err := h.service.Update(appCtx.Request.Context(), id, req)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "global parameter updated successfully",
		Data:      data,
	}
}

func (h *HTTPHandler) DeleteGlobalParameter(appCtx *app.Context) *app.CostumeResponse {
	idParam := appCtx.Param("id")

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}

	_, err = h.service.GetByID(appCtx.Request.Context(), id)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusNotFound,
			Message:   "Not Found. Global parameter does not exist",
		}
	}

	err = h.service.Delete(appCtx.Request.Context(), id)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "global parameter deleted successfully",
	}
}
