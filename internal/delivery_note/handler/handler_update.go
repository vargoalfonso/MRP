package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/delivery_note/models"
	deliveryNoteService "github.com/ganasa18/go-template/internal/delivery_note/service"
	"github.com/ganasa18/go-template/pkg/validator"
)

// UpdateDeliveryNote handles PUT /delivery-notes/:id (permission delivery_note:update).
// See models.UpdateDNRequest for what can be edited.
func (h *HTTPHandler) UpdateDeliveryNote(appCtx *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(appCtx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}

	var req models.UpdateDNRequest
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
		if errors.Is(err, deliveryNoteService.ErrDNNotFound) {
			return &app.CostumeResponse{
				RequestID: appCtx.APIReqID,
				Status:    http.StatusNotFound,
				Message:   "Not Found. Delivery note does not exist",
			}
		}
		status := http.StatusUnprocessableEntity
		if errors.Is(err, deliveryNoteService.ErrDNLocked) {
			status = http.StatusConflict
		}
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    status,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "delivery note updated successfully",
		Data:      data,
	}
}
