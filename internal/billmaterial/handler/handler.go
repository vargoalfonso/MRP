// Package handler exposes BOM HTTP endpoints.
//
// GET  /api/v1/products/bom        — list (expandable tree)
// POST /api/v1/products/bom        — create (wizard: parent + children in one call)
// GET  /api/v1/products/bom/:id    — detail (full tree with routing + material spec)
package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	authModels "github.com/ganasa18/go-template/internal/auth/models"
	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/billmaterial/models"
	"github.com/ganasa18/go-template/internal/billmaterial/service"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/approval"
	"github.com/ganasa18/go-template/pkg/pagination"
	"github.com/ganasa18/go-template/pkg/validator"
	"github.com/gin-gonic/gin"
)

type HTTPHandler struct {
	svc service.IService
}

func New(svc service.IService) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

// ListBom  GET /api/v1/products/bom
//
// Query params:
//
//	limit=20&page=1&search=LV7&uniq_code=LV7&status=Active
//	orderBy=created_at&orderDirection=desc
//	filter=status:eq:Active
func (h *HTTPHandler) ListBom(ctx *app.Context) *app.CostumeResponse {
	p := pagination.BomPagination(ctx)

	resp, err := h.svc.ListBom(ctx.Request.Context(), models.ListBomQuery{
		UniqCode:       p.UniqCode,
		Status:         p.Status,
		Search:         p.Search,
		Page:           p.Page,
		Limit:          p.Limit,
		OrderBy:        p.OrderBy,
		OrderDirection: p.OrderDirection,
	})
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      resp,
	}
}

// CreateBom  POST /api/v1/products/bom
func (h *HTTPHandler) CreateBom(ctx *app.Context) *app.CostumeResponse {
	var req models.CreateBomRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}

	result, err := h.svc.CreateBom(ctx.Request.Context(), req)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusCreated,
		Message:   http.StatusText(http.StatusCreated),
		Data:      result,
	}
}

// GetBomDetail  GET /api/v1/products/bom/:id
func (h *HTTPHandler) GetBomDetail(ctx *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	versionParam := ctx.Query("version")
	if versionParam != "" {
		version, err := strconv.Atoi(versionParam)
		if err != nil {
			return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid version"}
		}
		result, err := h.svc.GetBomDetailByVersion(ctx.Request.Context(), id, version)
		if err != nil {
			return app.NewError(ctx, err)
		}
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: result}
	}
	result, err := h.svc.GetBomDetail(ctx.Request.Context(), id)
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

func (h *HTTPHandler) GetBomVersions(ctx *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	result, err := h.svc.GetBomVersions(ctx.Request.Context(), id)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: result}
}

func (h *HTTPHandler) CreateBomRevision(ctx *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	var req models.CreateBomRevisionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	result, err := h.svc.CreateBomRevision(ctx.Request.Context(), id, req)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusCreated, Message: http.StatusText(http.StatusCreated), Data: result}
}

// ActivateBomVersion  POST /api/v1/products/bom/:id/activate
func (h *HTTPHandler) ActivateBomVersion(ctx *app.Context) *app.CostumeResponse {
	bomID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	result, err := h.svc.ActivateBomVersion(ctx.Request.Context(), bomID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: result}
}

func (h *HTTPHandler) AddProcessRoute(ctx *app.Context) *app.CostumeResponse {
	bomID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	var reqs []models.AddProcessRouteRequest
	if err := ctx.ShouldBindJSON(&reqs); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	for _, req := range reqs {
		if errs := validator.Validate(req); errs != nil {
			return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
		}
	}
	result, err := h.svc.AddProcessRoute(ctx.Request.Context(), bomID, reqs)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusCreated, Message: http.StatusText(http.StatusCreated), Data: result}
}

func (h *HTTPHandler) PatchProcessRoute(ctx *app.Context) *app.CostumeResponse {
	bomID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	routeID, err := strconv.ParseInt(ctx.Param("route_id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid route_id"}
	}
	var req models.PatchProcessRouteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	result, err := h.svc.PatchProcessRoute(ctx.Request.Context(), bomID, routeID, req)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: result}
}

