package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/delivery_note/models"
	deliveryNoteService "github.com/ganasa18/go-template/internal/delivery_note/service"
	"github.com/ganasa18/go-template/pkg/validator"
)

// HTTPHandler holds delivery note endpoints
type HTTPHandler struct {
	service deliveryNoteService.IDeliveryNoteService
}

// New constructs handler
func New(service deliveryNoteService.IDeliveryNoteService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) GetDeliveryNotes(appCtx *app.Context) *app.CostumeResponse {
	query := appCtx.Request.URL.Query()

	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	data, pagination, err := h.service.GetAll(appCtx.Request.Context(), page, limit)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data: map[string]interface{}{
			"data":       data,
			"pagination": pagination,
		},
	}
}

func (h *HTTPHandler) CreateDeliveryNote(appCtx *app.Context) *app.CostumeResponse {
	var req models.CreateDNRequest

	// bind request
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body",
		}
	}

	// validate
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": errs},
		}
	}

	// call service
	_, err := h.service.Create(appCtx.Request.Context(), req)
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
		Status:    http.StatusCreated,
		Message:   "delivery note created successfully",
		Data:      nil,
	}
}

func (h *HTTPHandler) GetDeliveryNoteByID(appCtx *app.Context) *app.CostumeResponse {
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
			Message:   "Not Found. Delivery note does not exist",
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

func (h *HTTPHandler) ScanDeliveryNoteItem(appCtx *app.Context) *app.CostumeResponse {
	var req models.QRPayload

	// bind request
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body",
		}
	}

	// validate
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": errs},
		}
	}

	response, err := h.service.ScanAndUpdate(appCtx.Request.Context(), req)
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
		Message:   "item status updated to " + response,
		Data:      nil,
	}
}

func (h *HTTPHandler) PreviewDN(appCtx *app.Context) *app.CostumeResponse {
	var req models.CreateDNRequest

	// bind request
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body",
		}
	}

	// validate
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": errs},
		}
	}

	data, err := h.service.PreviewDN(appCtx.Request.Context(), req)
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

func (h *HTTPHandler) PreviewItem(appCtx *app.Context) *app.CostumeResponse {
	var req models.PreviewDNItem

	// bind request
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body",
		}
	}

	// validate
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": errs},
		}
	}

	// 🔥 2. call service
	data, err := h.service.PreviewItem(appCtx.Request.Context(), req)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	// 🔥 3. response
	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "success",
		Data:      data,
	}
}
