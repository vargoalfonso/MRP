package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/approval_manager/models"
	approvalService "github.com/ganasa18/go-template/internal/approval_manager/service"
	"github.com/ganasa18/go-template/internal/base/app"
	userPkg "github.com/ganasa18/go-template/pkg/auth"
	"github.com/ganasa18/go-template/pkg/pagination"
)

type HTTPHandler struct{ svc approvalService.IService }

func New(svc approvalService.IService) *HTTPHandler { return &HTTPHandler{svc: svc} }

func (h *HTTPHandler) GetSummary(ctx *app.Context) *app.CostumeResponse {
	resp, err := h.svc.GetSummary(ctx.Request.Context(), ctx.Query("type"))
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: resp}
}

func (h *HTTPHandler) ListItems(ctx *app.Context) *app.CostumeResponse {
	p := pagination.ApprovalManagerPagination(ctx)
	userCtx := userPkg.MustExtractUserContext(ctx)
	resp, err := h.svc.ListItems(ctx.Request.Context(), models.ListQuery{Type: p.Type, Status: p.Status, Search: p.Search, SubmittedBy: p.SubmittedBy, CurrentLevel: p.CurrentLevel, Scope: p.Scope, Page: p.Page, Limit: p.Limit, Offset: p.Offset()}, userCtx.Roles)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: resp}
}

func (h *HTTPHandler) GetDetail(ctx *app.Context) *app.CostumeResponse {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	resp, svcErr := h.svc.GetDetail(ctx.Request.Context(), id, userCtx.Roles)
	if svcErr != nil {
		return app.NewError(ctx, svcErr)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: resp}
}
