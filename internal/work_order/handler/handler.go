package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/base/app"
	woModels "github.com/ganasa18/go-template/internal/work_order/models"
	woService "github.com/ganasa18/go-template/internal/work_order/service"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/approval"
	userPkg "github.com/ganasa18/go-template/pkg/auth"
	"github.com/ganasa18/go-template/pkg/pagination"
	"github.com/ganasa18/go-template/pkg/validator"
	"github.com/gin-gonic/gin"
)

type HTTPHandler struct {
	svc woService.IService
}

func New(svc woService.IService) *HTTPHandler { return &HTTPHandler{svc: svc} }

func (h *HTTPHandler) ListBulkSourceDocuments(ctx *app.Context) *app.CostumeResponse {
	limit := 20
	if v := strings.TrimSpace(ctx.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	data, err := h.svc.ListBulkSourceDocuments(ctx.Request.Context(), ctx.Query("document_type"), ctx.Query("q"), ctx.Query("target_date"), limit)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: data}
}

func (h *HTTPHandler) ListBulkSourceDocumentItems(ctx *app.Context) *app.CostumeResponse {
	data, err := h.svc.ListBulkSourceDocumentItems(ctx.Request.Context(), ctx.Query("document_id"), ctx.Query("document_type"))
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: data}
}

func (h *HTTPHandler) ListBulkWorkOrders(ctx *app.Context) *app.CostumeResponse {
	p := pagination.WorkOrderPagination(ctx)
	data, err := h.svc.ListBulk(ctx.Request.Context(), p)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: data}
}

func (h *HTTPHandler) CreateBulkWorkOrder(ctx *app.Context) *app.CostumeResponse {
	var req woModels.CreateBulkWorkOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body: " + err.Error()}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	data, err := h.svc.CreateBulk(ctx.Request.Context(), req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusCreated, Message: "Created", Data: data}
}

func (h *HTTPHandler) GetBulkWorkOrderSummary(ctx *app.Context) *app.CostumeResponse {
	data, err := h.svc.GetBulkSummary(ctx.Request.Context())
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: data}
}

func (h *HTTPHandler) GetBulkWorkOrderDetail(ctx *app.Context) *app.CostumeResponse {
	data, err := h.svc.GetBulkDetail(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: data}
}

func (h *HTTPHandler) ApprovalBulkWorkOrder(ctx *app.Context) *app.CostumeResponse {
	var req woModels.WorkOrderApprovalRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body: " + err.Error()}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	data, err := h.svc.ApprovalBulk(ctx.Request.Context(), ctx.Param("id"), req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: data}
}

func (h *HTTPHandler) BulkApprovalBulkWorkOrder(ctx *app.Context) *app.CostumeResponse {
	var req woModels.BulkWorkOrderApprovalRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body: " + err.Error()}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	data, err := h.svc.BulkApprovalBulk(ctx.Request.Context(), req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: data}
}

// ListRMProcessingWorkOrders returns paginated RM processing work order list.
// GET /api/v1/working-order/rm-processing/work-orders
func (h *HTTPHandler) ListRMProcessingWorkOrders(ctx *app.Context) *app.CostumeResponse {
	p := pagination.WorkOrderPagination(ctx)
	data, err := h.svc.ListRMProcessing(ctx.Request.Context(), p)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// CreateRMProcessingWorkOrder creates an RM processing work order.
// POST /api/v1/working-order/rm-processing/work-orders
func (h *HTTPHandler) CreateRMProcessingWorkOrder(ctx *app.Context) *app.CostumeResponse {
	var req woModels.CreateRMProcessingWorkOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": errs},
		}
	}

	userCtx := userPkg.MustExtractUserContext(ctx)
	data, err := h.svc.CreateRMProcessing(ctx.Request.Context(), req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "Created",
		Data:      data,
	}
}

// ListWorkOrders returns paginated work order list.
// GET /api/v1/working-order/work-orders
func (h *HTTPHandler) ListWorkOrders(ctx *app.Context) *app.CostumeResponse {
	p := pagination.WorkOrderPagination(ctx)
	data, err := h.svc.List(ctx.Request.Context(), p)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// CreateWorkOrder creates a work order header and items (kanbans).
// POST /api/v1/working-order/work-orders
func (h *HTTPHandler) CreateWorkOrder(ctx *app.Context) *app.CostumeResponse {
	var req woModels.CreateWorkOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": errs},
		}
	}

	userCtx := userPkg.MustExtractUserContext(ctx)
	data, err := h.svc.Create(ctx.Request.Context(), req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "Created",
		Data:      data,
	}
}

