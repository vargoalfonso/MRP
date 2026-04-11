package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/employee/models"
	employeeService "github.com/ganasa18/go-template/internal/employee/service"
	"github.com/ganasa18/go-template/pkg/validator"
)

// HTTPHandler holds employee endpoints
type HTTPHandler struct {
	service employeeService.IEmployeeService
}

// New constructs handler
func New(service employeeService.IEmployeeService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) GetEmployees(appCtx *app.Context) *app.CostumeResponse {
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

func (h *HTTPHandler) CreateEmployee(appCtx *app.Context) *app.CostumeResponse {
	var req models.CreateEmployeeRequest

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
		Message:   "employee created successfully",
		Data:      data,
	}
}

func (h *HTTPHandler) GetEmployeeByID(appCtx *app.Context) *app.CostumeResponse {
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
			Message:   "Not Found. Employee does not exist",
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

func (h *HTTPHandler) UpdateEmployee(appCtx *app.Context) *app.CostumeResponse {
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
			Message:   "Not Found. Employee does not exist",
		}
	}

	var req models.UpdateEmployeeRequest
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
		Message:   "employee updated successfully",
		Data:      data,
	}
}

func (h *HTTPHandler) DeleteEmployee(appCtx *app.Context) *app.CostumeResponse {
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
			Message:   "Not Found. Employee does not exist",
		}
	}

	err = h.service.Delete(appCtx.Request.Context(), id)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "employee deleted successfully",
	}
}
