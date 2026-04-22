package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/delivery_scheduling_customer/models"
	"github.com/ganasa18/go-template/internal/delivery_scheduling_customer/service"
	"github.com/ganasa18/go-template/pkg/validator"
)

type HTTPHandler struct {
	service service.IService
}

func New(svc service.IService) *HTTPHandler {
	return &HTTPHandler{service: svc}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func callerEmail(ctx *app.Context) string {
	if claims, ok := ctx.Get("claims"); ok && claims != nil {
		type withEmail interface{ GetEmail() string }
		if e, ok := claims.(withEmail); ok {
			return e.GetEmail()
		}
	}
	return ""
}

func badRequest(ctx *app.Context, msg string) *app.CostumeResponse {
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: msg}
}

func unprocessable(ctx *app.Context, msg string, detail interface{}) *app.CostumeResponse {
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: msg,
		Data: map[string]interface{}{"errors": detail},
	}
}

func notFound(ctx *app.Context, msg string) *app.CostumeResponse {
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusNotFound, Message: msg}
}

func ok(ctx *app.Context, data interface{}) *app.CostumeResponse {
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusOK, Message: http.StatusText(http.StatusOK), Data: data}
}

func created(ctx *app.Context, data interface{}) *app.CostumeResponse {
	return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusCreated, Message: "Created", Data: data}
}

// ─── Schedule Handlers ────────────────────────────────────────────────────────

func (h *HTTPHandler) CreateSchedule(ctx *app.Context) *app.CostumeResponse {
	var req models.CreateScheduleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return badRequest(ctx, "invalid request body")
	}
	if errs := validator.Validate(req); errs != nil {
		return unprocessable(ctx, "validation failed", errs)
	}

	data, err := h.service.CreateSchedule(ctx.Request.Context(), req, callerEmail(ctx))
	if err != nil {
		return unprocessable(ctx, err.Error(), nil)
	}
	return created(ctx, data)
}

func (h *HTTPHandler) GetSchedulesSummary(ctx *app.Context) *app.CostumeResponse {
	deliveryDate := ctx.Query("delivery_date")
	data, err := h.service.GetSchedulesSummary(ctx.Request.Context(), deliveryDate)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return ok(ctx, data)
}

func (h *HTTPHandler) GetSchedulesList(ctx *app.Context) *app.CostumeResponse {
	q := ctx.Request.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	customerID, _ := strconv.ParseInt(q.Get("customer_id"), 10, 64)

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	f := models.ScheduleListFilter{
		DeliveryDate: q.Get("delivery_date"),
		CustomerID:   customerID,
		Status:       q.Get("status"),
		Page:         page,
		Limit:        limit,
	}

	data, err := h.service.GetSchedulesList(ctx.Request.Context(), f)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return ok(ctx, data)
}

func (h *HTTPHandler) GetScheduleDetail(ctx *app.Context) *app.CostumeResponse {
	id := ctx.Param("id")
	data, err := h.service.GetScheduleDetail(ctx.Request.Context(), id)
	if err != nil {
		return notFound(ctx, err.Error())
	}
	return ok(ctx, data)
}

func (h *HTTPHandler) ApproveSchedule(ctx *app.Context) *app.CostumeResponse {
	id := ctx.Param("id")
	var req models.ApproveScheduleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return badRequest(ctx, "invalid request body")
	}

	data, err := h.service.ApproveSchedule(ctx.Request.Context(), id, req, callerEmail(ctx))
	if err != nil {
		return unprocessable(ctx, err.Error(), nil)
	}
	return ok(ctx, data)
}

func (h *HTTPHandler) ApproveAll(ctx *app.Context) *app.CostumeResponse {
	var req models.ApproveAllRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return badRequest(ctx, "invalid request body")
	}
	if errs := validator.Validate(req); errs != nil {
		return unprocessable(ctx, "validation failed", errs)
	}

	data, err := h.service.ApproveAll(ctx.Request.Context(), req, callerEmail(ctx))
	if err != nil {
		return unprocessable(ctx, err.Error(), nil)
	}
	return ok(ctx, data)
}

func (h *HTTPHandler) ApprovePartial(ctx *app.Context) *app.CostumeResponse {
	var req models.ApprovePartialRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return badRequest(ctx, "invalid request body")
	}
	if errs := validator.Validate(req); errs != nil {
		return unprocessable(ctx, "validation failed", errs)
	}

	data, err := h.service.ApprovePartial(ctx.Request.Context(), req, callerEmail(ctx))
	if err != nil {
		return unprocessable(ctx, err.Error(), nil)
	}
	return ok(ctx, data)
}

// ─── Customer DN Handlers ─────────────────────────────────────────────────────

func (h *HTTPHandler) CreateCustomerDN(ctx *app.Context) *app.CostumeResponse {
	var req models.CreateCustomerDNRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return badRequest(ctx, "invalid request body")
	}
	if errs := validator.Validate(req); errs != nil {
		return unprocessable(ctx, "validation failed", errs)
	}

	data, err := h.service.CreateCustomerDN(ctx.Request.Context(), req, callerEmail(ctx))
	if err != nil {
		return unprocessable(ctx, err.Error(), nil)
	}
	return created(ctx, data)
}

func (h *HTTPHandler) GetDNList(ctx *app.Context) *app.CostumeResponse {
	q := ctx.Request.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	customerID, _ := strconv.ParseInt(q.Get("customer_id"), 10, 64)

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	f := models.DNListFilter{
		DeliveryDate: q.Get("delivery_date"),
		CustomerID:   customerID,
		Status:       q.Get("status"),
		Page:         page,
		Limit:        limit,
	}

	data, err := h.service.GetDNList(ctx.Request.Context(), f)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return ok(ctx, data)
}

func (h *HTTPHandler) GetDNDetail(ctx *app.Context) *app.CostumeResponse {
	id := ctx.Param("id")
	data, err := h.service.GetDNDetail(ctx.Request.Context(), id)
	if err != nil {
		return notFound(ctx, err.Error())
	}
	return ok(ctx, data)
}

func (h *HTTPHandler) ConfirmDN(ctx *app.Context) *app.CostumeResponse {
	id := ctx.Param("id")
	var req models.ConfirmDNRequest
	_ = ctx.ShouldBindJSON(&req)

	data, err := h.service.ConfirmDN(ctx.Request.Context(), id, req)
	if err != nil {
		return unprocessable(ctx, err.Error(), nil)
	}
	return ok(ctx, data)
}

// ─── Delivery Scan Handlers ───────────────────────────────────────────────────

func (h *HTTPHandler) LookupDeliveryItem(ctx *app.Context) *app.CostumeResponse {
	dnNumber := ctx.Query("dn_number")
	uniqCode := ctx.Query("item_uniq_code")

	if dnNumber == "" || uniqCode == "" {
		return badRequest(ctx, "dn_number dan item_uniq_code wajib diisi")
	}

	data, err := h.service.LookupDeliveryItem(ctx.Request.Context(), dnNumber, uniqCode)
	if err != nil {
		return notFound(ctx, err.Error())
	}
	return ok(ctx, data)
}

func (h *HTTPHandler) SubmitDeliveryScan(ctx *app.Context) *app.CostumeResponse {
	var req models.SubmitScanRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return badRequest(ctx, "invalid request body")
	}
	if errs := validator.Validate(req); errs != nil {
		return unprocessable(ctx, "validation failed", errs)
	}

	data, err := h.service.SubmitDeliveryScan(ctx.Request.Context(), req, callerEmail(ctx))
	if err != nil {
		return unprocessable(ctx, err.Error(), nil)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "Delivery scan processed",
		Data:      data,
	}
}
