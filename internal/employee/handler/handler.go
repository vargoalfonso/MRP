package handler

import (
	"net/http"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/employee/models"
	employeeService "github.com/ganasa18/go-template/internal/employee/service"
)

// HTTPHandler exposes employee endpoints.
type HTTPHandler struct {
	EmployeeService employeeService.Service
}

// NewHTTPHandler constructs an HTTPHandler.
func NewHTTPHandler(svc employeeService.Service) *HTTPHandler {
	return &HTTPHandler{EmployeeService: svc}
}

func (h *HTTPHandler) GetAllDataEmployee(appCtx *app.Context) *app.CostumeResponse {
	resp, statusCode, err := h.EmployeeService.GetAllDataEmployee(appCtx.Request.Context())
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    statusCode,
		Message:   http.StatusText(statusCode),
		Data:      resp,
	}
}

func (h *HTTPHandler) GetDataEmployeeByID(appCtx *app.Context) *app.CostumeResponse {
	id := appCtx.Query("id")
	resp, statusCode, err := h.EmployeeService.GetDataEmployeeByID(appCtx.Request.Context(), id)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    statusCode,
		Message:   http.StatusText(statusCode),
		Data:      resp,
	}
}

func (h *HTTPHandler) Register(appCtx *app.Context) *app.CostumeResponse {
	req := models.EmployeeReq{}
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body",
		}
	}
	statusCode, err := h.EmployeeService.Register(appCtx.Request.Context(), req)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    statusCode,
		Message:   "employee registered successfully",
	}
}

func (h *HTTPHandler) UpdateDataEmployee(appCtx *app.Context) *app.CostumeResponse {
	id := appCtx.Query("id")
	req := models.EmployeeReq{}
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body",
		}
	}
	req.EmployeeID = id
	statusCode, err := h.EmployeeService.UpdateDataEmployee(appCtx.Request.Context(), req)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    statusCode,
		Message:   "employee updated successfully",
		Data:      req,
	}
}
