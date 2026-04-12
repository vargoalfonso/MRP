// Package service implements business logic for the PO Budget module.
package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/po_budget/models"
	"github.com/ganasa18/go-template/internal/po_budget/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
)

// IService is the business-logic contract for PO Budget.
type IService interface {
	// Entries
	ListEntries(ctx context.Context, q models.ListBudgetQuery) (*models.ListEntryResponse, error)
	CreateEntry(ctx context.Context, req models.CreateEntryRequest, createdBy string) (*models.EntryResponse, error)
	GetEntry(ctx context.Context, id int64) (*models.EntryResponse, error)
	GetEntryDetail(ctx context.Context, budgetType string, id int64) (*models.EntryDetailResponse, error)
	UpdateEntry(ctx context.Context, id int64, req models.UpdateEntryRequest, updatedBy string) (*models.EntryResponse, error)
	DeleteEntry(ctx context.Context, id int64) error

	// Aggregated (default dashboard view per period)
	ListAggregated(ctx context.Context, q models.ListBudgetQuery) (*models.AggregatedResponse, error)
	// Summary cards
	GetSummary(ctx context.Context, budgetType, period string) (*models.SummaryResponse, error)

	// Clear
	ClearEntries(ctx context.Context, req models.ClearRequest) error

	// Approval
	ApproveEntry(ctx context.Context, id int64, req models.ApproveRequest) (*models.EntryResponse, error)

	// Split settings
	ListSplitSettings(ctx context.Context) ([]models.SplitSettingResponse, error)
	UpdateSplitSetting(ctx context.Context, budgetType string, req models.UpdateSplitSettingRequest) (*models.SplitSettingResponse, error)

	// Excel bulk import
	ImportEntries(ctx context.Context, budgetType, period string, rows []models.ImportRow, createdBy string) (*models.ImportResult, error)

	// Bulk from PRL (wizard 3-step)
	// Step 1 data: GET /prl (list) and GET /prl/:id (detail with items + allocation)
	ListPRL(ctx context.Context, customerCode string, period string, page, limit int) (*models.ListPrlResponse, error)
	GetPRLWithAllocation(ctx context.Context, prlID string, budgetType string) (*models.PrlForecastResponse, error)
	// Step 2+3: create entries; enforces quantity ceiling per PRL item
	BulkCreateFromPRL(ctx context.Context, budgetType string, req models.BulkFromPRLRequest, createdBy string) (*models.BulkFromPRLResult, error)

	// Robot split preview
	// source="manual" → { robot: false }
	// source="robot"  → call external robot URL → { robot: true, po1_pct, po2_pct }
	GetRobotSplit(ctx context.Context, budgetType string, req models.RobotSplitRequest) (*models.RobotSplitResponse, error)
}

type svc struct {
	repo     repository.IRepository
	robotURL string // ROBOT_SPLIT_URL env
}

func New(r repository.IRepository, robotURL string) IService {
	return &svc{repo: r, robotURL: robotURL}
}

// ---------------------------------------------------------------------------
// List entries
// ---------------------------------------------------------------------------

func (s *svc) ListEntries(ctx context.Context, q models.ListBudgetQuery) (*models.ListEntryResponse, error) {
	rows, total, err := s.repo.ListEntries(ctx, repository.ListFilter{
		BudgetType:     q.BudgetType,
		BudgetSubtype:  q.BudgetSubtype,
		UniqCode:       q.UniqCode,
		CustomerID:     q.CustomerID,
		Period:         q.Period,
		Status:         q.Status,
		Search:         q.Search,
		Page:           q.Page,
		Limit:          q.Limit,
		OrderBy:        q.OrderBy,
		OrderDirection: q.OrderDirection,
	})
	if err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}

	items := make([]models.EntryResponse, len(rows))
	for i, r := range rows {
		items[i] = toResponse(r)
	}

	totalPages := 0
	if q.Limit > 0 {
		totalPages = int((total + int64(q.Limit) - 1) / int64(q.Limit))
	}
	return &models.ListEntryResponse{
		Items: items,
		Meta: models.ListMeta{
			Total:      total,
			Page:       q.Page,
			Limit:      q.Limit,
			TotalPages: totalPages,
		},
	}, nil
}

// ---------------------------------------------------------------------------
// Create entry
// ---------------------------------------------------------------------------