// PreviewWorkOrder returns computed wo_number + kanban lines without inserting.
// POST /api/v1/working-order/work-orders/preview
func (h *HTTPHandler) PreviewWorkOrder(ctx *app.Context) *app.CostumeResponse {
	var req woModels.CreateWorkOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": errs},
		}
	}

	data, err := h.svc.Preview(ctx.Request.Context(), req)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// GetWorkOrderSummary returns board summary counters.
// GET /api/v1/working-order/work-orders/summary
func (h *HTTPHandler) GetWorkOrderSummary(ctx *app.Context) *app.CostumeResponse {
	data, err := h.svc.GetSummary(ctx.Request.Context())
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// GetRMProcessingWorkOrderSummary returns RM processing summary counters.
// GET /api/v1/working-order/rm-processing/work-orders/summary
func (h *HTTPHandler) GetRMProcessingWorkOrderSummary(ctx *app.Context) *app.CostumeResponse {
	data, err := h.svc.GetRMProcessingSummary(ctx.Request.Context())
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// GetWorkOrderDetail returns WO header + items.
// GET /api/v1/working-order/work-orders/:id
func (h *HTTPHandler) GetWorkOrderDetail(ctx *app.Context) *app.CostumeResponse {
	woUUID := ctx.Param("id")
	data, err := h.svc.GetDetail(ctx.Request.Context(), woUUID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// GetRMProcessingWorkOrderDetail returns RM processing WO header + detail.
// GET /api/v1/working-order/rm-processing/work-orders/:id
func (h *HTTPHandler) GetRMProcessingWorkOrderDetail(ctx *app.Context) *app.CostumeResponse {
	woUUID := ctx.Param("id")
	data, err := h.svc.GetRMProcessingDetail(ctx.Request.Context(), woUUID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// Approval approves or rejects a work order.
// POST /api/v1/working-order/work-orders/:id/approval
func (h *HTTPHandler) Approval(ctx *app.Context) *app.CostumeResponse {
	woUUID := ctx.Param("id")

	var req woModels.WorkOrderApprovalRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": errs},
		}
	}

	userCtx := userPkg.MustExtractUserContext(ctx)
	data, err := h.svc.Approval(ctx.Request.Context(), woUUID, req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// ApprovalRMProcessing approves or rejects an RM processing work order.
// POST /api/v1/working-order/rm-processing/work-orders/:id/approval
func (h *HTTPHandler) ApprovalRMProcessing(ctx *app.Context) *app.CostumeResponse {
	woUUID := ctx.Param("id")

	var req woModels.WorkOrderApprovalRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": errs},
		}
	}

	userCtx := userPkg.MustExtractUserContext(ctx)
	data, err := h.svc.ApprovalRMProcessing(ctx.Request.Context(), woUUID, req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// BulkApproval approves or rejects multiple work orders by wo_number.
// POST /api/v1/working-order/work-orders/bulk-approval
func (h *HTTPHandler) BulkApproval(ctx *app.Context) *app.CostumeResponse {
	var req woModels.BulkWorkOrderApprovalRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": errs},
		}
	}

	userCtx := userPkg.MustExtractUserContext(ctx)
	data, err := h.svc.BulkApproval(ctx.Request.Context(), req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// GetWorkOrderQR returns (and caches) QR base64 for WO header.
// GET /api/v1/working-order/work-orders/:id/qr
func (h *HTTPHandler) GetWorkOrderQR(ctx *app.Context) *app.CostumeResponse {
	woUUID := ctx.Param("id")
	refresh := strings.EqualFold(ctx.Query("refresh"), "1") || strings.EqualFold(ctx.Query("refresh"), "true")

	data, err := h.svc.GetWorkOrderQR(ctx.Request.Context(), woUUID, refresh)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// GetWorkOrderItemQR returns (and caches) QR base64 for WO item/kanban.
// GET /api/v1/working-order/work-order-items/:id/qr
func (h *HTTPHandler) GetWorkOrderItemQR(ctx *app.Context) *app.CostumeResponse {
	itemUUID := ctx.Param("id")
	refresh := strings.EqualFold(ctx.Query("refresh"), "1") || strings.EqualFold(ctx.Query("refresh"), "true")

	data, err := h.svc.GetWorkOrderItemQR(ctx.Request.Context(), itemUUID, refresh)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// UniqFormOptions returns union uniq_code options across items + inventory tables.
// GET /api/v1/working-order/work-orders/form-options/uniq?q=...&limit=20&sources=items,raw_material,indirect,subcon
func (h *HTTPHandler) UniqFormOptions(ctx *app.Context) *app.CostumeResponse {
	q := ctx.Query("q")
	limit := 20
	if v := strings.TrimSpace(ctx.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	var sources []string
	if s := strings.TrimSpace(ctx.Query("sources")); s != "" {
		sources = strings.Split(s, ",")
	}

	data, err := h.svc.ListUniqOptions(ctx.Request.Context(), q, limit, sources)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// ProcessFormOptions returns active process options for Work Order create form.
// GET /api/v1/working-order/work-orders/form-options/processes
func (h *HTTPHandler) ProcessFormOptions(ctx *app.Context) *app.CostumeResponse {
	data, err := h.svc.ListProcessOptions(ctx.Request.Context())
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      data,
	}
}

// ImportWorkOrderExcel POST /api/v1/working-order/work-orders/import
func (h *HTTPHandler) ImportWorkOrderExcel(ctx *app.Context) *app.CostumeResponse {
	file, err := ctx.FormFile("file")
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "file wajib diisi"}
	}
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".xlsx") {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "file harus format .xlsx"}
	}

	if err := os.MkdirAll("tmp", 0o755); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusInternalServerError, Message: "gagal membuat direktori tmp"}
	}
	tmpPath := filepath.Join("tmp", fmt.Sprintf("wo_import_%d_%s", time.Now().UnixNano(), filepath.Base(file.Filename)))
	if err := ctx.SaveUploadedFile(file, tmpPath); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusInternalServerError, Message: "gagal menyimpan file"}
	}
	defer os.Remove(tmpPath)

	userCtx := userPkg.MustExtractUserContext(ctx)
	uploadedBy := userCtx.UserID

	result, err := h.svc.ImportFromExcel(ctx.Request.Context(), tmpPath, file.Filename, uploadedBy, ctx.APIReqID)
	if err != nil {
		return app.NewError(ctx, err)
	}

	downloadURL := ""
	if result.ErrorToken != "" {
		scheme := "http"
		if proto := ctx.GetHeader("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else if ctx.Request.TLS != nil {
			scheme = "https"
		}
		downloadURL = fmt.Sprintf("%s://%s/api/v1/working-order/work-orders/import/errors/%s", scheme, ctx.Request.Host, result.ErrorToken)
	}

	status, message, data := approval.BuildBulkImportResponse(result, downloadURL)
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: status, Message: message, Data: data}
}

