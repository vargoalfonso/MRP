package handler

import (
	"net/http"

	actionModels "github.com/ganasa18/go-template/internal/action_ui/models"
	actionService "github.com/ganasa18/go-template/internal/action_ui/service"
	authModels "github.com/ganasa18/go-template/internal/auth/models"
	"github.com/ganasa18/go-template/internal/base/app"
)

type HTTPHandler struct {
	svc actionService.IService
}

func New(svc actionService.IService) *HTTPHandler {
	return &HTTPHandler{svc: svc}
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

	scannedBy := mustUsername(ctx)
	resp, idempotentHit, err := h.svc.CreateIncomingScan(ctx.Request.Context(), req, scannedBy)
	if err != nil {
		return app.NewError(ctx, err)
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

func mustUsername(ctx *app.Context) string {
	raw, exists := ctx.Get("claims")
	if !exists {
		return "system"
	}
	claims, ok := raw.(*authModels.Claims)
	if !ok || claims == nil {
		return "system"
	}
	if claims.UserID != "" {
		return claims.UserID
	}
	return "system"
}
