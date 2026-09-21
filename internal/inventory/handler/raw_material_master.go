package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	invModels "github.com/ganasa18/go-template/internal/inventory/models"
	userPkg "github.com/ganasa18/go-template/pkg/auth"
	"github.com/ganasa18/go-template/pkg/validator"
)

func masterID(ctx *app.Context) (int64, bool) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	return id, err == nil && id > 0
}

func (h *HTTPHandler) ListRawMaterialMasters(ctx *app.Context) *app.CostumeResponse {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	data, err := h.svc.ListRawMaterialMasters(ctx.Request.Context(), ctx.Query("search"), page, limit)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: data}
}
func (h *HTTPHandler) GetRawMaterialMaster(ctx *app.Context) *app.CostumeResponse {
	id, ok := masterID(ctx)
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: 400, Message: "invalid id"}
	}
	data, err := h.svc.GetRawMaterialMaster(ctx.Request.Context(), id)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: 200, Message: "OK", Data: data}
}
func (h *HTTPHandler) CreateRawMaterialMaster(ctx *app.Context) *app.CostumeResponse {
	var req invModels.CreateRawMaterialMasterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: 400, Message: "invalid request body: " + err.Error()}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: 422, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	u := userPkg.MustExtractUserContext(ctx)
	data, err := h.svc.CreateRawMaterialMaster(ctx.Request.Context(), req, u.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: 201, Message: "Created", Data: data}
}
func (h *HTTPHandler) UpdateRawMaterialMaster(ctx *app.Context) *app.CostumeResponse {
	id, ok := masterID(ctx)
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: 400, Message: "invalid id"}
	}
	var req invModels.UpdateRawMaterialMasterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: 400, Message: "invalid request body: " + err.Error()}
	}
	u := userPkg.MustExtractUserContext(ctx)
	data, err := h.svc.UpdateRawMaterialMaster(ctx.Request.Context(), id, req, u.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: 200, Message: "OK", Data: data}
}
func (h *HTTPHandler) DeleteRawMaterialMaster(ctx *app.Context) *app.CostumeResponse {
	id, ok := masterID(ctx)
	if !ok {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: 400, Message: "invalid id"}
	}
	u := userPkg.MustExtractUserContext(ctx)
	if err := h.svc.DeleteRawMaterialMaster(ctx.Request.Context(), id, u.UserID); err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: 200, Message: "deleted"}
}
func (h *HTTPHandler) ListRawMaterialPlanning(ctx *app.Context) *app.CostumeResponse {
	data, err := h.svc.ListRawMaterialPlanning(ctx.Request.Context())
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: 200, Message: "OK", Data: map[string]interface{}{"items": data}}
}