func (s *svc) CreateEntry(ctx context.Context, req models.CreateEntryRequest, createdBy string) (*models.EntryResponse, error) {
	// Resolve PO split percentages
	po1Pct, po2Pct, err := s.resolveSplit(ctx, req.BudgetType, req.Po1Pct, req.Po2Pct)
	if err != nil {
		return nil, err
	}

	subtype := "regular"
	if req.BudgetSubtype != nil {
		subtype = *req.BudgetSubtype
	}

	periodDate, err := parsePeriod(req.Period)
	if err != nil {
		return nil, apperror.BadRequest("invalid period format, use 'Month YYYY', e.g. 'October 2025'")
	}

	e := models.POBudgetEntry{
		BudgetType:      req.BudgetType,
		BudgetSubtype:   &subtype,
		CustomerID:      req.CustomerID,
		CustomerName:    req.CustomerName,
		UniqCode:        req.UniqCode,
		ProductModel:    req.ProductModel,
		MaterialType:    req.MaterialType,
		PartName:        req.PartName,
		PartNumber:      req.PartNumber,
		Quantity:        req.Quantity,
		Uom:             req.Uom,
		WeightKg:        req.WeightKg,
		Description:     req.Description,
		SupplierID:      req.SupplierID,
		SupplierName:    req.SupplierName,
		Period:          req.Period,
		PeriodDate:      periodDate,
		SalesPlan:       req.SalesPlan,
		PurchaseRequest: req.PurchaseRequest,
		Po1Pct:          po1Pct,
		Po2Pct:          po2Pct,
		Prl:             req.Prl,
		Status:          "Pending",
		CreatedBy:       &createdBy,
	}

	if err := s.repo.CreateEntry(ctx, &e); err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}

	// History: Created + Submitted
	cb := createdBy
	if err := s.repo.CreateLog(ctx, &models.POBudgetEntryLog{EntryID: e.ID, Action: "Created", Username: &cb, Notes: strPtr("Initial creation")}); err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}
	if err := s.repo.CreateLog(ctx, &models.POBudgetEntryLog{EntryID: e.ID, Action: "Submitted", Username: &cb, Notes: strPtr("Submitted for approval")}); err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}

	// Re-fetch to get generated columns
	created, err := s.repo.GetEntryByID(ctx, e.ID)
	if err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}
	resp := toResponse(*created)
	return &resp, nil
}

// ---------------------------------------------------------------------------
// Get single entry
// ---------------------------------------------------------------------------

func (s *svc) GetEntry(ctx context.Context, id int64) (*models.EntryResponse, error) {
	e, err := s.repo.GetEntryByID(ctx, id)
	if err != nil {
		return nil, apperror.NotFound(fmt.Sprintf("entry %d not found", id))
	}
	resp := toResponse(*e)
	return &resp, nil
}

func (s *svc) GetEntryDetail(ctx context.Context, budgetType string, id int64) (*models.EntryDetailResponse, error) {
	e, err := s.repo.GetEntryByID(ctx, id)
	if err != nil {
		return nil, apperror.NotFound(fmt.Sprintf("entry %d not found", id))
	}
	if e.BudgetType != budgetType {
		return nil, apperror.NotFound(fmt.Sprintf("entry %d not found", id))
	}
	entry := toResponse(*e)

	summary, err := s.repo.GetSummary(ctx, budgetType, e.Period)
	if err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}
	logs, err := s.repo.ListLogsByEntryID(ctx, id)
	if err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}

	// Resolve uid -> username for display.
	var uids []string
	if e.CreatedBy != nil {
		uids = append(uids, *e.CreatedBy)
	}
	if e.ApprovedBy != nil {
		uids = append(uids, *e.ApprovedBy)
	}
	for _, l := range logs {
		if l.Username != nil {
			uids = append(uids, *l.Username)
		}
	}
	uidToName, err := s.repo.ResolveUsernames(ctx, uids)
	if err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}
	if entry.CreatedBy != nil {
		if name, ok := uidToName[*entry.CreatedBy]; ok {
			entry.CreatedByName = &name
		}
	}
	if entry.ApprovedBy != nil {
		if name, ok := uidToName[*entry.ApprovedBy]; ok {
			entry.ApprovedByName = &name
		}
	}
	if entry.SubmittedBy != nil {
		if name, ok := uidToName[*entry.SubmittedBy]; ok {
			entry.SubmittedByName = &name
		}
	}

	hist := make([]models.HistoryLogItem, len(logs))
	for i, l := range logs {
		uid := l.Username
		var display *string
		if uid != nil {
			if name, ok := uidToName[*uid]; ok {
				display = &name
			} else {
				display = uid
			}
		}
		hist[i] = models.HistoryLogItem{
			DateTime: l.CreatedAt.Format(time.RFC3339),
			Action:   l.Action,
			User:     display,
			UserID:   uid,
			Notes:    l.Notes,
		}
	}
	return &models.EntryDetailResponse{Entry: entry, Summary: *summary, History: hist}, nil
}

// ---------------------------------------------------------------------------
// Update entry
// ---------------------------------------------------------------------------

