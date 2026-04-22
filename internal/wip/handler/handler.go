package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/wip/models"
	wipService "github.com/ganasa18/go-template/internal/wip/service"
)

type HTTPHandler struct {
	service wipService.IWIPService
}

func New(service wipService.IWIPService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) GetAll(appCtx *app.Context) *app.CostumeResponse {

	pageStr := appCtx.Query("page")
	limitStr := appCtx.Query("limit")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	data, total, err := h.service.GetAll(appCtx.Request.Context(), page, limit)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusInternalServerError,
			Message:   err.Error(),
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "Success",
		Data: map[string]interface{}{
			"data":  data,
			"total": total,
			"page":  page,
			"limit": limit,
		},
	}
}

func (h *HTTPHandler) GetByID(appCtx *app.Context) *app.CostumeResponse {

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
			Message:   "WIP not found",
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "Success",
		Data:      data,
	}
}

func (h *HTTPHandler) Create(appCtx *app.Context) *app.CostumeResponse {

	var req models.CreateWIPRequest

	if err := appCtx.Bind(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   err.Error(),
		}
	}

	data, err := h.service.Create(appCtx.Request.Context(), req)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusInternalServerError,
			Message:   err.Error(),
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "WIP created",
		Data:      data,
	}
}

func (h *HTTPHandler) Update(appCtx *app.Context) *app.CostumeResponse {

	id, err := strconv.ParseInt(appCtx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}

	var req models.UpdateWIPRequest
	if err := appCtx.Bind(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   err.Error(),
		}
	}

	data, err := h.service.Update(appCtx.Request.Context(), id, req)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusInternalServerError,
			Message:   err.Error(),
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "WIP updated",
		Data:      data,
	}
}

func (h *HTTPHandler) Delete(appCtx *app.Context) *app.CostumeResponse {

	id, err := strconv.ParseInt(appCtx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}

	if err := h.service.Delete(appCtx.Request.Context(), id); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusInternalServerError,
			Message:   err.Error(),
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "WIP deleted",
	}
}

func (h *HTTPHandler) GetItems(appCtx *app.Context) *app.CostumeResponse {

	wipID, err := strconv.ParseInt(appCtx.Query("wip_id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid wip_id",
		}
	}

	data, err := h.service.GetItems(appCtx.Request.Context(), wipID)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusInternalServerError,
			Message:   err.Error(),
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "Success",
		Data:      data,
	}
}

func (h *HTTPHandler) Scan(appCtx *app.Context) *app.CostumeResponse {

	var req models.ScanRequest

	if err := appCtx.Bind(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   err.Error(),
		}
	}

	if err := h.service.Scan(appCtx.Request.Context(), req); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   err.Error(),
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "Scan success",
	}
}
