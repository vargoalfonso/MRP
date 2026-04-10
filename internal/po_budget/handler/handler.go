// Package handler exposes PO Budget HTTP endpoints.
//
// Parent menu: Sales
// Sub menus  : Raw Material Budget (/raw-material), Subcon (/subcon), Indirect (/indirect)
//
// Routes (prefix /api/v1/sales):
//
//	GET    /{type}/budget              list entries (paginated)
//	GET    /{type}/budget/aggregate    aggregated view by Uniq + Period
//	POST   /{type}/budget              create entry
//	PUT    /{type}/budget/:id          update entry (partial)
//	DELETE /{type}/budget/:id          delete single entry
//	POST   /{type}/budget/clear        clear entries (bulk by IDs or filter)
//	POST   /{type}/budget/:id/approve  approve / reject entry
//	POST   /{type}/budget/import       bulk import from Excel (multipart)
//
//	GET    /po-split-settings          list PO split settings
//	PUT    /po-split-settings/:type    update split percentage for a budget type
package handler

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	authModels "github.com/ganasa18/go-template/internal/auth/models"
	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/po_budget/models"
	"github.com/ganasa18/go-template/internal/po_budget/service"
	"github.com/ganasa18/go-template/pkg/pagination"
	"github.com/ganasa18/go-template/pkg/validator"
)

// HTTPHandler holds the service dependency for all PO Budget endpoints.
type HTTPHandler struct {
	svc service.IService
}

func New(svc service.IService) *HTTPHandler { return &HTTPHandler{svc: svc} }

// ---------------------------------------------------------------------------
// SUMMARY  GET /{type}/budget/summary
// ---------------------------------------------------------------------------

func (h *HTTPHandler) GetSummary(ctx *app.Context) *app.CostumeResponse {
	budgetType := normalizeBudgetType(ctx.Param("type"))
	period := ctx.Query("period")
	res, err := h.svc.GetSummary(ctx.Request.Context(), budgetType, period)
	if err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      res,
	}
}

// ---------------------------------------------------------------------------
// LIST entries  GET /{type}/budget
// ---------------------------------------------------------------------------