func (s *svc) UpdateEntry(ctx context.Context, id int64, req models.UpdateEntryRequest, updatedBy string) (*models.EntryResponse, error) {
	e, err := s.repo.GetEntryByID(ctx, id)
	if err != nil {
		return nil, apperror.NotFound(fmt.Sprintf("entry %d not found", id))
	}

	// Apply partial updates
	if req.BudgetSubtype != nil {
		e.BudgetSubtype = req.BudgetSubtype
	}
	if req.CustomerID != nil {
		e.CustomerID = req.CustomerID
	}
	if req.CustomerName != nil {
		e.CustomerName = req.CustomerName
	}
	if req.ProductModel != nil {
		e.ProductModel = req.ProductModel
	}
	if req.MaterialType != nil {
		e.MaterialType = req.MaterialType
	}
	if req.PartName != nil {
		e.PartName = req.PartName
	}
	if req.PartNumber != nil {
		e.PartNumber = req.PartNumber
	}
	if req.Quantity != nil {
		e.Quantity = *req.Quantity
	}
	if req.Uom != nil {
		e.Uom = req.Uom
	}
	if req.WeightKg != nil {
		e.WeightKg = req.WeightKg
	}
	if req.Description != nil {
		e.Description = req.Description
	}
	if req.SupplierID != nil {
		e.SupplierID = req.SupplierID
	}
	if req.SupplierName != nil {
		e.SupplierName = req.SupplierName
	}
	if req.SalesPlan != nil {
		e.SalesPlan = *req.SalesPlan
	}
	if req.PurchaseRequest != nil {
		e.PurchaseRequest = *req.PurchaseRequest
	}
	if req.Po1Pct != nil {
		e.Po1Pct = *req.Po1Pct
	}
	if req.Po2Pct != nil {
		e.Po2Pct = *req.Po2Pct
	}
	if req.Prl != nil {
		e.Prl = *req.Prl
	}
	if req.Status != nil {
		e.Status = *req.Status
	}
	e.UpdatedBy = &updatedBy

	if err := s.repo.UpdateEntry(ctx, e); err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}
	ub := updatedBy
	_ = s.repo.CreateLog(ctx, &models.POBudgetEntryLog{EntryID: id, Action: "Updated", Username: &ub, Notes: strPtr("Updated budget details")})

	updated, err := s.repo.GetEntryByID(ctx, id)
	if err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}
	resp := toResponse(*updated)
	return &resp, nil
}

// ---------------------------------------------------------------------------
// Delete single entry
// ---------------------------------------------------------------------------

