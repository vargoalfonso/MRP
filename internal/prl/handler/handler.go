package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	authModels "github.com/ganasa18/go-template/internal/auth/models"
	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/prl/models"
	prlService "github.com/ganasa18/go-template/internal/prl/service"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/validator"
	"github.com/gin-gonic/gin"
)

type HTTPHandler struct {
	service prlService.Service
}

func New(service prlService.Service) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) CreateUniqBOM(appCtx *app.Context) *app.CostumeResponse {
	var req models.CreateUniqBOMRequest
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return badRequest(appCtx, "invalid request body")
	}
	if errs := validator.Validate(req); errs != nil {
		return validationError(appCtx, errs)
	}
	item, err := h.service.CreateUniqBOM(appCtx.Request.Context(), req)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusCreated, Message: "uniq bom created successfully", Data: item}
}

func (h *HTTPHandler) ListUniqBOMs(appCtx *app.Context) *app.CostumeResponse {
	var query models.ListUniqBOMQuery
	if err := appCtx.ShouldBindQuery(&query); err != nil {
		return badRequest(appCtx, "invalid query params")
	}
	result, err := h.service.ListUniqBOMs(appCtx.Request.Context(), query)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return ok(appCtx, result)
}

func (h *HTTPHandler) GetUniqBOM(appCtx *app.Context) *app.CostumeResponse {
	item, err := h.service.GetUniqBOMByUUID(appCtx.Request.Context(), appCtx.Param("id"))
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return ok(appCtx, item)
}

func (h *HTTPHandler) UpdateUniqBOM(appCtx *app.Context) *app.CostumeResponse {
	var req models.UpdateUniqBOMRequest
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return badRequest(appCtx, "invalid request body")
	}
	if errs := validator.Validate(req); errs != nil {
		return validationError(appCtx, errs)
	}
	item, err := h.service.UpdateUniqBOM(appCtx.Request.Context(), appCtx.Param("id"), req)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: "uniq bom updated successfully", Data: item}
}

func (h *HTTPHandler) DeleteUniqBOM(appCtx *app.Context) *app.CostumeResponse {
	if err := h.service.DeleteUniqBOM(appCtx.Request.Context(), appCtx.Param("id")); err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: "uniq bom deleted successfully"}
}

// CreatePRL supports:
//  1. Single create payload (no wrapper): {customer_uuid, uniq_code, product_model, part_name, part_number, forecast_period, quantity}
//  2. Bulk create payload (wrapper): {"entries":[...]}
func (h *HTTPHandler) CreatePRL(appCtx *app.Context) *app.CostumeResponse {
	body, err := appCtx.GetRawData()
	if err != nil {
		return badRequest(appCtx, "invalid request body")
	}

	// Backward compatibility: if request has "entries", treat it as bulk create.
	var bulkProbe struct {
		Entries []models.CreatePRLEntryRequest `json:"entries"`
	}
	if unmarshalErr := json.Unmarshal(body, &bulkProbe); unmarshalErr == nil && len(bulkProbe.Entries) > 0 {
		if errs := validator.Validate(models.BulkCreatePRLRequest{Entries: bulkProbe.Entries}); errs != nil {
			return validationError(appCtx, errs)
		}
		userID, claimErr := mustUserID(appCtx)
		if claimErr != nil {
			return app.NewError(appCtx, claimErr)
		}
		result, svcErr := h.service.BulkCreatePRLs(appCtx.Request.Context(), models.BulkCreatePRLRequest{Entries: bulkProbe.Entries}, userID)
		if svcErr != nil {
			return app.NewError(appCtx, svcErr)
		}
		return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusCreated, Message: "prls created successfully", Data: result}
	}

	var req models.CreatePRLRequest
	if unmarshalErr := json.Unmarshal(body, &req); unmarshalErr != nil {
		return badRequest(appCtx, "invalid request body")
	}
	if errs := validator.Validate(req); errs != nil {
		return validationError(appCtx, errs)
	}
	userID, claimErr := mustUserID(appCtx)
	if claimErr != nil {
		return app.NewError(appCtx, claimErr)
	}
	item, svcErr := h.service.CreatePRL(appCtx.Request.Context(), req, userID)
	if svcErr != nil {
		return app.NewError(appCtx, svcErr)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusCreated, Message: "prl created successfully", Data: item}
}

