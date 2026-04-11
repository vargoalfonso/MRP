package handler

import (
	"net/http"
	"strconv"

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

func (h *HTTPHandler) BulkCreatePRLs(appCtx *app.Context) *app.CostumeResponse {
	var req models.BulkCreatePRLRequest
	if err := appCtx.ShouldBindJSON(&req); err != nil {
		return badRequest(appCtx, "invalid request body")
	}
	result, err := h.service.BulkCreatePRLs(appCtx.Request.Context(), req)
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

	result, serviceErr := h.service.ImportPRLs(c.Request.Context(), fileHeader.Filename, file)
	if serviceErr != nil {
		if appErr, ok := apperror.As(serviceErr); ok {
			c.JSON(appErr.HTTPStatus, gin.H{"request_id": c.GetHeader("X-Request-Id"), "status": appErr.HTTPStatus, "message": appErr.Message, "data": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"request_id": c.GetHeader("X-Request-Id"), "status": http.StatusInternalServerError, "message": "import prl failed", "data": nil})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"request_id": c.GetHeader("X-Request-Id"), "status": http.StatusCreated, "message": "prl import successful", "data": result})
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
	if status == models.PRLStatusApproved {
		result, err = h.service.ApprovePRLs(appCtx.Request.Context(), req)
	} else {
		result, err = h.service.RejectPRLs(appCtx.Request.Context(), req)
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