func (s *svc) DeleteEntry(ctx context.Context, id int64) error {
	if _, err := s.repo.GetEntryByID(ctx, id); err != nil {
		return apperror.NotFound(fmt.Sprintf("entry %d not found", id))
	}
	if err := s.repo.DeleteEntry(ctx, id); err != nil {
		return apperror.InternalWrap("database error", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Aggregated view
// ---------------------------------------------------------------------------

func (s *svc) ListAggregated(ctx context.Context, q models.ListBudgetQuery) (*models.AggregatedResponse, error) {
	rows, total, err := s.repo.ListAggregated(ctx, repository.AggFilter{
		BudgetType:    q.BudgetType,
		BudgetSubtype: q.BudgetSubtype,
		UniqCode:      q.UniqCode,
		CustomerID:    q.CustomerID,
		Period:        q.Period,
		Page:          q.Page,
		Limit:         q.Limit,
	})
	if err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}
	for i := range rows {
		rows[i].Uniq = rows[i].UniqCode
	}

	totalPages := 0
	if q.Limit > 0 {
		totalPages = int((total + int64(q.Limit) - 1) / int64(q.Limit))
	}
	return &models.AggregatedResponse{
		Items: rows,
		Meta: models.ListMeta{
			Total:      total,
			Page:       q.Page,
			Limit:      q.Limit,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *svc) GetSummary(ctx context.Context, budgetType, period string) (*models.SummaryResponse, error) {
	res, err := s.repo.GetSummary(ctx, budgetType, period)
	if err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}
	return res, nil
}

// ---------------------------------------------------------------------------
// Clear entries
// ---------------------------------------------------------------------------

func (s *svc) ClearEntries(ctx context.Context, req models.ClearRequest) error {
	if len(req.IDs) > 0 {
		if err := s.repo.DeleteEntriesByIDs(ctx, req.IDs); err != nil {
			return apperror.InternalWrap("database error", err)
		}
		return nil
	}
	if err := s.repo.DeleteEntriesByFilter(ctx, req.BudgetType, req.Period); err != nil {
		return apperror.InternalWrap("database error", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Approval
// ---------------------------------------------------------------------------

func (s *svc) ApproveEntry(ctx context.Context, id int64, req models.ApproveRequest) (*models.EntryResponse, error) {
	e, err := s.repo.GetEntryByID(ctx, id)
	if err != nil {
		return nil, apperror.NotFound(fmt.Sprintf("entry %d not found", id))
	}

	now := time.Now()
	e.Status = req.Status
	e.ApprovedBy = &req.ApprovedBy
	e.ApprovedAt = &now

	if err := s.repo.UpdateEntry(ctx, e); err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}
	user := req.ApprovedBy
	notes := req.Notes
	// Best-effort history log
	_ = s.repo.CreateLog(ctx, &models.POBudgetEntryLog{EntryID: id, Action: req.Status, Username: &user, Notes: notes})

	updated, err := s.repo.GetEntryByID(ctx, id)
	if err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}
	resp := toResponse(*updated)
	return &resp, nil
}

// ---------------------------------------------------------------------------
// Split settings
// ---------------------------------------------------------------------------

func (s *svc) ListSplitSettings(ctx context.Context) ([]models.SplitSettingResponse, error) {
	rows, err := s.repo.ListSplitSettings(ctx)
	if err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}
	out := make([]models.SplitSettingResponse, len(rows))
	for i, r := range rows {
		out[i] = toSettingResponse(r)
	}
	return out, nil
}

func (s *svc) UpdateSplitSetting(ctx context.Context, budgetType string, req models.UpdateSplitSettingRequest) (*models.SplitSettingResponse, error) {
	setting, err := s.repo.GetSplitSetting(ctx, budgetType)
	if err != nil {
		return nil, apperror.NotFound(fmt.Sprintf("split setting for %q not found", budgetType))
	}

	// Update PO1/PO2 only if both provided.
	if req.Po1Pct != nil || req.Po2Pct != nil {
		if req.Po1Pct == nil || req.Po2Pct == nil {
			return nil, apperror.BadRequest("po1_pct and po2_pct must be provided together")
		}
		if *req.Po1Pct+*req.Po2Pct != 100 {
			return nil, apperror.BadRequest("po1_pct + po2_pct must equal 100")
		}
		setting.Po1Pct = *req.Po1Pct
		setting.Po2Pct = *req.Po2Pct
	}
	if req.MinOrderQty != nil {
		setting.MinOrderQty = req.MinOrderQty
	}
	if req.MaxSplitLines != nil {
		setting.MaxSplitLines = req.MaxSplitLines
	}
	if req.SplitRule != nil {
		setting.SplitRule = req.SplitRule
	}
	if req.Status != nil {
		setting.Status = *req.Status
	}
	if req.Description != nil {
		setting.Description = *req.Description
	}

	if err := s.repo.UpdateSplitSetting(ctx, setting); err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}
	resp := toSettingResponse(*setting)
	return &resp, nil
}

// ---------------------------------------------------------------------------
// Excel bulk import
// ---------------------------------------------------------------------------

func (s *svc) ImportEntries(ctx context.Context, budgetType, period string, rows []models.ImportRow, createdBy string) (*models.ImportResult, error) {
	periodDate, err := parsePeriod(period)
	if err != nil {
		return nil, apperror.BadRequest("invalid period format, use 'Month YYYY', e.g. 'October 2025'")
	}

	// Fetch split setting once for this budget_type
	setting, _ := s.repo.GetSplitSetting(ctx, budgetType)
	po1Pct, po2Pct := 60.0, 40.0
	if setting != nil {
		po1Pct = setting.Po1Pct
		po2Pct = setting.Po2Pct
	}

	result := &models.ImportResult{}
	var entries []models.POBudgetEntry

	for i, row := range rows {
		if row.UniqCode == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: uniq_code is required", i+1))
			result.Skipped++
			continue
		}
		cb := createdBy
		entries = append(entries, models.POBudgetEntry{
			BudgetType:      budgetType,
			CustomerName:    strPtr(row.CustomerName),
			UniqCode:        row.UniqCode,
			ProductModel:    strPtr(row.ProductModel),
			MaterialType:    strPtr(row.MaterialType),
			PartName:        strPtr(row.PartName),
			PartNumber:      strPtr(row.PartNumber),
			Quantity:        row.Quantity,
			Uom:             strPtr(row.Uom),
			WeightKg:        &row.WeightKg,
			Description:     strPtr(row.Description),
			SupplierName:    strPtr(row.SupplierName),
			Period:          period,
			PeriodDate:      periodDate,
			SalesPlan:       row.SalesPlan,
			PurchaseRequest: row.PurchaseRequest,
			Po1Pct:          po1Pct,
			Po2Pct:          po2Pct,
			Prl:             row.Prl,
			Status:          "Pending",
			CreatedBy:       &cb,
		})
	}

	if len(entries) > 0 {
		if err := s.repo.BulkCreateEntries(ctx, entries); err != nil {
			return nil, apperror.InternalWrap("database error", err)
		}
		// Best-effort logs
		for _, e := range entries {
			cb := createdBy
			_ = s.repo.CreateLog(ctx, &models.POBudgetEntryLog{EntryID: e.ID, Action: "Created", Username: &cb, Notes: strPtr("Bulk from PRL")})
			_ = s.repo.CreateLog(ctx, &models.POBudgetEntryLog{EntryID: e.ID, Action: "Submitted", Username: &cb, Notes: strPtr("Submitted for approval")})
		}
	}

	result.Imported = len(entries)
	return result, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// resolveSplit returns the po1/po2 percentages to use, falling back to
// the stored po_split_settings for the given budget_type.
func (s *svc) resolveSplit(ctx context.Context, budgetType string, po1 *float64, po2 *float64) (float64, float64, error) {
	if po1 != nil && po2 != nil {
		if *po1+*po2 != 100 {
			return 0, 0, apperror.BadRequest("po1_pct + po2_pct must equal 100")
		}
		return *po1, *po2, nil
	}
	setting, err := s.repo.GetSplitSetting(ctx, budgetType)
	if err != nil {
		// fallback default
		return 60, 40, nil
	}
	return setting.Po1Pct, setting.Po2Pct, nil
}

// parsePeriod parses period in two formats:
//   - "October 2025"  (month name + year)
//   - "10-2025"       (MM-YYYY numeric)
func parsePeriod(period string) (time.Time, error) {
	// Try MM-YYYY numeric format first
	if strings.Contains(period, "-") {
		parts := strings.SplitN(period, "-", 2)
		if len(parts) == 2 {
			monthNum, err1 := strconv.Atoi(parts[0])
			year, err2 := strconv.Atoi(parts[1])
			if err1 == nil && err2 == nil && monthNum >= 1 && monthNum <= 12 {
				return time.Date(year, time.Month(monthNum), 1, 0, 0, 0, 0, time.UTC), nil
			}
		}
	}
	// Fall back to "Month YYYY" text format
	parts := strings.Fields(period)
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid period %q: use 'October 2025' or '10-2025'", period)
	}
	year, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid year in period: %q", period)
	}
	month, err := parseMonthName(parts[0])
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC), nil
}

