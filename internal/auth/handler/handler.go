// Package handler exposes the auth HTTP endpoints.
//
// POST /auth/register — public
// POST /auth/login    — public
// POST /auth/refresh  — stateful mode only
// POST /auth/logout   — stateful mode only (requires JWTMiddleware)
package handler

import (
	"log/slog"
	"net/http"

	"github.com/ganasa18/go-template/internal/auth/models"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/pkg/logger"
	"github.com/ganasa18/go-template/pkg/secret"
	"github.com/ganasa18/go-template/pkg/validator"
)

// HTTPHandler holds all auth endpoint implementations.
type HTTPHandler struct {
	auth authService.Authenticator
}

// New constructs an HTTPHandler.
func New(auth authService.Authenticator) *HTTPHandler {
	return &HTTPHandler{auth: auth}
}

// Register handles POST /auth/register.
func (h *HTTPHandler) Register(appCtx *app.Context) *app.CostumeResponse {
	var req models.RegisterRequest
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

	user, err := h.auth.Register(appCtx.Request.Context(), req)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "registration successful",
		Data: models.RegisterResponse{
			ID:        user.UUID,
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	}
}

// Login handles POST /auth/login.
func (h *HTTPHandler) Login(appCtx *app.Context) *app.CostumeResponse {
	var req models.LoginRequest
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

	pair, err := h.auth.Login(appCtx.Request.Context(), req)
	if err != nil {
		logger.FromContext(appCtx.Request.Context()).Error("login failed",
			slog.String("email", secret.MaskEmail(req.Email)),
			slog.Any("error", err),
		)
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      pair,
	}
}

// Refresh handles POST /auth/refresh (stateful mode only).
func (h *HTTPHandler) Refresh(appCtx *app.Context) *app.CostumeResponse {
	var req models.RefreshRequest
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

	pair, err := h.auth.RefreshTokens(appCtx.Request.Context(), req.RefreshToken)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      pair,
	}
}

// Logout handles POST /auth/logout (stateful mode only, requires JWTMiddleware).
func (h *HTTPHandler) Logout(appCtx *app.Context) *app.CostumeResponse {
	claimsRaw, exists := appCtx.Get("claims")
	if !exists {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusUnauthorized,
			Message:   "not authenticated",
		}
	}

	jwtClaims, ok := claimsRaw.(*models.Claims)
	if !ok {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusUnauthorized,
			Message:   "invalid claims",
		}
	}

	if err := h.auth.RevokeToken(appCtx.Request.Context(), jwtClaims.ID); err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "logged out successfully",
	}
}