func (h *HTTPHandler) BulkCreatePRLs(appCtx *app.Context) *app.CostumeResponse {
	var req models.BulkCreatePRLRequest
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return badRequest(appCtx, "invalid request body")
	}
	userID, claimErr := mustUserID(appCtx)
	if claimErr != nil {
		return app.NewError(appCtx, claimErr)
	}
	result, err := h.service.BulkCreatePRLs(appCtx.Request.Context(), req, userID)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusCreated, Message: "prls created successfully", Data: result}
}

func (h *HTTPHandler) ListPRLs(appCtx *app.Context) *app.CostumeResponse {
	var query models.ListPRLQuery
	if err := appCtx.ShouldBindQuery(&query); err != nil {
		return badRequest(appCtx, "invalid query params")
	}
	result, err := h.service.ListPRLs(appCtx.Request.Context(), query)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return ok(appCtx, result)
}

func (h *HTTPHandler) GetPRL(appCtx *app.Context) *app.CostumeResponse {
	item, err := h.service.GetPRLByUUID(appCtx.Request.Context(), appCtx.Param("id"))
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return ok(appCtx, item)
}

func (h *HTTPHandler) GetPRLDetail(appCtx *app.Context) *app.CostumeResponse {
	item, err := h.service.GetPRLDetail(appCtx.Request.Context(), appCtx.Param("id"))
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return ok(appCtx, item)
}

func (h *HTTPHandler) UpdatePRL(appCtx *app.Context) *app.CostumeResponse {
	var req models.UpdatePRLRequest
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return badRequest(appCtx, "invalid request body")
	}
	if errs := validator.Validate(req); errs != nil {
		return validationError(appCtx, errs)
	}
	item, err := h.service.UpdatePRL(appCtx.Request.Context(), appCtx.Param("id"), req)
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: "prl updated successfully", Data: item}
}

func (h *HTTPHandler) DeletePRL(appCtx *app.Context) *app.CostumeResponse {
	if err := h.service.DeletePRL(appCtx.Request.Context(), appCtx.Param("id")); err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: "prl deleted successfully"}
}

func (h *HTTPHandler) ApprovePRLs(appCtx *app.Context) *app.CostumeResponse {
	return h.bulkStatusAction(appCtx, models.PRLStatusApproved)
}

func (h *HTTPHandler) RejectPRLs(appCtx *app.Context) *app.CostumeResponse {
	return h.bulkStatusAction(appCtx, models.PRLStatusRejected)
}

func (h *HTTPHandler) ListCustomerLookups(appCtx *app.Context) *app.CostumeResponse {
	items, err := h.service.ListCustomerLookups(appCtx.Request.Context(), appCtx.Query("search"))
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return ok(appCtx, items)
}

func (h *HTTPHandler) ListForecastPeriods(appCtx *app.Context) *app.CostumeResponse {
	year, _ := strconv.Atoi(appCtx.DefaultQuery("year", "0"))
	items := h.service.ListForecastPeriodOptions(year)
	return ok(appCtx, items)
}

