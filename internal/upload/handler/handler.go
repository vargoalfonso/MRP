// Package handler exposes chunked upload HTTP endpoints.
//
//	POST   /api/v1/uploads/sessions                      — create session
//	GET    /api/v1/uploads/sessions/:session_id           — get session status (resume)
//	POST   /api/v1/uploads/sessions/:session_id/chunks/:index — upload one chunk (raw binary)
//	POST   /api/v1/uploads/sessions/:session_id/complete  — assemble & finalise
//	DELETE /api/v1/uploads/sessions/:session_id           — cancel & cleanup
package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/upload/models"
	"github.com/ganasa18/go-template/internal/upload/service"
	"github.com/ganasa18/go-template/pkg/validator"
	"github.com/google/uuid"
)

type HTTPHandler struct {
	svc service.IService
}

func New(svc service.IService) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

// parseSessionID extracts and validates the session UUID from the URL param.
func parseSessionID(ctx *app.Context) (uuid.UUID, error) {
	return uuid.Parse(ctx.Param("session_id"))
}

// ---------------------------------------------------------------------------
// POST /api/v1/uploads/sessions
// ---------------------------------------------------------------------------

// CreateSession initialises a new resumable upload.
//
//	Body: { item_id, asset_type, file_name, mime_type?, file_size, chunk_size, total_chunks }
//	Response: SessionResponse with missing_chunks (all indices at start)
func (h *HTTPHandler) CreateSession(ctx *app.Context) *app.CostumeResponse {
	var req models.CreateSessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}

	// Extract caller identity from JWT claims if present
	var createdBy *uuid.UUID
	if raw, ok := ctx.Get("user_uuid"); ok {
		if s, ok := raw.(string); ok {
			if uid, err := uuid.Parse(s); err == nil {
				createdBy = &uid
			}
		}
	}

	resp, err := h.svc.CreateSession(ctx.Request.Context(), req, createdBy)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusCreated, resp)
}

// ---------------------------------------------------------------------------
// GET /api/v1/uploads/sessions/:session_id
// ---------------------------------------------------------------------------

// GetSession returns current session state including uploaded/missing chunk indices.
// Frontend uses this to resume an interrupted upload.
func (h *HTTPHandler) GetSession(ctx *app.Context) *app.CostumeResponse {
	sid, err := parseSessionID(ctx)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid session_id"}
	}
	resp, err := h.svc.GetSession(ctx.Request.Context(), sid)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// POST /api/v1/uploads/sessions/:session_id/chunks/:index
// ---------------------------------------------------------------------------

// UploadChunk receives a single chunk (raw binary body).
//
//	Content-Type: application/octet-stream
//	X-Chunk-Checksum: <md5 hex>   (optional, for verification)
//
// The same chunk index can be re-uploaded (idempotent retry).
func (h *HTTPHandler) UploadChunk(ctx *app.Context) *app.CostumeResponse {
	sid, err := parseSessionID(ctx)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid session_id"}
	}

	index, err := strconv.Atoi(ctx.Param("index"))
	if err != nil || index < 0 {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid chunk index"}
	}

	clientChecksum := ctx.GetHeader("X-Chunk-Checksum")

	resp, err := h.svc.UploadChunk(ctx.Request.Context(), sid, index, ctx.Request.Body, clientChecksum)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// POST /api/v1/uploads/sessions/:session_id/complete
// ---------------------------------------------------------------------------

// Complete signals that all chunks have been uploaded.
// Backend assembles the file, creates the item_asset record, and returns the final URL.
func (h *HTTPHandler) Complete(ctx *app.Context) *app.CostumeResponse {
	sid, err := parseSessionID(ctx)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid session_id"}
	}

	var req models.CompleteSessionRequest
	_ = ctx.ShouldBindJSON(&req) // optional body

	resp, err := h.svc.Complete(ctx.Request.Context(), sid, req)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// DELETE /api/v1/uploads/sessions/:session_id
// ---------------------------------------------------------------------------

// Cancel removes temporary chunk files and marks the session cancelled.
func (h *HTTPHandler) Cancel(ctx *app.Context) *app.CostumeResponse {
	sid, err := parseSessionID(ctx)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid session_id"}
	}
	if err := h.svc.Cancel(ctx.Request.Context(), sid); err != nil {
		return app.NewError(ctx, err)
	}
	return app.NewSuccess(ctx, http.StatusOK, map[string]string{"status": "cancelled"})
}
