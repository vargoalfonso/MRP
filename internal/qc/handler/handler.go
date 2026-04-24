package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	qcModels "github.com/ganasa18/go-template/internal/qc/models"
	qcRepo "github.com/ganasa18/go-template/internal/qc/repository"
	qcService "github.com/ganasa18/go-template/internal/qc/service"
	userPkg "github.com/ganasa18/go-template/pkg/auth"
	"github.com/ganasa18/go-template/pkg/pagination"
)

type HTTPHandler struct {
	svc qcService.IService
}

func New(svc qcService.IService) *HTTPHandler { return &HTTPHandler{svc: svc} }

// GET /api/v1/qc/tasks
func (h *HTTPHandler) ListTasks(ctx *app.Context) *app.CostumeResponse {
	p := pagination.QCTaskPagination(ctx)

	f := qcRepo.ListFilter{
		TaskType: p.TaskType,
		Status:   p.Status,
		Page:     p.Page,
		Limit:    p.Limit,
		Offset:   p.Offset(),
	}

	rows, total, err := h.svc.ListTasks(ctx.Request.Context(), f)
	if err != nil {
		return app.NewError(ctx, err)
	}

	resp := qcModels.TaskListResponse{}
	resp.Items = make([]qcModels.TaskListItem, 0, len(rows))
	for _, r := range rows {
		item := qcModels.TaskListItem{
			ID:            r.ID,
			TaskType:      r.TaskType,
			Status:        r.Status,
			CreatedAt:     r.CreatedAt,
			PackingNumber: r.PackingNumber,
			DnNumber:      r.DnNumber,
			PoNumber:      r.PoNumber,
			SupplierName:  r.SupplierName,
			ItemUniqCode:  r.ItemUniqCode,
			QtyReceived:   r.QtyReceived,
			UOM:           r.UOM,
		}
		resp.Items = append(resp.Items, item)
	}
	resp.Pagination.Total = total
	resp.Pagination.Page = p.Page
	resp.Pagination.Limit = p.Limit
	resp.Pagination.TotalPages = qcService.TotalPages(total, p.Limit)

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      resp,
	}
}

// POST /api/v1/qc/tasks/:id/start
func (h *HTTPHandler) StartTask(ctx *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	if err := h.svc.StartTask(ctx.Request.Context(), id, userCtx.UserID); err != nil {
		return app.NewError(ctx, err)
	}

	resp := qcModels.StartTaskResponse{TaskID: id, Status: "in_progress"}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      resp,
	}
}

// POST /api/v1/qc/tasks/:id/approve
func (h *HTTPHandler) ApproveIncoming(ctx *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	var req qcModels.ApproveIncomingTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body: " + err.Error()}
	}

	userCtx := userPkg.MustExtractUserContext(ctx)
	if err := h.svc.ApproveIncoming(ctx.Request.Context(), id, req.NumberOfDefects, req.DateChecked, userCtx.UserID, req.Defects); err != nil {
		return app.NewError(ctx, err)
	}

	// Minimal response; client can refetch task list / DN item.
	resp := map[string]interface{}{"qc_task_id": id, "status": "approved"}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: resp}
}

// POST /api/v1/qc/tasks/:id/reject
func (h *HTTPHandler) RejectIncoming(ctx *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	var req qcModels.RejectIncomingTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body: " + err.Error()}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	if err := h.svc.RejectIncoming(ctx.Request.Context(), id, req.NumberOfDefects, req.DateChecked, userCtx.UserID, req.Defects); err != nil {
		return app.NewError(ctx, err)
	}
	resp := map[string]interface{}{"qc_task_id": id, "status": "rejected"}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: resp}
}
