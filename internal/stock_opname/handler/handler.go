package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	stockModels "github.com/ganasa18/go-template/internal/stock_opname/models"
	"github.com/ganasa18/go-template/internal/stock_opname/repository"
	stockService "github.com/ganasa18/go-template/internal/stock_opname/service"
	userPkg "github.com/ganasa18/go-template/pkg/auth"
	"github.com/ganasa18/go-template/pkg/pagination"
	"github.com/ganasa18/go-template/pkg/validator"
)

type HTTPHandler struct{ svc stockService.IService }

func New(svc stockService.IService) *HTTPHandler { return &HTTPHandler{svc: svc} }

func parseID(ctx *app.Context, key string) (int64, bool) {
	id, err := strconv.ParseInt(ctx.Param(key), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func (h *HTTPHandler) GetStats(ctx *app.Context) *app.CostumeResponse {
	resp, err := h.svc.GetStats(ctx.Request.Context(), ctx.Query("type"))
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: resp}
}

func (h *HTTPHandler) ListUniqOptions(ctx *app.Context) *app.CostumeResponse {
	limit := 20
	if v := ctx.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	resp, err := h.svc.ListUniqOptions(ctx.Request.Context(), stockModels.FormOptionsQuery{Type: ctx.Query("type"), Q: ctx.Query("q"), Limit: limit})
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: resp}
}

func (h *HTTPHandler) GetHistoryLogs(ctx *app.Context) *app.CostumeResponse {
	p := pagination.Pagination(ctx)
	resp, err := h.svc.GetHistoryLogs(ctx.Request.Context(), stockModels.HistoryLogsQuery{Type: ctx.Query("type"), UniqCode: ctx.Query("uniq_code"), From: ctx.Query("from"), To: ctx.Query("to"), Limit: p.Limit, Offset: p.Offset(), Page: p.Page})
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: resp}
}

func (h *HTTPHandler) ListSessions(ctx *app.Context) *app.CostumeResponse {
	p := pagination.StockOpnamePagination(ctx)
	resp, err := h.svc.ListSessions(ctx.Request.Context(), repository.SessionFilter{Type: p.Type, Status: p.Status, Period: p.Period, Page: p.Page, Limit: p.Limit, Offset: p.Offset(), OrderBy: p.OrderBy, OrderDirection: p.OrderDirection})
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: resp}
}

func (h *HTTPHandler) CreateSession(ctx *app.Context) *app.CostumeResponse {
	var req stockModels.CreateSessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body: " + err.Error()}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	resp, err := h.svc.CreateSession(ctx.Request.Context(), req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusCreated, Message: http.StatusText(http.StatusCreated), Data: resp}
}

func (h *HTTPHandler) GetSessionByID(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx, "id")
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	resp, err := h.svc.GetSessionByID(ctx.Request.Context(), id)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: resp}
}

func (h *HTTPHandler) GetAuditLogs(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx, "id")
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	p := pagination.Pagination(ctx)
	resp, err := h.svc.GetAuditLogs(ctx.Request.Context(), id, p.Page, p.Limit)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: resp}
}

func (h *HTTPHandler) UpdateSession(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx, "id")
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	var req stockModels.UpdateSessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body: " + err.Error()}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	resp, err := h.svc.UpdateSession(ctx.Request.Context(), id, req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: resp}
}

func (h *HTTPHandler) DeleteSession(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx, "id")
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	if err := h.svc.DeleteSession(ctx.Request.Context(), id, userCtx.UserID); err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK)}
}

func (h *HTTPHandler) AddEntry(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx, "id")
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	var req stockModels.CreateEntryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body: " + err.Error()}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	resp, err := h.svc.AddEntry(ctx.Request.Context(), id, req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusCreated, Message: http.StatusText(http.StatusCreated), Data: resp}
}

func (h *HTTPHandler) BulkAddEntries(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx, "id")
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	var req stockModels.BulkCreateEntryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body: " + err.Error()}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	resp, err := h.svc.BulkAddEntries(ctx.Request.Context(), id, req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusCreated, Message: http.StatusText(http.StatusCreated), Data: resp}
}

func (h *HTTPHandler) UpdateEntry(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx, "id")
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	entryID, ok := parseID(ctx, "entryId")
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid entry id"}
	}
	var req stockModels.UpdateEntryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body: " + err.Error()}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	resp, err := h.svc.UpdateEntry(ctx.Request.Context(), id, entryID, req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: resp}
}

func (h *HTTPHandler) DeleteEntry(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx, "id")
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	entryID, ok := parseID(ctx, "entryId")
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid entry id"}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	if err := h.svc.DeleteEntry(ctx.Request.Context(), id, entryID, userCtx.UserID); err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK)}
}

func (h *HTTPHandler) SubmitSession(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx, "id")
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	resp, err := h.svc.SubmitSession(ctx.Request.Context(), id, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: resp}
}

func (h *HTTPHandler) ApproveSession(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx, "id")
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	var req stockModels.ApproveRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body: " + err.Error()}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	resp, err := h.svc.ApproveSession(ctx.Request.Context(), id, req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: resp}
}

func (h *HTTPHandler) ApproveEntry(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx, "id")
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	entryID, ok := parseID(ctx, "entryId")
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid entry id"}
	}
	var req stockModels.ApproveRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body: " + err.Error()}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	resp, err := h.svc.ApproveEntry(ctx.Request.Context(), id, entryID, req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: resp}
}