func parseMonthName(name string) (time.Month, error) {
	months := map[string]time.Month{
		"january": time.January, "february": time.February, "march": time.March,
		"april": time.April, "may": time.May, "june": time.June,
		"july": time.July, "august": time.August, "september": time.September,
		"october": time.October, "november": time.November, "december": time.December,
	}
	m, ok := months[strings.ToLower(name)]
	if !ok {
		return 0, fmt.Errorf("unknown month: %q", name)
	}
	return m, nil
}

func toResponse(e models.POBudgetEntry) models.EntryResponse {
	po1 := e.PurchaseRequest * e.Po1Pct / 100
	po2 := e.PurchaseRequest * e.Po2Pct / 100
	total := po1 + po2
	// Use GENERATED values if populated by DB
	if e.Po1Qty != 0 {
		po1 = e.Po1Qty
	}
	if e.Po2Qty != 0 {
		po2 = e.Po2Qty
	}
	if e.TotalPO != 0 {
		total = e.TotalPO
	}
	apoPrl := total - e.Prl
	state := "match"
	if apoPrl > 0 {
		state = "over"
	} else if apoPrl < 0 {
		state = "under"
	}
	return models.EntryResponse{
		ID:              e.ID,
		PoBudgetRef:     firstNonEmpty(e.PoBudgetRef, poBudgetRef(e)),
		BudgetType:      e.BudgetType,
		CustomerID:      e.CustomerID,
		CustomerName:    e.CustomerName,
		Uniq:            e.UniqCode,
		UniqCode:        e.UniqCode,
		ProductModel:    e.ProductModel,
		MaterialType:    e.MaterialType,
		PartName:        e.PartName,
		PartNumber:      e.PartNumber,
		Quantity:        e.Quantity,
		Uom:             e.Uom,
		WeightKg:        e.WeightKg,
		Description:     e.Description,
		SupplierID:      e.SupplierID,
		SupplierName:    e.SupplierName,
		Period:          e.Period,
		PeriodDate:      e.PeriodDate,
		SalesPlan:       e.SalesPlan,
		PurchaseRequest: e.PurchaseRequest,
		Po1Pct:          e.Po1Pct,
		Po2Pct:          e.Po2Pct,
		Po1Qty:          po1,
		Po2Qty:          po2,
		TotalPO:         total,
		Po1Amount:       po1,
		Po2Amount:       po2,
		TotalPOAmount:   total,
		ApoPrlAmount:    apoPrl,
		ApoPrlState:     state,
		Prl:             e.Prl,
		DeltaApoPrl:     apoPrl,
		Status:          e.Status,
		BudgetSubtype:   e.BudgetSubtype,
		PrlRef:          e.PrlRef,
		PrlRowID:        e.PrlRowID,
		ApprovedBy:      e.ApprovedBy,
		ApprovedAt:      e.ApprovedAt,
		SubmittedBy:     e.CreatedBy,
		SubmittedAt:     e.CreatedAt,
		CreatedBy:       e.CreatedBy,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func poBudgetRef(e models.POBudgetEntry) string {
	// Use the budget period year for easier tracing.
	y := e.PeriodDate.Year()
	if y <= 0 {
		y = time.Now().Year()
	}

	code := "UNK"
	switch strings.ToLower(strings.TrimSpace(e.BudgetType)) {
	case "raw_material":
		code = "RM"
	case "subcon":
		code = "SC"
	case "indirect":
		code = "IB"
	}

	// id is already unique; we just format it nicely.
	return fmt.Sprintf("POB-%04d-%s-%06d", y, code, e.ID)
}

func toSettingResponse(s models.POSplitSetting) models.SplitSettingResponse {
	return models.SplitSettingResponse{
		ID:            s.ID,
		BudgetType:    s.BudgetType,
		Po1Pct:        s.Po1Pct,
		Po2Pct:        s.Po2Pct,
		MinOrderQty:   s.MinOrderQty,
		MaxSplitLines: s.MaxSplitLines,
		SplitRule:     s.SplitRule,
		Status:        s.Status,
		Description:   s.Description,
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// normalizeForecastPeriod converts UI period ("October 2025") to data-asli format.
// Supported data-asli formats:
// - YYYY-QN (e.g. 2026-Q2)
// - MM-YYYY (e.g. 10-2025)
//
// If input is Month YYYY, we convert it to YYYY-QN (quarter) because many
// PRL sources store forecast_period as quarters.
func normalizeForecastPeriod(p string) (string, error) {
	if p == "" {
		return "", nil
	}
	// YYYY-QN
	if len(p) == 7 && p[4] == '-' && (p[5] == 'Q' || p[5] == 'q') {
		return strings.ToUpper(p), nil
	}
	// MM-YYYY
	if len(p) == 7 && p[2] == '-' {
		return p, nil
	}
	// Month YYYY
	periodDate, err := parsePeriod(p)
	if err != nil {
		return "", err
	}
	q := (int(periodDate.Month())-1)/3 + 1
	return fmt.Sprintf("%04d-Q%d", periodDate.Year(), q), nil
}

// displayFromForecastPeriod converts known PRL forecast_period formats to a UI label.
func displayFromForecastPeriod(fp string) string {
	// YYYY-QN
	if len(fp) == 7 && fp[4] == '-' && (fp[5] == 'Q' || fp[5] == 'q') {
		yyyy := fp[0:4]
		q := fp[6:7]
		return fmt.Sprintf("Q%s %s", q, yyyy)
	}
	if len(fp) != 7 || fp[2] != '-' {
		return fp
	}
	mm, err1 := strconv.Atoi(fp[0:2])
	yy, err2 := strconv.Atoi(fp[3:7])
	if err1 != nil || err2 != nil || mm < 1 || mm > 12 {
		return fp
	}
	month := time.Month(mm)
	return fmt.Sprintf("%s %d", month.String(), yy)
}

// ---------------------------------------------------------------------------
// PRL — list + detail with allocation
// ---------------------------------------------------------------------------

func (s *svc) ListPRL(ctx context.Context, customerCode string, period string, page, limit int) (*models.ListPrlResponse, error) {
	forecastPeriod, err := normalizeForecastPeriod(period)
	if err != nil {
		return nil, apperror.BadRequest("invalid period format; use 'Month YYYY' (e.g. 'October 2025') or 'MM-YYYY' (e.g. '10-2025')")
	}
	rows, total, err := s.repo.ListPRL(ctx, customerCode, forecastPeriod, page, limit)
	if err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}
	items := make([]models.PrlForecastResponse, len(rows))
	for i, r := range rows {
		items[i] = models.PrlForecastResponse{
			ID:           r.PrlID,
			PrlNumber:    r.PrlID,
			CustomerID:   nil,
			CustomerName: r.CustomerName,
			Period:       displayFromForecastPeriod(ptrVal(r.ForecastPeriod)),
			Status:       ptrVal(r.Status),
		}
	}
	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}
	return &models.ListPrlResponse{
		Items: items,
		Meta:  models.ListMeta{Total: total, Page: page, Limit: limit, TotalPages: totalPages},
	}, nil
}

// GetPRLWithAllocation returns a PRL header + items each annotated with how
// much quantity is already allocated in po_budget_entries for budgetType.
// UI uses this to display "Budget: X | Allocated: Y" in the Add Supplier modal.

func ptrVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (s *svc) GetPRLWithAllocation(ctx context.Context, prlID string, budgetType string) (*models.PrlForecastResponse, error) {
	prl, err := s.repo.GetPRLDocByPrlID(ctx, prlID)
	if err != nil {
		return nil, apperror.NotFound(fmt.Sprintf("PRL %s not found", prlID))
	}
	items, err := s.repo.GetPRLRowsByPrlID(ctx, prlID)
	if err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}

	// Allocation requires po_budget_entries.prl_row_id (migration 0011).
	has, err := s.repo.HasColumn(ctx, "po_budget_entries", "prl_row_id")
	if err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}
	if !has {
		return nil, apperror.BadRequest("missing DB column po_budget_entries.prl_row_id; run migration scripts/migrations/0011_po_budget_link_prls_up.sql")
	}

	// Batch-fetch allocations for all items in one query
	itemIDs := make([]int64, len(items))
	for i, it := range items {
		itemIDs[i] = it.ID
	}
	allocMap, err := s.repo.SumAllocatedQtyBatch(ctx, itemIDs, budgetType)
	if err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}

	resp := models.PrlForecastResponse{
		ID:           prl.PrlID,
		PrlNumber:    prl.PrlID,
		CustomerID:   nil,
		CustomerName: prl.CustomerName,
		Period:       displayFromForecastPeriod(ptrVal(prl.ForecastPeriod)),
		Status:       ptrVal(prl.Status),
	}
	resp.Items = make([]models.PrlForecastItemResponse, len(items))
	for i, it := range items {
		allocated := allocMap[it.ID]
		resp.Items[i] = models.PrlForecastItemResponse{
			ID:           it.ID,
			UniqCode:     ptrVal(it.UniqCode),
			ProductModel: it.ProductModel,
			PartName:     it.PartName,
			PartNumber:   it.PartNumber,
			WeightKg:     nil,
			Quantity:     it.Quantity,
			AllocatedQty: allocated,
			RemainingQty: it.Quantity - allocated,
			Uom:          nil,
		}
	}
	return &resp, nil
}

