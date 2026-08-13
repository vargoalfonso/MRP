package handler

import (
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/action_ui/dto"
	actionModels "github.com/ganasa18/go-template/internal/action_ui/models"
	actionService "github.com/ganasa18/go-template/internal/action_ui/service"
	"github.com/ganasa18/go-template/internal/base/app"
	userPkg "github.com/ganasa18/go-template/pkg/auth"
)

type HTTPHandler struct {
	svc actionService.IService
}

func New(svc actionService.IService) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

// GET /api/v1/action-ui/incoming/lookup?packing_number=KB-123456&item_uniq_code=UQ-123456
// Called when QR is scanned — auto-fills PO Number, Supplier, DN Number, Type on the form.
func (h *HTTPHandler) LookupByPackingNumber(ctx *app.Context) *app.CostumeResponse {
	packingNumber := ctx.Query("packing_number")
	if packingNumber == "" {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "packing_number is required",
		}
	}
	itemUniqCode := ctx.Query("item_uniq_code")

	result, err := h.svc.LookupByPackingNumber(ctx.Request.Context(), packingNumber, itemUniqCode)
	if err != nil {
		return app.NewError(ctx, err)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

// POST /api/v1/action-ui/incoming/scans
func (h *HTTPHandler) CreateIncomingScan(ctx *app.Context) *app.CostumeResponse {
	var req actionModels.IncomingScanRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}

	userCtx := userPkg.MustExtractUserContext(ctx)
	resp, idempotentHit, err := h.svc.CreateIncomingScan(ctx.Request.Context(), req, userCtx.UserID)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	status := http.StatusCreated
	message := "Created"
	if idempotentHit {
		status = http.StatusOK
		message = http.StatusText(http.StatusOK)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    status,
		Message:   message,
		Data:      resp,
	}
}

func (h *HTTPHandler) ScanContext(ctx *app.Context) *app.CostumeResponse {
	wo := ctx.Query("wo")
	if wo == "" {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "wo is required",
		}
	}

	result, err := h.svc.ScanContext(ctx.Request.Context(), wo)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

func (h *HTTPHandler) ScanContextMachine(ctx *app.Context) *app.CostumeResponse {
	machine := ctx.Query("machine")
	if machine == "" {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "machine is required",
		}
	}

	result, err := h.svc.ScanContextMachine(ctx.Request.Context(), machine)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

// GET /api/v1/action-ui/production/wo-list?search=
func (h *HTTPHandler) WOList(ctx *app.Context) *app.CostumeResponse {
	result, err := h.svc.WOList(ctx.Request.Context(), ctx.Query("search"))
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

// GET /api/v1/action-ui/production/wo-detail?wo=WO-204501
func (h *HTTPHandler) WODetail(ctx *app.Context) *app.CostumeResponse {
	wo := ctx.Query("wo")
	if wo == "" {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "wo is required",
		}
	}
	result, err := h.svc.WODetail(ctx.Request.Context(), wo)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

// GET /api/v1/action-ui/production/raw-material?code=RM-160637
func (h *HTTPHandler) RawMaterialLookup(ctx *app.Context) *app.CostumeResponse {
	code := ctx.Query("code")
	if code == "" {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "code is required",
		}
	}
	result, err := h.svc.RawMaterialLookup(ctx.Request.Context(), code)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

func (h *HTTPHandler) ScanIn(ctx *app.Context) *app.CostumeResponse {
	var req dto.ScanInRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}

	err := h.svc.ScanIn(ctx.Request.Context(), req)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "Scan In Success",
	}
}

func (h *HTTPHandler) ScanOut(ctx *app.Context) *app.CostumeResponse {
	var req dto.ScanOutRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}

	err := h.svc.ScanOut(ctx.Request.Context(), req)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "Scan Out Success",
	}
}

func (h *HTTPHandler) CompleteProduction(ctx *app.Context) *app.CostumeResponse {
	var req dto.CompleteWORequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}

	err := h.svc.CompleteProduction(ctx.Request.Context(), req.WOID)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "Work Order Completed",
	}
}

func (h *HTTPHandler) QCApprove(ctx *app.Context) *app.CostumeResponse {
	return h.qcSubmitWithStatus(ctx, "approve")
}

func (h *HTTPHandler) QCReject(ctx *app.Context) *app.CostumeResponse {
	return h.qcSubmitWithStatus(ctx, "reject")
}

func (h *HTTPHandler) qcSubmitWithStatus(ctx *app.Context, status string) *app.CostumeResponse {
	var req dto.QCSubmitRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}

	req.Status = status

	userCtx := userPkg.MustExtractUserContext(ctx)

	err := h.svc.QCSubmit(ctx.Request.Context(), req, userCtx.UserID)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "QC Submit Success",
	}
}

func (h *HTTPHandler) QCFinish(ctx *app.Context) *app.CostumeResponse {
	var req dto.QCFinishRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}

	userCtx := userPkg.MustExtractUserContext(ctx)

	if err := h.svc.QCFinish(ctx.Request.Context(), req, userCtx.UserID); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "QC Finish Success",
	}
}

// GET /api/v1/action-ui/qc/rounds?wo_id=&wo_item_id=
// [qc-round-db] Mengembalikan Round 1 & 2 yang sudah tersubmit untuk satu
// wo_item (dari qc_logs + qc_defect_items) supaya frontend QC Process bisa
// memulihkan data ronde lintas gadget tanpa localStorage.
func (h *HTTPHandler) QCRounds(ctx *app.Context) *app.CostumeResponse {
	qcTaskID, err := strconv.ParseInt(ctx.Query("qc_task_id"), 10, 64)
	if err != nil || qcTaskID == 0 {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "qc_task_id is required",
		}
	}

	data, err := h.svc.ListQCRounds(ctx.Request.Context(), qcTaskID)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "failed get qc rounds",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "QC Rounds Success",
		Data:      data,
	}
}

