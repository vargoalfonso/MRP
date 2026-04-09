package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ganasa18/go-template/internal/access_control/models"
	acmService "github.com/ganasa18/go-template/internal/access_control/service"
	registerService "github.com/ganasa18/go-template/internal/auth/service"
	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/pkg/validator"
)

// HTTPHandler holds access control matrix endpoints
type HTTPHandler struct {
	service acmService.IACMService
	auth    registerService.Authenticator
}

// New constructs handler
func New(service acmService.IACMService, auth registerService.Authenticator) *HTTPHandler {
	return &HTTPHandler{service: service, auth: auth}
}

func (h *HTTPHandler) GetACMs(appCtx *app.Context) *app.CostumeResponse {
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

func (h *HTTPHandler) CreateACM(appCtx *app.Context) *app.CostumeResponse {
	var req models.CreateACMRequest

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
		msg := "Error" + err.Error()

		if strings.Contains(err.Error(), "username") {
			msg = "username sudah digunakan"
		}

		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   msg,
			Data:      nil,
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "User created successfully",
		Data:      data,
	}
}

func (h *HTTPHandler) GetACMByID(appCtx *app.Context) *app.CostumeResponse {
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
			Message:   "Not Found. User does not exist",
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

func (h *HTTPHandler) UpdateACM(appCtx *app.Context) *app.CostumeResponse {
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
			Message:   "Not Found. User does not exist",
		}
	}

	var req models.UpdateACMRequest
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
		Message:   "User updated successfully",
		Data:      data,
	}
}

func (h *HTTPHandler) DeleteACM(appCtx *app.Context) *app.CostumeResponse {
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
			Message:   "Not Found. User does not exist",
		}
	}

	err = h.service.Delete(appCtx.Request.Context(), id)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "User deleted successfully",
	}
}