func (h *HTTPHandler) ImportPRLs(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"request_id": c.GetHeader("X-Request-Id"), "status": http.StatusBadRequest, "message": "file is required", "data": nil})
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"request_id": c.GetHeader("X-Request-Id"), "status": http.StatusBadRequest, "message": "failed to open uploaded file", "data": nil})
		return
	}
	defer func() { _ = file.Close() }()

	appCtx := app.NewContext(c)
	userID, claimErr := mustUserID(appCtx)
	if claimErr != nil {
		if appErr, ok := apperror.As(claimErr); ok {
			c.JSON(appErr.HTTPStatus, gin.H{"request_id": appCtx.APIReqID, "status": appErr.HTTPStatus, "message": appErr.Message, "data": nil})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"request_id": appCtx.APIReqID, "status": http.StatusUnauthorized, "message": "not authenticated", "data": nil})
		return
	}
	result, serviceErr := h.service.ImportPRLs(c.Request.Context(), fileHeader.Filename, file, userID)
	if serviceErr != nil {
		if appErr, ok := apperror.As(serviceErr); ok {
			c.JSON(appErr.HTTPStatus, gin.H{"request_id": appCtx.APIReqID, "status": appErr.HTTPStatus, "message": appErr.Message, "data": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"request_id": appCtx.APIReqID, "status": http.StatusInternalServerError, "message": "import prl failed", "data": nil})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"request_id": appCtx.APIReqID, "status": http.StatusCreated, "message": "prl import successful", "data": result})
}

func (h *HTTPHandler) ExportPRLs(c *gin.Context) {
	query := models.ListPRLQuery{
		Search:         c.Query("search"),
		Status:         c.Query("status"),
		ForecastPeriod: c.Query("forecast_period"),
		CustomerUUID:   c.Query("customer_uuid"),
		UniqCode:       c.Query("uniq_code"),
	}
	filename, content, err := h.service.ExportPRLs(c.Request.Context(), query)
	if err != nil {
		if appErr, ok := apperror.As(err); ok {
			c.JSON(appErr.HTTPStatus, gin.H{"request_id": c.GetHeader("X-Request-Id"), "status": appErr.HTTPStatus, "message": appErr.Message, "data": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"request_id": c.GetHeader("X-Request-Id"), "status": http.StatusInternalServerError, "message": "export prl failed", "data": nil})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", content)
}

func (h *HTTPHandler) bulkStatusAction(appCtx *app.Context, status string) *app.CostumeResponse {
	var req models.BulkStatusActionRequest
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return badRequest(appCtx, "invalid request body")
	}
	var (
		result *models.BulkStatusActionResponse
		err    error
	)
	userID, roles, claimErr := mustApprovalActor(appCtx)
	if claimErr != nil {
		return app.NewError(appCtx, claimErr)
	}
	if status == models.PRLStatusApproved {
		result, err = h.service.ApprovePRLs(appCtx.Request.Context(), req, userID, roles)
	} else {
		result, err = h.service.RejectPRLs(appCtx.Request.Context(), req, userID, roles)
	}
	if err != nil {
		return app.NewError(appCtx, err)
	}
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: "prl status updated successfully", Data: result}
}

func ok(appCtx *app.Context, data interface{}) *app.CostumeResponse {
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: data}
}

func badRequest(appCtx *app.Context, message string) *app.CostumeResponse {
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusBadRequest, Message: message}
}

func validationError(appCtx *app.Context, errs interface{}) *app.CostumeResponse {
	return &app.CostumeResponse{RequestID: appCtx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
}

func mustUserID(appCtx *app.Context) (string, error) {
	claimsRaw, ok := appCtx.Get("claims")
	if !ok {
		return "", apperror.Unauthorized("not authenticated")
	}
	claims, ok := claimsRaw.(*authModels.Claims)
	if !ok || claims == nil || claims.UserID == "" {
		return "", apperror.Unauthorized("not authenticated")
	}
	return claims.UserID, nil
}

func mustApprovalActor(appCtx *app.Context) (string, []string, error) {
	claimsRaw, ok := appCtx.Get("claims")
	if !ok {
		return "", nil, apperror.Unauthorized("not authenticated")
	}
	claims, ok := claimsRaw.(*authModels.Claims)
	if !ok || claims == nil || claims.UserID == "" {
		return "", nil, apperror.Unauthorized("not authenticated")
	}
	return claims.UserID, claims.Roles, nil
}
