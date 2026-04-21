package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	outModels "github.com/ganasa18/go-template/internal/outgoing_material/models"
	outService "github.com/ganasa18/go-template/internal/outgoing_material/service"
	userPkg "github.com/ganasa18/go-template/pkg/auth"
	"github.com/ganasa18/go-template/pkg/pagination"
	"github.com/ganasa18/go-template/pkg/validator"
)

type HTTPHandler struct {
	svc outService.IService
}

func New(svc outService.IService) *HTTPHandler { return &HTTPHandler{svc: svc} }

func parseID(ctx *app.Context) (int64, bool) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// ListOutgoingRM returns paginated outgoing RM transactions.
//
//	GET /api/v1/outgoing-raw-materials?date_from=2024-01-01&date_to=2024-12-31&reason=Production+Use&uniq=RM-PL-689795&limit=20&page=1
func (h *HTTPHandler) ListOutgoingRM(ctx *app.Context) *app.CostumeResponse {
	p := pagination.OutgoingRMPagination(ctx)
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

// GetOutgoingRMByID returns a single outgoing RM transaction detail.
//
//	GET /api/v1/outgoing-raw-materials/:id
func (h *HTTPHandler) GetOutgoingRMByID(ctx *app.Context) *app.CostumeResponse {
	id, ok := parseID(ctx)
	if !ok {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid id",
		}
	}
	data, err := h.svc.GetByID(ctx.Request.Context(), id)
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

// CreateOutgoingRM processes an outgoing RM transaction.
// Atomically deducts stock from raw_materials, records the transaction, and writes audit log.
//
//	POST /api/v1/outgoing-raw-materials
func (h *HTTPHandler) CreateOutgoingRM(ctx *app.Context) *app.CostumeResponse {
	var req outModels.CreateOutgoingRMRequest
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
		Message:   http.StatusText(http.StatusCreated),
		Data:      data,
	}
}
