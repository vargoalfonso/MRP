package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/stockdaysparameter/models"
	stockdays "github.com/ganasa18/go-template/internal/stockdaysparameter/service"
)

// HTTPHandler holds stockdays parameter endpoints
type HTTPHandler struct {
	service stockdays.IStockdayService
}

// New constructs handler
func New(service stockdays.IStockdayService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) GetByID(appCtx *app.Context) *app.CostumeResponse {
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
			Message:   "Not Found. Stockdays parameter does not exist",
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

func (h *HTTPHandler) Create(appCtx *app.Context) *app.CostumeResponse {
	var req models.CreateStockdaysRequest

	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   err.Error(),
		}
	}

	data, err := h.service.Create(appCtx.Request.Context(), req)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "created",
		Data:      data,
	}
}

func (h *HTTPHandler) BulkCreate(appCtx *app.Context) *app.CostumeResponse {
	var req models.BulkCreateStockdaysRequest

	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   err.Error(),
		}
	}

	if err := h.service.BulkCreate(appCtx.Request.Context(), req); err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "bulk created",
	}
}

func (h *HTTPHandler) GetAll(appCtx *app.Context) *app.CostumeResponse {
	data, err := h.service.GetAll(appCtx.Request.Context())
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "success",
		Data:      data,
	}
}

func (h *HTTPHandler) Update(appCtx *app.Context) *app.CostumeResponse {
	id, _ := strconv.ParseInt(appCtx.Param("id"), 10, 64)

	var req models.UpdateStockdaysRequest
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			Status:  http.StatusBadRequest,
			Message: err.Error(),
		}
	}

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