func (h *HTTPHandler) ListQCTask(ctx *app.Context) *app.CostumeResponse {
	var req dto.ListQCTaskRequest

	if err := ctx.ShouldBindQuery(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid query params: " + err.Error(),
		}
	}

	data, err := h.svc.ListQCTask(ctx.Request.Context(), req)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "failed get qc task list",
			Data: map[string]interface{}{
				"errors": err.Error(),
			},
		}
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "QC Task List Success",
		Data:      data,
	}
}

func (h *HTTPHandler) IssueList(ctx *app.Context) *app.CostumeResponse {

	result, err := h.svc.IssueList(ctx.Request.Context())
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}
func (h *HTTPHandler) ScanMachine(ctx *app.Context) *app.CostumeResponse {
	var req dto.ScanMachineRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}

	result, err := h.svc.ScanMachine(ctx.Request.Context(), req)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

func (h *HTTPHandler) ScanOutContext(ctx *app.Context) *app.CostumeResponse {
	wo := ctx.Query("wo")
	if wo == "" {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "wo is required",
		}
	}

	result, err := h.svc.ScanOutContext(ctx.Request.Context(), wo)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

// POST /api/v1/action-ui/qc-return/scan
func (h *HTTPHandler) ScanReturn(ctx *app.Context) *app.CostumeResponse {
	var req actionModels.ScanReturnRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}

	result, err := h.svc.ScanReturn(ctx.Request.Context(), req)
	if err != nil {
		return app.NewError(ctx, err)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

// POST /api/v1/action-ui/qc-return/submit-to-qc
func (h *HTTPHandler) SubmitReturnToQC(ctx *app.Context) *app.CostumeResponse {
	var req actionModels.SubmitReturnToQCRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}

	userCtx := userPkg.MustExtractUserContext(ctx)
	result, err := h.svc.SubmitReturnToQC(ctx.Request.Context(), req, userCtx.UserID)
	if err != nil {
		return app.NewError(ctx, err)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusCreated,
		Message:   "OK",
		Data:      result,
	}
}

// GET /api/v1/action-ui/qc-return/pending-tasks
func (h *HTTPHandler) PendingReturnTasks(ctx *app.Context) *app.CostumeResponse {
	result, err := h.svc.PendingReturnTasks(ctx.Request.Context())
	if err != nil {
		return app.NewError(ctx, err)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

// POST /api/v1/action-ui/qc-return/submit-validation
func (h *HTTPHandler) SubmitReturnValidation(ctx *app.Context) *app.CostumeResponse {
	var req actionModels.SubmitReturnValidationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}

	userCtx := userPkg.MustExtractUserContext(ctx)
	if err := h.svc.SubmitReturnValidation(ctx.Request.Context(), req, userCtx.UserID); err != nil {
		return app.NewError(ctx, err)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
	}
}

// POST /api/v1/action-ui/production/rm-repack
// [repack-sisa] Pindahkan sisa material dari packing asal ke packing tujuan
// yang masih punya slot (dipakai modal Repacking setelah Scan Out).
func (h *HTTPHandler) RMRepack(ctx *app.Context) *app.CostumeResponse {
	var req dto.RMRepackRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}

	result, err := h.svc.RMRepack(ctx.Request.Context(), req)
	if err != nil {
		return app.NewError(ctx, err)
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

// GET /api/v1/action-ui/production/rm-packing-list?rm_uuid=..&code=..
// Dipakai modal Repacking untuk menampilkan packing/kanban milik satu Raw Material.
func (h *HTTPHandler) RMPackingList(ctx *app.Context) *app.CostumeResponse {
	rmUUID := ctx.Query("rm_uuid")
	code := ctx.Query("code")
	if rmUUID == "" && code == "" {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "rm_uuid atau code wajib diisi",
		}
	}

	result, err := h.svc.RMPackingList(ctx.Request.Context(), rmUUID, code)
	if err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      map[string]interface{}{"errors": err.Error()},
		}
	}

	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

// ================================
// [scanin-draft-db] Draft Scan-In (seed) bersama lintas gadget
// ================================

// GET /api/v1/action-ui/production/scanin-draft?wo_id=123&current_step=1
func (h *HTTPHandler) ListScanInDrafts(ctx *app.Context) *app.CostumeResponse {
	woID, _ := strconv.ParseInt(ctx.Query("wo_id"), 10, 64)
	if woID <= 0 {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "wo_id is required",
		}
	}
	currentStep, _ := strconv.Atoi(ctx.Query("current_step"))
	result, err := h.svc.ListScanInDrafts(ctx.Request.Context(), woID, currentStep)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
		Data:      result,
	}
}

// PUT /api/v1/action-ui/production/scanin-draft
func (h *HTTPHandler) UpsertScanInDraft(ctx *app.Context) *app.CostumeResponse {
	var req dto.UpsertScanInDraftRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}
	userCtx := userPkg.MustExtractUserContext(ctx)
	if err := h.svc.UpsertScanInDraft(ctx.Request.Context(), req, userCtx.UserID); err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
	}
}

// DELETE /api/v1/action-ui/production/scanin-draft
func (h *HTTPHandler) DeleteScanInDraft(ctx *app.Context) *app.CostumeResponse {
	var req dto.DeleteScanInDraftRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusBadRequest,
			Message:   "invalid request body: " + err.Error(),
		}
	}
	if err := h.svc.DeleteScanInDraft(ctx.Request.Context(), req); err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "OK",
	}
}