// ---------------------------------------------------------------------------
// BulkCreateFromPRL — wizard steps 2+3
// ---------------------------------------------------------------------------

// BulkCreateFromPRL validates that no supplier allocation exceeds the PRL item
// quantity ceiling, then creates one po_budget_entry row per supplier per UNIQ.
//
// Ceiling rule (per item):
//
//	existing_allocated + SUM(new suppliers for this item) ≤ item.Quantity
func (s *svc) BulkCreateFromPRL(ctx context.Context, budgetType string, req models.BulkFromPRLRequest, createdBy string) (*models.BulkFromPRLResult, error) {
	periodDate, err := parsePeriod(req.Period)
	if err != nil {
		return nil, apperror.BadRequest("invalid period format, use 'Month YYYY', e.g. 'October 2025'")
	}

	// Validate PRL exists
	prl, err := s.repo.GetPRLDocByPrlID(ctx, req.PrlID)
	if err != nil {
		return nil, apperror.NotFound(fmt.Sprintf("PRL %s not found", req.PrlID))
	}

	// Allocation requires po_budget_entries.prl_row_id (migration 0011).
	has, err := s.repo.HasColumn(ctx, "po_budget_entries", "prl_row_id")
	if err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}
	if !has {
		return nil, apperror.BadRequest("missing DB column po_budget_entries.prl_row_id; run migration scripts/migrations/0011_po_budget_link_prls_up.sql")
	}

	// Load PRL rows for requested item IDs and validate they belong to the PRL doc.
	itemIDs := make([]int64, len(req.Items))
	for i, it := range req.Items {
		itemIDs[i] = it.PrlItemID
	}
	prlRows, err := s.repo.GetPRLRowsByIDs(ctx, itemIDs)
	if err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}
	rowMap := make(map[int64]models.PRLRow, len(prlRows))
	for _, r := range prlRows {
		rowMap[r.ID] = r
	}
	for _, it := range req.Items {
		r, ok := rowMap[it.PrlItemID]
		if !ok {
			return nil, apperror.BadRequest(fmt.Sprintf("prl_item_id %d not found", it.PrlItemID))
		}
		if r.PrlID != req.PrlID {
			return nil, apperror.BadRequest(fmt.Sprintf("prl_item_id %d does not belong to prl_id %s (belongs to %s)", it.PrlItemID, req.PrlID, r.PrlID))
		}
	}

	// Batch-fetch existing allocations for all requested prl_row_ids
	allocMap, err := s.repo.SumAllocatedQtyBatch(ctx, itemIDs, budgetType)
	if err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}

	result := &models.BulkFromPRLResult{}
	var entries []models.POBudgetEntry

	prlRef := req.PrlID
	subtype := req.BudgetSubtype

	for _, item := range req.Items {
		r := rowMap[item.PrlItemID]
		budgetQty := r.Quantity
		uniqCode := ptrVal(r.UniqCode)
		if uniqCode == "" {
			uniqCode = item.UniqCode
		}
		productModel := r.ProductModel
		if productModel == nil {
			productModel = item.ProductModel
		}
		partName := r.PartName
		if partName == nil {
			partName = item.PartName
		}
		partNumber := r.PartNumber
		if partNumber == nil {
			partNumber = item.PartNumber
		}

		// Resolve PO split per item (falls back to po_split_settings)
		po1Pct, po2Pct, err := s.resolveSplit(ctx, budgetType, item.Po1Pct, item.Po2Pct)
		if err != nil {
			return nil, err
		}

		// Validate: sum of new supplier quantities for this item
		var newTotal float64
		for _, sup := range item.Suppliers {
			newTotal += sup.Quantity
		}

		existingAllocated := allocMap[item.PrlItemID]
		if existingAllocated+newTotal > budgetQty {
			result.Errors = append(result.Errors,
				fmt.Sprintf("uniq %s: total allocated (%.2f existing + %.2f new = %.2f) exceeds PRL budget (%.2f)",
					uniqCode, existingAllocated, newTotal, existingAllocated+newTotal, budgetQty),
			)
			continue
		}

		// Create one entry per supplier
		for _, sup := range item.Suppliers {
			cb := createdBy
			prlRowID := item.PrlItemID

			entries = append(entries, models.POBudgetEntry{
				BudgetType:      budgetType,
				CustomerID:      nil,
				CustomerName:    prl.CustomerName,
				UniqCode:        uniqCode,
				ProductModel:    productModel,
				MaterialType:    nil,
				PartName:        partName,
				PartNumber:      partNumber,
				Quantity:        sup.Quantity,
				Uom:             item.Uom,
				WeightKg:        item.WeightKg,
				SupplierID:      sup.SupplierID,
				SupplierName:    strPtr(sup.SupplierName),
				Period:          req.Period,
				PeriodDate:      periodDate,
				SalesPlan:       item.SalesPlan,
				PurchaseRequest: sup.Quantity, // PR = supplier's allocated qty
				Po1Pct:          po1Pct,
				Po2Pct:          po2Pct,
				Prl:             budgetQty,
				PrlRef:          &prlRef,
				PrlRowID:        &prlRowID,
				BudgetSubtype:   &subtype,
				Status:          "Pending",
				CreatedBy:       &cb,
			})
		}
	}

	if len(entries) > 0 {
		if err := s.repo.BulkCreateEntries(ctx, entries); err != nil {
			return nil, apperror.InternalWrap("database error", err)
		}
	}

	result.Created = len(entries)
	return result, nil
}

