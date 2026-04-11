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
	data, err := h.service.Create(appCtx.Request.Context(), req)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "delivery note created successfully",
		Data:      data,
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
	// ambil query param
	idParam := appCtx.Query("id")
	dnIDParam := appCtx.Query("dn_id")
	itemCode := appCtx.Query("item")

	// validasi basic
	if idParam == "" || dnIDParam == "" || itemCode == "" {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid qr parameters",
		}
	}

	// parse id
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}

	dnID, err := strconv.ParseInt(dnIDParam, 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid dn_id",
		}
	}

	// call service
	err = h.service.ScanAndUpdate(appCtx.Request.Context(), id, dnID, itemCode)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "item status updated to incoming",
		Data:      nil,
	}
}
