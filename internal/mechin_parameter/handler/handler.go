package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/mechin_parameter/models"
	mechinService "github.com/ganasa18/go-template/internal/mechin_parameter/service"
	"github.com/ganasa18/go-template/pkg/pagination"
	"github.com/ganasa18/go-template/pkg/validator"
)

// HTTPHandler holds mechin parameter endpoints
type HTTPHandler struct {
	service mechinService.IMechinParameterService
}

// New constructs handler
func New(service mechinService.IMechinParameterService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) GetMechinParameters(appCtx *app.Context) *app.CostumeResponse {
	pg := pagination.Pagination(appCtx)

	data, err := h.service.GetAll(appCtx.Request.Context(), mechinService.ListQuery{
		Limit:  pg.Limit,
		Page:   pg.Page,
		Search: pg.Search,
		Status: appCtx.Query("status"),
	})
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

func (h *HTTPHandler) GetMechinParameterByID(appCtx *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(appCtx.Param("id"), 10, 64)
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
			Message:   "Not Found. Mechin parameter does not exist",
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

func (h *HTTPHandler) CreateMechinParameter(appCtx *app.Context) *app.CostumeResponse {
	var req models.CreateMechinParameterRequest

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
		Message:   "mechin parameter created successfully",
		Data:      data,
	}
}

func (h *HTTPHandler) UpdateMechinParameter(appCtx *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(appCtx.Param("id"), 10, 64)
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
			Message:   "Not Found. Mechin parameter does not exist",
		}
	}

	var req models.UpdateMechinParameterRequest
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
		Message:   "mechin parameter updated successfully",
		Data:      data,
	}
}

func (h *HTTPHandler) DeleteMechinParameter(appCtx *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(appCtx.Param("id"), 10, 64)
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
			Message:   "Not Found. Mechin parameter does not exist",
		}
	}

	if err := h.service.Delete(appCtx.Request.Context(), id); err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "mechin parameter deleted successfully",
	}
}
