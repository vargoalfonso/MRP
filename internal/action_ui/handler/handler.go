package handler

import (
	"net/http"

	"github.com/ganasa18/go-template/internal/action_ui/dto"
	actionModels "github.com/ganasa18/go-template/internal/action_ui/models"
	actionService "github.com/ganasa18/go-template/internal/action_ui/service"
	"github.com/ganasa18/go-template/internal/base/app"
	userPkg "github.com/ganasa18/go-template/pkg/auth"
)

type HTTPHandler struct {
	svc actionService.IService
}

func New(svc actionService.IService) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

// GET /api/v1/action-ui/incoming/lookup?packing_number=KB-123456&item_uniq_code=UQ-123456
// Called when QR is scanned — auto-fills PO Number, Supplier, DN Number, Type on the form.
func (h *HTTPHandler) LookupByPackingNumber(ctx *app.Context) *app.CostumeResponse {
	packingNumber := ctx.Query("packing_number")
	if packingNumber == "" {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "packing_number is required",
		}
	}
	itemUniqCode := ctx.Query("item_uniq_code")

	result, err := h.svc.LookupByPackingNumber(ctx.Request.Context(), packingNumber, itemUniqCode)
	if err != nil {
		return app.NewError(ctx, err)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

// POST /api/v1/action-ui/incoming/scans
func (h *HTTPHandler) CreateIncomingScan(ctx *app.Context) *app.CostumeResponse {
	var req actionModels.IncomingScanRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}

	userCtx := userPkg.MustExtractUserContext(ctx)
	resp, idempotentHit, err := h.svc.CreateIncomingScan(ctx.Request.Context(), req, userCtx.UserID)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	status := http.StatusCreated
	message := "Created"
	if idempotentHit {
		status = http.StatusOK
		message = http.StatusText(http.StatusOK)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    status,
		Message:   message,
		Data:      resp,
	}
}

func (h *HTTPHandler) ScanContext(ctx *app.Context) *app.CostumeResponse {
	wo := ctx.Query("wo")
	if wo == "" {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "wo is required",
		}
	}

	result, err := h.svc.ScanContext(ctx.Request.Context(), wo)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

func (h *HTTPHandler) ScanIn(ctx *app.Context) *app.CostumeResponse {
	var req dto.ScanInRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}

	err := h.svc.ScanIn(ctx.Request.Context(), req)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "Scan In Success",
	}
}

func (h *HTTPHandler) ScanOut(ctx *app.Context) *app.CostumeResponse {
	var req dto.ScanOutRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}

	err := h.svc.ScanOut(ctx.Request.Context(), req)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "Scan Out Success",
	}
}

func (h *HTTPHandler) QCSubmit(ctx *app.Context) *app.CostumeResponse {
	var req dto.QCSubmitRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}

	err := h.svc.QCSubmit(ctx.Request.Context(), req)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "QC Submit Success",
	}
}
