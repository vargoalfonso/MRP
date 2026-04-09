package handler

import (
	"net/http"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/role/models"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	"github.com/ganasa18/go-template/pkg/validator"
)

// HTTPHandler holds role endpoints
type HTTPHandler struct {
	service roleService.IRoleService
}

// New constructs handler
func New(service roleService.IRoleService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

// GetRoles handles GET /roles
func (h *HTTPHandler) GetRoles(appCtx *app.Context) *app.CostumeResponse {
	roles, err := h.service.GetAll(appCtx.Request.Context())
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      roles,
	}
}

// CreateRole handles POST /roles
func (h *HTTPHandler) CreateRole(appCtx *app.Context) *app.CostumeResponse {
	var req models.CreateRoleRequest

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

	role, err := h.service.Create(appCtx.Request.Context(), req)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "role created successfully",
		Data:      role,
	}
}