func toPrlResponse(p models.PrlForecast) models.PrlForecastResponse {
	return models.PrlForecastResponse{
		ID:           fmt.Sprintf("%d", p.ID),
		PrlNumber:    p.PrlNumber,
		CustomerID:   p.CustomerID,
		CustomerName: p.CustomerName,
		Period:       p.Period,
		Status:       p.Status,
	}
}

// ---------------------------------------------------------------------------
// GetRobotSplit
// ---------------------------------------------------------------------------

// robotSplitPayload is the request body sent to the external robot service.
type robotSplitPayload struct {
	UniqCode string `json:"uniq_code"`
}

// robotSplitResult is the expected response from the external robot service.
type robotSplitResult struct {
	Po1Pct float64 `json:"po1_pct"`
	Po2Pct float64 `json:"po2_pct"`
}

func (s *svc) GetRobotSplit(ctx context.Context, budgetType string, req models.RobotSplitRequest) (*models.RobotSplitResponse, error) {
	if req.PoType == "manual" {
		return &models.RobotSplitResponse{Robot: false}, nil
	}

	// TODO: replace with real robot HTTP call when ROBOT_SPLIT_URL is ready.
	// Mock response — hardcoded split 60/40.
	return &models.RobotSplitResponse{
		Robot:  true,
		Po1Pct: 60,
		Po2Pct: 40,
	}, nil
}
