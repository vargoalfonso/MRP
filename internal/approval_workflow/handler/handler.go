package handler

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/ganasa18/go-template/internal/approval_workflow/models"
	approvalWorkflowService "github.com/ganasa18/go-template/internal/approval_workflow/service"
	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/pkg/validator"
)

// HTTPHandler holds approval workflow endpoints
type HTTPHandler struct {
	service approvalWorkflowService.IApprovalWorkflowService
}

// New constructs handler
func New(service approvalWorkflowService.IApprovalWorkflowService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) GetApprovalWorkflows(appCtx *app.Context) *app.CostumeResponse {
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

func (h *HTTPHandler) CreateApprovalWorkflow(appCtx *app.Context) *app.CostumeResponse {
	var req models.CreateApprovalWorkflowRequest

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

	data, err := h.service.Create(appCtx.Request.Context(), req)
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
		Message:   "approval workflow created successfully",
		Data:      data,
	}
}

func (h *HTTPHandler) GetApprovalWorkflowByID(appCtx *app.Context) *app.CostumeResponse {
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
			Message:   "Not Found. Approval workflow does not exist",
		}
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

func (h *HTTPHandler) UpdateApprovalWorkflow(appCtx *app.Context) *app.CostumeResponse {
	idParam := appCtx.Param("id")

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}

	_, err = h.service.GetByID(appCtx.Request.Context(), id)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusNotFound,
			Message:   "Not Found. Approval workflow does not exist",
		}
	}

	var req models.UpdateApprovalWorkflowRequest
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
		Message:   "approval workflow updated successfully",
		Data:      data,
	}
}

func (h *HTTPHandler) DeleteApprovalWorkflow(appCtx *app.Context) *app.CostumeResponse {
	idParam := appCtx.Param("id")

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}

	_, err = h.service.GetByID(appCtx.Request.Context(), id)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusNotFound,
			Message:   "Not Found. Approval workflow does not exist",
		}
	}

	err = h.service.Delete(appCtx.Request.Context(), id)
	if err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "approval workflow deleted successfully",
	}
}

func (h *HTTPHandler) Approve(appCtx *app.Context) *app.CostumeResponse {

	// ==============================
	// 🔥 GET CLAIMS
	// ==============================
	claimsRaw, exists := appCtx.Get("claims")
	if !exists || claimsRaw == nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusUnauthorized,
			Message:   "not authenticated",
		}
	}

	// ==============================
	// 🔥 EXTRACT ROLES (NO TYPE ISSUE)
	// ==============================
	roles, err := extractRoles(claimsRaw)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusUnauthorized,
			Message:   err.Error(),
		}
	}

	// ==============================
	// 🔥 PARAM
	// ==============================
	idParam := appCtx.Param("id")
	instanceID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}

	// ==============================
	// 🔥 SERVICE
	// ==============================
	if err := h.service.Approve(appCtx.Request.Context(), instanceID, roles); err != nil {
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
		Message:   "approved successfully",
	}
}

func (h *HTTPHandler) Reject(appCtx *app.Context) *app.CostumeResponse {

	// ==============================
	// 🔥 CLAIMS
	// ==============================
	claimsRaw, exists := appCtx.Get("claims")
	if !exists || claimsRaw == nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusUnauthorized,
			Message:   "not authenticated",
		}
	}

	// ==============================
	// 🔥 EXTRACT ROLES (NO TYPE ISSUE)
	// ==============================
	roles, err := extractRoles(claimsRaw)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusUnauthorized,
			Message:   err.Error(),
		}
	}

	// ==============================
	// 🔥 PARAM
	// ==============================
	idParam := appCtx.Param("id")
	instanceID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}

	// ==============================
	// 🔥 BODY
	// ==============================
	var req models.RejectRequest
	if err := appCtx.BindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body",
		}
	}

	if strings.TrimSpace(req.Note) == "" {
		return &app.CostumeResponse{
			RequestID: appCtx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "note is required",
		}
	}

	// ==============================
	// 🔥 SERVICE
	// ==============================
	if err := h.service.Reject(appCtx.Request.Context(), instanceID, roles, req.Note); err != nil {
		return app.NewError(appCtx, err)
	}

	return &app.CostumeResponse{
		RequestID: appCtx.APIReqID,
		Status:    http.StatusOK,
		Message:   "rejected successfully",
	}
}

func extractRoles(claimsRaw interface{}) ([]string, error) {

	// ==============================
	// 🔥 CASE 1: STRUCT (REFLECT)
	// ==============================
	val := reflect.ValueOf(claimsRaw)

	// handle pointer
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() == reflect.Struct {
		field := val.FieldByName("Roles")

		if field.IsValid() && field.Kind() == reflect.Slice {

			var roles []string
			for i := 0; i < field.Len(); i++ {
				roles = append(roles, field.Index(i).String())
			}

			if len(roles) > 0 {
				return roles, nil
			}
		}
	}

	// ==============================
	// 🔥 CASE 2: MAP (fallback)
	// ==============================
	if claimsMap, ok := claimsRaw.(map[string]interface{}); ok {

		if rolesInterface, ok := claimsMap["roles"]; ok {

			var roles []string

			for _, r := range rolesInterface.([]interface{}) {
				roles = append(roles, fmt.Sprintf("%v", r))
			}

			return roles, nil
		}
	}

	return nil, fmt.Errorf("roles not found in token")
}