func (h *HTTPHandler) ListEntries(ctx *app.Context) *app.CostumeResponse {
	p := pagination.POBudgetPagination(ctx)
	budgetType := ctx.Param("type")

	resp, err := h.svc.ListEntries(ctx.Request.Context(), models.ListBudgetQuery{
		BudgetType:     normalizeBudgetType(budgetType),
		BudgetSubtype:  p.BudgetSubtype,
		UniqCode:       p.UniqCode,
		CustomerID:     p.CustomerID,
		Period:         p.Period,
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

// ---------------------------------------------------------------------------
// AGGREGATE view  GET /{type}/budget/aggregate
// ---------------------------------------------------------------------------

func (h *HTTPHandler) ListAggregated(ctx *app.Context) *app.CostumeResponse {
	p := pagination.POBudgetPagination(ctx)
	budgetType := ctx.Param("type")

	resp, err := h.svc.ListAggregated(ctx.Request.Context(), models.ListBudgetQuery{
		BudgetType:    normalizeBudgetType(budgetType),
		BudgetSubtype: p.BudgetSubtype,
		UniqCode:      p.UniqCode,
		CustomerID:    p.CustomerID,
		Period:        p.Period,
		Page:          p.Page,
		Limit:         p.Limit,
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

// ---------------------------------------------------------------------------
// CREATE entry  POST /{type}/budget
// ---------------------------------------------------------------------------

func (h *HTTPHandler) CreateEntry(ctx *app.Context) *app.CostumeResponse {
	budgetType := ctx.Param("type")
	var req models.CreateEntryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	req.BudgetType = normalizeBudgetType(budgetType)
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}

	createdBy := mustUserID(ctx)
	result, err := h.svc.CreateEntry(ctx.Request.Context(), req, createdBy)
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

// ---------------------------------------------------------------------------
// GET single  GET /{type}/budget/:id
// ---------------------------------------------------------------------------

func (h *HTTPHandler) GetEntry(ctx *app.Context) *app.CostumeResponse {
	id, err := parseID(ctx.Param("id"))
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	result, err := h.svc.GetEntry(ctx.Request.Context(), id)
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

// ---------------------------------------------------------------------------
// GET detail + summary  GET /{type}/budget/:id/detail
// ---------------------------------------------------------------------------

func (h *HTTPHandler) GetEntryDetail(ctx *app.Context) *app.CostumeResponse {
	id, err := parseID(ctx.Param("id"))
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	budgetType := normalizeBudgetType(ctx.Param("type"))
	result, err := h.svc.GetEntryDetail(ctx.Request.Context(), budgetType, id)
	if err != nil {
		return app.NewError(ctx, err)
	}

	if strings.EqualFold(ctx.Query("format"), "grouped") {
		e := result.Entry
		grouped := models.EntryDetailGroupedResponse{
			BasicInformation: models.EntryBasicInformation{
				ID:           e.ID,
				CustomerName: e.CustomerName,
				Uniq:         e.Uniq,
				ProductModel: e.ProductModel,
				PartName:     e.PartName,
				PartNumber:   e.PartNumber,
				SupplierName: e.SupplierName,
				BudgetType:   e.BudgetType,
				TypeLabel:    e.BudgetSubtype,
				Period:       e.Period,
			},
			BudgetCalculations: models.EntryBudgetCalculations{
				SalesPlan:       e.SalesPlan,
				PurchaseRequest: e.PurchaseRequest,
				PrlAmount:       e.Prl,
				Po1Pct:          e.Po1Pct,
				Po2Pct:          e.Po2Pct,
			},
			CalculationResults: models.EntryCalculationResults{
				Po1Amount:   e.Po1Amount,
				Po2Amount:   e.Po2Amount,
				TotalPO:     e.TotalPO,
				ApoPrlAbs:   result.Summary.ApoPrlAbs,
				ApoPrlState: result.Summary.ApoPrlState,
			},
			AdditionalInformation: models.EntryAdditionalInformation{
				SubmittedBy:     e.SubmittedBy,
				SubmittedByName: e.SubmittedByName,
				SubmittedAt:     e.SubmittedAt,
				ApprovedBy:      e.ApprovedBy,
				ApprovedByName:  e.ApprovedByName,
				ApprovedAt:      e.ApprovedAt,
				Notes:           e.Description,
			},
			Summary: result.Summary,
			History: result.History,
		}

		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusOK,
			Message:   http.StatusText(http.StatusOK),
			Data:      grouped,
		}
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   http.StatusText(http.StatusOK),
		Data:      result,
	}
}

// ---------------------------------------------------------------------------
// UPDATE entry  PUT /{type}/budget/:id
// ---------------------------------------------------------------------------

func (h *HTTPHandler) UpdateEntry(ctx *app.Context) *app.CostumeResponse {
	id, err := parseID(ctx.Param("id"))
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	var req models.UpdateEntryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}

	updatedBy := mustUserID(ctx)
	result, err := h.svc.UpdateEntry(ctx.Request.Context(), id, req, updatedBy)
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

// ---------------------------------------------------------------------------
// DELETE entry  DELETE /{type}/budget/:id
// ---------------------------------------------------------------------------

func (h *HTTPHandler) DeleteEntry(ctx *app.Context) *app.CostumeResponse {
	id, err := parseID(ctx.Param("id"))
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	if err := h.svc.DeleteEntry(ctx.Request.Context(), id); err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "entry deleted",
	}
}

// ---------------------------------------------------------------------------
// CLEAR  POST /{type}/budget/clear
// ---------------------------------------------------------------------------

func (h *HTTPHandler) ClearEntries(ctx *app.Context) *app.CostumeResponse {
	budgetType := ctx.Param("type")
	var req models.ClearRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	req.BudgetType = normalizeBudgetType(budgetType)
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	if err := h.svc.ClearEntries(ctx.Request.Context(), req); err != nil {
		return app.NewError(ctx, err)
	}
	return &app.CostumeResponse{
		RequestID: ctx.APIReqID,
		Status:    http.StatusOK,
		Message:   "entries cleared",
	}
}

// ---------------------------------------------------------------------------
// APPROVE  POST /{type}/budget/:id/approve
// ---------------------------------------------------------------------------

func (h *HTTPHandler) ApproveEntry(ctx *app.Context) *app.CostumeResponse {
	id, err := parseID(ctx.Param("id"))
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	var req models.ApproveRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// Allow empty body; infer action from URL.
		if !errors.Is(err, io.EOF) {
			return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
		}
	}
	if req.Status == "" {
		path := strings.ToLower(ctx.Request.URL.Path)
		if strings.HasSuffix(path, "/reject") {
			req.Status = "Rejected"
		} else {
			req.Status = "Approved"
		}
	}
	// Always derive from JWT uid, never trust frontend payload.
	req.ApprovedBy = mustUserID(ctx)
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	result, err := h.svc.ApproveEntry(ctx.Request.Context(), id, req)
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

// ---------------------------------------------------------------------------
// IMPORT (Excel / CSV)  POST /{type}/budget/import
//
// Accepts multipart/form-data with field "file" (CSV or .xlsx treated as CSV
// for now — frontend can export Excel as CSV before upload, or add excelize later).
//
// Expected CSV columns (header row required):
//   uniq_code, customer_name, product_model, material_type, part_name,
//   part_number, quantity, uom, weight_kg, description, supplier_name,
//   sales_plan, purchase_request, prl
//
// Query params: period=October+2025
// ---------------------------------------------------------------------------

func (h *HTTPHandler) ImportEntries(ctx *app.Context) *app.CostumeResponse {
	budgetType := normalizeBudgetType(ctx.Param("type"))
	period := ctx.Query("period")
	if period == "" {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "query param 'period' is required (e.g. October 2025)"}
	}

	file, _, err := ctx.Request.FormFile("file")
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "file field is required (multipart/form-data)"}
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: fmt.Sprintf("failed to parse CSV: %v", err)}
	}
	if len(records) < 2 {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "CSV must have a header row and at least one data row"}
	}

	header := records[0]
	colIdx := buildColIndex(header)

	var rows []models.ImportRow
	for _, rec := range records[1:] {
		rows = append(rows, models.ImportRow{
			UniqCode:        csvVal(rec, colIdx, "uniq_code"),
			CustomerName:    csvVal(rec, colIdx, "customer_name"),
			ProductModel:    csvVal(rec, colIdx, "product_model"),
			MaterialType:    csvVal(rec, colIdx, "material_type"),
			PartName:        csvVal(rec, colIdx, "part_name"),
			PartNumber:      csvVal(rec, colIdx, "part_number"),
			Quantity:        csvFloat(rec, colIdx, "quantity"),
			Uom:             csvVal(rec, colIdx, "uom"),
			WeightKg:        csvFloat(rec, colIdx, "weight_kg"),
			Description:     csvVal(rec, colIdx, "description"),
			SupplierName:    csvVal(rec, colIdx, "supplier_name"),
			SalesPlan:       csvFloat(rec, colIdx, "sales_plan"),
			PurchaseRequest: csvFloat(rec, colIdx, "purchase_request"),
			Prl:             csvFloat(rec, colIdx, "prl"),
		})
	}

	createdBy := mustUserID(ctx)
	result, err := h.svc.ImportEntries(ctx.Request.Context(), budgetType, period, rows, createdBy)
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

// ---------------------------------------------------------------------------
// PO Split Settings
// ---------------------------------------------------------------------------

func (h *HTTPHandler) ListSplitSettings(ctx *app.Context) *app.CostumeResponse {
	result, err := h.svc.ListSplitSettings(ctx.Request.Context())
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

func (h *HTTPHandler) UpdateSplitSetting(ctx *app.Context) *app.CostumeResponse {
	budgetType := ctx.Param("type")
	var req models.UpdateSplitSettingRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}
	result, err := h.svc.UpdateSplitSetting(ctx.Request.Context(), budgetType, req)
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

// ---------------------------------------------------------------------------
// PRL — list + detail with allocation (Step 1 & Step 2 data)
// ---------------------------------------------------------------------------

// ListPRL  GET /api/v1/sales/prl
//
// Query params: customer_id, period, limit, page
func (h *HTTPHandler) ListPRL(ctx *app.Context) *app.CostumeResponse {
	p := pagination.Pagination(ctx)
	// Data asli uses customer_code (varchar) not numeric customer_id.
	customerCode := ctx.Query("customer_code")
	result, err := h.svc.ListPRL(ctx.Request.Context(), customerCode, ctx.Query("period"), p.Page, p.Limit)
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

// GetPRLDetail  GET /api/v1/sales/prl/:id?budget_type=raw_material
//
// Returns PRL header + items each with budget ceiling + already-allocated qty.
// UI shows "Budget: X | Allocated: Y" per item in the Add Supplier modal.
func (h *HTTPHandler) GetPRLDetail(ctx *app.Context) *app.CostumeResponse {
	// :id is prl_id (varchar(32)) from table "prls".
	id := strings.TrimSpace(ctx.Param("id"))
	if id == "" {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid id"}
	}
	budgetType := normalizeBudgetType(ctx.Query("budget_type"))
	if budgetType == "" {
		budgetType = "raw_material"
	}
	result, err := h.svc.GetPRLWithAllocation(ctx.Request.Context(), id, budgetType)
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

// ---------------------------------------------------------------------------
// BulkCreateFromPRL  POST /api/v1/sales/:type/budget/bulk
//
// Wizard steps 2+3 in one call.
// Validates: sum of each item's supplier quantities MUST NOT exceed
// that item's PRL budget_qty (ceiling set by production planning).
// ---------------------------------------------------------------------------

func (h *HTTPHandler) BulkCreateFromPRL(ctx *app.Context) *app.CostumeResponse {
	budgetType := normalizeBudgetType(ctx.Param("type"))
	var req models.BulkFromPRLRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusBadRequest, Message: "invalid request body"}
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{RequestID: ctx.APIReqID, Status: http.StatusUnprocessableEntity, Message: "validation failed", Data: map[string]interface{}{"errors": errs}}
	}

	createdBy := mustUserID(ctx)
	result, err := h.svc.BulkCreateFromPRL(ctx.Request.Context(), budgetType, req, createdBy)
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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// normalizeBudgetType maps URL path segment to DB budget_type value.
//
//	raw-material → raw_material
//	subcon       → subcon
//	indirect     → indirect
func normalizeBudgetType(t string) string {
	return strings.ReplaceAll(strings.ToLower(t), "-", "_")
}

func parseID(s string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
}

func buildColIndex(header []string) map[string]int {
	m := make(map[string]int, len(header))
	for i, h := range header {
		m[strings.TrimSpace(strings.ToLower(h))] = i
	}
	return m
}

func csvVal(rec []string, idx map[string]int, col string) string {
	i, ok := idx[col]
	if !ok || i >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[i])
}

func csvFloat(rec []string, idx map[string]int, col string) float64 {
	v := csvVal(rec, idx, col)
	f, _ := strconv.ParseFloat(v, 64)
	return f
}

func mustUserID(ctx *app.Context) string {
	claimsRaw, ok := ctx.Get("claims")
	if !ok {
		return ""
	}
	claims, ok := claimsRaw.(*authModels.Claims)
	if !ok {
		return ""
	}
	return claims.UserID
}