// UpdateBom  PUT /api/v1/products/bom/:id
func (h *HTTPHandler) UpdateBom(ctx *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	var req models.UpdateBomRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	result, err := h.svc.UpdateBom(ctx.Request.Context(), id, req)
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

// DeleteBom DELETE /api/v1/products/bom/:id
func (h *HTTPHandler) DeleteBom(ctx *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	if err := h.svc.DeleteBom(ctx.Request.Context(), id); err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      map[string]interface{}{"deleted": true, "bom_id": id},
	}
}

// UpdateBomChild  PUT /api/v1/products/bom/:id/lines/:line_id
func (h *HTTPHandler) UpdateBomChild(ctx *app.Context) *app.CostumeResponse {
	bomID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	lineID, err := strconv.ParseInt(ctx.Param("line_id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid line_id"}
	}
	var req models.UpdateBomChildRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	result, err := h.svc.UpdateBomChild(ctx.Request.Context(), bomID, lineID, req)
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

// DeleteBomChild DELETE /api/v1/products/bom/:id/children/:child_id
func (h *HTTPHandler) DeleteBomChild(ctx *app.Context) *app.CostumeResponse {
	bomID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	childID, err := strconv.ParseInt(ctx.Param("child_id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid child_id"}
	}

	deletedLines, err := h.svc.DeleteBomChild(ctx.Request.Context(), bomID, childID)
	if err != nil {
		return app.NewError(ctx, err)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data: map[string]interface{}{
			"deleted":       true,
			"bom_id":        bomID,
			"child_item_id": childID,
			"deleted_lines": deletedLines,
		},
	}
}

// DeleteBomLine DELETE /api/v1/products/bom/:id/lines/:line_id
func (h *HTTPHandler) DeleteBomLine(ctx *app.Context) *app.CostumeResponse {
	bomID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	lineID, err := strconv.ParseInt(ctx.Param("line_id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid line_id"}
	}

	deletedLines, err := h.svc.DeleteBomLine(ctx.Request.Context(), bomID, lineID)
	if err != nil {
		return app.NewError(ctx, err)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data: map[string]interface{}{
			"deleted":        true,
			"delete_scope":   "subtree",
			"bom_id":         bomID,
			"target_line_id": lineID,
			"deleted_lines":  deletedLines,
		},
	}
}

// ApproveBom  POST /api/v1/products/bom/:id/approval
//
// Body: { "action": "approve"|"reject", "notes": "..." }
// The caller must have the role required for the current approval level.
func (h *HTTPHandler) ApproveBom(ctx *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}

	var req models.ApproveBomRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}

	raw, ok := ctx.Get("claims")
	if !ok {
		return app.NewError(ctx, apperror.Unauthorized("missing auth claims"))
	}
	claims, ok := raw.(*authModels.Claims)
	if !ok || claims == nil || claims.UserID == "" {
		return app.NewError(ctx, apperror.Unauthorized("invalid auth claims"))
	}

	result, err := h.svc.ApproveBom(ctx.Request.Context(), id, claims.UserID, claims.Roles, req)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: result}
}

// ImportBomExcel POST /api/v1/products/bom/import
func (h *HTTPHandler) ImportBomExcel(ctx *app.Context) *app.CostumeResponse {
	file, err := ctx.FormFile("file")
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "file wajib diisi"}
	}
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".xlsx") {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "file harus format .xlsx"}
	}

	tmpPath := filepath.Join("tmp", fmt.Sprintf("bom_import_%d_%s", time.Now().UnixNano(), filepath.Base(file.Filename)))
	if err := ctx.SaveUploadedFile(file, tmpPath); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusInternalServerError, Message: "gagal menyimpan file"}
	}
	defer os.Remove(tmpPath)

	result, err := h.svc.ImportFromExcel(ctx.Request.Context(), tmpPath)
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
		downloadURL = fmt.Sprintf("%s://%s/api/v1/products/bom/import/errors/%s", scheme, ctx.Request.Host, result.ErrorToken)
	}

	status, message, data := approval.BuildBulkImportResponse(result, downloadURL)
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: status, Message: message, Data: data}
}

// DownloadImportTemplateRaw GET /api/v1/products/bom/import/template
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
	c.Header("Content-Disposition", "attachment; filename=bom_template.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// DownloadImportErrorsRaw GET /api/v1/products/bom/import/errors/:token
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
	c.Header("Content-Disposition", "attachment; filename=bom_errors.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}