// DownloadImportTemplateRaw GET /api/v1/working-order/work-orders/import/template
func (h *HTTPHandler) DownloadImportTemplateRaw(c *gin.Context) {
	data, err := h.svc.DownloadImportTemplate(c.Request.Context())
	if err != nil {
		status := http.StatusInternalServerError
		msg := "gagal generate template"
		if appErr, ok := apperror.As(err); ok {
			status = appErr.HTTPStatus
			msg = appErr.Message
		}
		c.JSON(status, gin.H{"status": status, "message": msg})
		return
	}
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=work_order_template.xlsx")
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// DownloadImportErrorsRaw GET /api/v1/working-order/work-orders/import/errors/:token
func (h *HTTPHandler) DownloadImportErrorsRaw(c *gin.Context) {
	token := c.Param("token")
	data, err := h.svc.DownloadImportErrors(c.Request.Context(), token)
	if err != nil {
		status := http.StatusInternalServerError
		msg := "gagal download error file"
		if appErr, ok := apperror.As(err); ok {
			status = appErr.HTTPStatus
			msg = appErr.Message
		}
		c.JSON(status, gin.H{"status": status, "message": msg})
		return
	}
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=work_order_import_errors.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// GetImportHistory GET /api/v1/working-order/work-orders/import/history
func (h *HTTPHandler) GetImportHistory(ctx *app.Context) *app.CostumeResponse {
	result, err := h.svc.ListImportHistory(ctx.Request.Context(), 20)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      result,
	}
}

// DownloadImportHistoryErrorRaw GET /api/v1/working-order/work-orders/import/history/:id/errors
func (h *HTTPHandler) DownloadImportHistoryErrorRaw(c *gin.Context) {
	id := c.Param("id")
	data, err := h.svc.DownloadImportHistoryError(c.Request.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		msg := "gagal download error file"
		if appErr, ok := apperror.As(err); ok {
			status = appErr.HTTPStatus
			msg = appErr.Message
		}
		c.JSON(status, gin.H{"status": status, "message": msg})
		return
	}
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=work_order_import_errors.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}
