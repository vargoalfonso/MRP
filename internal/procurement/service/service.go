// Package service implements the business logic for the Procurement module.
package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/procurement/models"
	"github.com/ganasa18/go-template/internal/procurement/repository"
	"github.com/ganasa18/go-template/pkg/pagination"
)

// IService is the business-logic contract for the Procurement module.
type IService interface {
	// KPI summary card
	GetSummary(ctx context.Context, poType, period string) (*models.SummaryResponse, error)

	// PO board list (paginated)
	ListPOBoard(ctx context.Context, p pagination.POBoardPaginationInput) (*models.POBoardListResponse, error)

	// PO detail + history
	GetPODetail(ctx context.Context, poID int64) (*models.PODetailResponse, error)

	// DN list + detail
	ListDNs(ctx context.Context, f models.DNListFilter) (*models.DNListResponse, error)
	GetDNDetail(ctx context.Context, dnID int64) (*models.DNDetailResponse, error)

	// Wizard: form options dropdown data
	GetFormOptions(ctx context.Context, poType, period string) (*models.FormOptionsResponse, error)

	// Wizard: generate PO (single supplier or bulk-all-suppliers)
	GeneratePO(ctx context.Context, req models.GeneratePORequest, createdBy string) (*models.GeneratePOResponse, error)
}

// ---------------------------------------------------------------------------

type svc struct {
	repo repository.IRepository
}

// New returns a new IService.
func New(repo repository.IRepository) IService {
	return &svc{repo: repo}
}

// ---------------------------------------------------------------------------
// GetSummary
// ---------------------------------------------------------------------------

func (s *svc) GetSummary(ctx context.Context, poType, period string) (*models.SummaryResponse, error) {
	row, err := s.repo.GetSummary(ctx, poType, period) // poType = raw_material|indirect|subcon
	if err != nil {
		return nil, err
	}
	return &models.SummaryResponse{
		TotalPos:        row.TotalPos,
		ActiveSuppliers: row.ActiveSuppliers,
		TotalPoValue:    row.TotalPoValue,
		LateDeliveries:  row.LateDeliveries,
	}, nil
}

// ---------------------------------------------------------------------------
// ListPOBoard
// ---------------------------------------------------------------------------

func (s *svc) ListPOBoard(ctx context.Context, p pagination.POBoardPaginationInput) (*models.POBoardListResponse, error) {
	f := repository.POBoardFilter{
		PoType:     p.PoType,
		Period:     p.Period,
		SupplierID: p.SupplierID,
		UniqCode:   p.UniqCode,
		Status:     p.Status,
		LateOnly:   p.LateOnly,
		Search:     p.Search,
		Page:       p.Page,
		Limit:      p.Limit,
		OrderBy:    p.OrderBy,
		OrderDir:   p.OrderDirection,
	}

	rows, total, err := s.repo.ListPOBoard(ctx, f)
	if err != nil {
		return nil, err
	}

	items := make([]models.POBoardItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, models.POBoardItem{
			PoID:          r.PoID,
			PoType:        r.PoType,
			PoStage:       r.PoStage,
			Period:        r.Period,
			PoNumber:      r.PoNumber,
			TotalBudgetPo: r.TotalBudgetPo,
			QtyDelivered:  r.QtyDelivered,
			UniqCode:      r.UniqCode,
			SupplierID:    r.SupplierID,
			SupplierName:  r.SupplierName,
			Status:        r.Status,
			IsLate:        r.IsLate,
		})
	}

	totalPages := 0
	if p.Limit > 0 {
		totalPages = int((total + int64(p.Limit) - 1) / int64(p.Limit))
	}

	return &models.POBoardListResponse{
		Items: items,
		Pagination: models.PaginationMeta{
			Total:      total,
			Page:       p.Page,
			Limit:      p.Limit,
			TotalPages: totalPages,
		},
	}, nil
}

// ---------------------------------------------------------------------------
// GetPODetail
// ---------------------------------------------------------------------------

func (s *svc) GetPODetail(ctx context.Context, poID int64) (*models.PODetailResponse, error) {
	po, err := s.repo.GetPOByID(ctx, poID)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.GetPOItems(ctx, poID)
	if err != nil {
		return nil, err
	}

	logs, err := s.repo.GetPOLogs(ctx, poID)
	if err != nil {
		return nil, err
	}

	// Resolve supplier name
	var supplierName *string
	if po.SupplierID != nil && *po.SupplierID > 0 {
		if sup, serr := s.repo.GetLegacySupplier(ctx, *po.SupplierID); serr == nil {
			supplierName = &sup.SupplierName
		}
	}

	// Compute total_quantity from items
	var totalQty float64
	for _, it := range items {
		totalQty += it.OrderedQty
	}

	// Build po_budget_ref display string
	poBudgetRef := buildBudgetRef(po.PoType, po.Period, po.PoBudgetEntryID)

	header := models.POHeaderDetail{
		PoID:             po.PoID,
		PoType:           po.PoType,
		PoStage:          po.PoStage,
		Period:           po.Period,
		PoNumber:         po.PoNumber,
		PoBudgetRef:      poBudgetRef,
		TotalBudgetPo:    totalQty,
		SupplierID:       po.SupplierID,
		SupplierName:     supplierName,
		TotalQuantity:    totalQty,
		Status:           po.Status,
		ExternalSystem:   po.ExternalSystem,
		ExternalPoNumber: po.ExternalPoNumber,
	}

	itemDetails := make([]models.POItemDetail, 0, len(items))
	for _, it := range items {
		itemDetails = append(itemDetails, models.POItemDetail{
			ID:           it.ID,
			LineNo:       it.LineNo,
			UniqCode:     it.ItemUniqCode,
			PartNumber:   it.PartNumber,
			PartName:     it.PartName,
			Model:        it.ProductModel,
			Qty:          it.OrderedQty,
			Uom:          it.Uom,
			PcsPerKanban: it.PcsPerKanban,
			WeightKg:     it.WeightKg,
			UnitPrice:    it.UnitPrice,
		})
	}

	historyLogs := make([]models.POLogEntry, 0, len(logs))
	for _, l := range logs {
		historyLogs = append(historyLogs, models.POLogEntry{
			Action:     l.Action,
			Notes:      l.Notes,
			Username:   l.Username,
			OccurredAt: l.OccurredAt,
		})
	}

	return &models.PODetailResponse{
		PO:          header,
		Items:       itemDetails,
		HistoryLogs: historyLogs,
	}, nil
}

// ---------------------------------------------------------------------------
// DN list + detail
// ---------------------------------------------------------------------------

func (s *svc) ListDNs(ctx context.Context, f models.DNListFilter) (*models.DNListResponse, error) {
	dns, total, err := s.repo.ListDNs(ctx, f)
	if err != nil {
		return nil, err
	}

	items := make([]models.DNListItem, 0, len(dns))
	for _, dn := range dns {
		items = append(items, models.DNListItem{
			DnID:      dn.ID,
			DnNumber:  dn.DnNumber,
			Period:    dn.Period,
			PoNumber:  dn.PoNumber,
			DnType:    dn.DnType,
			Status:    dn.Status,
			CreatedAt: dn.CreatedAt,
		})
	}

	totalPages := 0
	if f.Limit > 0 {
		totalPages = int((total + int64(f.Limit) - 1) / int64(f.Limit))
	}

	return &models.DNListResponse{
		Items: items,
		Pagination: models.PaginationMeta{
			Total:      total,
			Page:       f.Page,
			Limit:      f.Limit,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *svc) GetDNDetail(ctx context.Context, dnID int64) (*models.DNDetailResponse, error) {
	dn, err := s.repo.GetDNByID(ctx, dnID)
	if err != nil {
		return nil, err
	}

	dnItems, err := s.repo.GetDNItems(ctx, dnID)
	if err != nil {
		return nil, err
	}

	// Resolve supplier
	var supplierName *string
	if dn.SupplierID != nil && *dn.SupplierID > 0 {
		if sup, serr := s.repo.GetLegacySupplier(ctx, *dn.SupplierID); serr == nil {
			supplierName = &sup.SupplierName
		}
	}

	itemDetails := make([]models.DNItemDetail, 0, len(dnItems))
	for _, it := range dnItems {
		itemDetails = append(itemDetails, models.DNItemDetail{
			ID:            it.ID,
			ItemUniqCode:  it.ItemUniqCode,
			OrderQty:      it.OrderQty,
			QtyStated:     it.QtyStated,
			QtyReceived:   it.QtyReceived,
			QualityStatus: it.QualityStatus,
			DateIncoming:  it.DateIncoming,
			Uom:           it.Uom,
		})
	}

	return &models.DNDetailResponse{
		DnID:            dn.ID,
		DnNumber:        dn.DnNumber,
		Period:          dn.Period,
		PoNumber:        dn.PoNumber,
		DnType:          dn.DnType,
		SupplierID:      dn.SupplierID,
		SupplierName:    supplierName,
		TotalPoQty:      dn.TotalPoQty,
		TotalDnCreated:  dn.TotalDnCreated,
		TotalDnIncoming: dn.TotalDnIncoming,
		Status:          dn.Status,
		CreatedAt:       dn.CreatedAt,
		Items:           itemDetails,
	}, nil
}

// ---------------------------------------------------------------------------
// GetFormOptions
// ---------------------------------------------------------------------------

func (s *svc) GetFormOptions(ctx context.Context, poType, period string) (*models.FormOptionsResponse, error) {
	// poType IS the budget_type (raw_material|indirect|subcon) — no conversion needed
	po1Pct, po2Pct, err := s.repo.GetSplitSetting(ctx, poType)
	if err != nil {
		return nil, err
	}

	// Suppliers with approved budget entries for this type+period
	legacySuppliers, err := s.repo.ListLegacySuppliersForBudget(ctx, poType, period)
	if err != nil {
		return nil, err
	}

	supplierOpts := make([]models.SupplierOption, 0, len(legacySuppliers))
	for _, sup := range legacySuppliers {
		supplierOpts = append(supplierOpts, models.SupplierOption{
			SupplierID:   sup.SupplierID,
			SupplierName: sup.SupplierName,
		})
	}

	// Budget options: aggregate by unique budget entry (group by period+type)
	// We fetch all approved entries and group them to build the dropdown.
	entries, err := s.repo.ListBudgetEntriesForGenerate(ctx, poType, period, nil, nil)
	if err != nil {
		return nil, err
	}

	// Group entries into budget options (one option per unique period+type combination)
	type budgetKey struct{ period string }
	budgetMap := map[budgetKey]*models.PoBudgetOption{}
	uniqSet := map[budgetKey]map[string]bool{}

	for _, e := range entries {
		k := budgetKey{period: e.Period}
		if _, ok := budgetMap[k]; !ok {
			// Use the first entry id as the reference anchor
			budgetMap[k] = &models.PoBudgetOption{
				PoBudgetID:  e.ID,
				PoBudgetRef: buildBudgetRefFromEntry(poType, e.Period, e.ID),
				Period:      e.Period,
			}
			uniqSet[k] = map[string]bool{}
		}
		budgetMap[k].TotalQuantity += e.Quantity
		budgetMap[k].TotalBudgetPo += e.Po1Qty + e.Po2Qty
		uniqSet[k][e.UniqCode] = true
	}
	for k, opt := range budgetMap {
		opt.TotalUniq = len(uniqSet[k])
	}

	budgetOpts := make([]models.PoBudgetOption, 0, len(budgetMap))
	for _, opt := range budgetMap {
		budgetOpts = append(budgetOpts, *opt)
	}

	return &models.FormOptionsResponse{
		PoStageOptions: []models.PoStageOption{
			{Stage: 1, Label: "PO 1"},
			{Stage: 2, Label: "PO 2"},
		},
		SplitSetting: models.SplitSettingOption{
			Po1Pct: po1Pct,
			Po2Pct: po2Pct,
		},
		SupplierOptions: supplierOpts,
		PoBudgetOptions: budgetOpts,
	}, nil
}

// ---------------------------------------------------------------------------
// GeneratePO
// ---------------------------------------------------------------------------

// GeneratePO creates PurchaseOrder records (+ items + logs) from budget entries.
//
// Multi-supplier logic:
//   - "bulk_all_suppliers": groups budget entries by supplier → one PO per supplier per stage.
//   - otherwise: all entries share the single SupplierID from the request.
//
// Type-safety rule:
//   - All resolved budget entries must have budget_type matching the request po_type.
//   - If ANY entry mismatches → return 422 (enforced here, not at DB level).
func (s *svc) GeneratePO(ctx context.Context, req models.GeneratePORequest, createdBy string) (*models.GeneratePOResponse, error) {
	// Fetch budget entries — supplier resolved from entries, not from request.
	entries, err := s.repo.ListBudgetEntriesForGenerate(ctx, req.PoType, req.Period, nil, req.PoBudgetEntryIDs)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no approved budget entries found for type=%s period=%s", req.PoType, req.Period)
	}

	// Type-safety: all entries must share the same budget_type.
	reqBudgetType := normalizeBudgetType(req.PoType)
	for _, e := range entries {
		if normalizeBudgetType(e.BudgetType) != reqBudgetType {
			return nil, fmt.Errorf(
				"type mismatch: po_type=%s but entry id=%d has budget_type=%s; cannot mix types in one PO",
				req.PoType, e.ID, e.BudgetType,
			)
		}
	}

	// ── Group entries by supplier ID (legacy BIGINT from budget entry) ──────
	type supplierGroup struct {
		legacyID     int64
		supplierName string
		entries      []models.POBudgetEntry
	}

	groupByID := map[int64]*supplierGroup{}
	for _, e := range entries {
		var id int64
		if e.SupplierID != nil {
			id = *e.SupplierID
		}
		if _, ok := groupByID[id]; !ok {
			name := ""
			if e.SupplierName != nil {
				name = *e.SupplierName
			}
			groupByID[id] = &supplierGroup{legacyID: id, supplierName: name}
		}
		groupByID[id].entries = append(groupByID[id].entries, e)
	}

	// ── Determine which stages to generate ───────────────────────────────
	stagesToGenerate := []int{1, 2}
	if req.GenerateMode == "stage_only" {
		if req.Stage != 1 && req.Stage != 2 {
			return nil, fmt.Errorf("stage_only mode requires stage=1 or stage=2, got %d", req.Stage)
		}
		stagesToGenerate = []int{req.Stage}
	}

	// ── Generate PO for each supplier × stage ────────────────────────────
	var result models.GeneratePOResponse

	for _, grp := range groupByID {
		// Compute budget-level stats (same for PO1 and PO2).
		stats := computeGroupStats(grp.entries)

		for _, stage := range stagesToGenerate {
			po, poItems, err := s.buildStagePO(ctx, grp.legacyID, grp.entries, req, createdBy, stage)
			if err != nil {
				return nil, err
			}
			if len(poItems) == 0 {
				continue
			}

			if err := s.repo.CreatePO(ctx, po); err != nil {
				return nil, err
			}
			for i := range poItems {
				poItems[i].PoID = po.PoID
			}
			if err := s.repo.CreatePOItems(ctx, poItems); err != nil {
				return nil, err
			}
			_ = s.repo.CreatePOLog(ctx, &models.PurchaseOrderLog{
				PoID:     po.PoID,
				Action:   "Created",
				Notes:    strPtr(fmt.Sprintf("Generated from budget — stage %d", stage)),
				Username: &createdBy,
			})

			budgetRef := buildBudgetRef(req.PoType, req.Period, po.PoBudgetEntryID)
			var totalQty float64
			itemDetails := make([]models.POItemDetail, 0, len(poItems))
			for _, it := range poItems {
				totalQty += it.OrderedQty
				itemDetails = append(itemDetails, models.POItemDetail{
					ID:           it.ID,
					LineNo:       it.LineNo,
					UniqCode:     it.ItemUniqCode,
					PartNumber:   it.PartNumber,
					PartName:     it.PartName,
					Model:        it.ProductModel,
					Qty:          it.OrderedQty,
					Uom:          it.Uom,
					PcsPerKanban: it.PcsPerKanban,
					WeightKg:     it.WeightKg,
					Budget:       it.SalesPlan,
					UnitPrice:    it.UnitPrice,
				})
			}

			supName := grp.supplierName
			result.Pos = append(result.Pos, models.GeneratedPOGroup{
				Stage: stage,
				PO: models.POHeaderDetail{
					PoID:             po.PoID,
					PoType:           po.PoType,
					PoStage:          po.PoStage,
					Period:           po.Period,
					PoNumber:         po.PoNumber,
					PoBudgetRef:      budgetRef,
					SalesPlan:        stats.salesPlan,
					TotalBudgetPo:    stats.totalBudgetPo,
					SupplierID:       &grp.legacyID,
					SupplierName:     &supName,
					TotalQuantity:    totalQty,
					TotalUniq:        stats.totalUniq,
					TotalWeight:      stats.totalWeight,
					Status:           po.Status,
					ExternalSystem:   po.ExternalSystem,
					ExternalPoNumber: po.ExternalPoNumber,
				},
				Items: itemDetails,
			})
		}
	}

	// Flag all used budget entries so they cannot be regenerated into another PO.
	usedEntryIDs := make([]int64, 0, len(entries))
	for _, e := range entries {
		usedEntryIDs = append(usedEntryIDs, e.ID)
	}
	if err := s.repo.MarkBudgetEntriesAsUsed(ctx, usedEntryIDs); err != nil {
		return nil, err
	}

	return &result, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// groupStats holds budget-level aggregates that are the same for PO1 and PO2.
type groupStats struct {
	salesPlan     float64
	totalBudgetPo float64 // sum of purchase_request (full, before split)
	totalWeight   float64 // sum(weight_kg * purchase_request)
	totalUniq     int
}

// computeGroupStats derives budget-level stats from a set of budget entries.
// These values are identical for PO1 and PO2 — they reflect the full budget, not the split.
func computeGroupStats(entries []models.POBudgetEntry) groupStats {
	uniqSet := map[string]bool{}
	var s groupStats
	for _, e := range entries {
		uniqSet[e.UniqCode] = true
		s.salesPlan += e.SalesPlan
		s.totalBudgetPo += e.PurchaseRequest
		if e.WeightKg != nil {
			s.totalWeight += *e.WeightKg * e.PurchaseRequest
		}
	}
	s.totalUniq = len(uniqSet)
	return s
}

// buildStagePO builds one PurchaseOrder for the given stage (1 or 2).
// Each budget entry is expanded into N kanban lines where N = ceil(rawQty / pcs_per_kanban).
// Every line has qty = pcs_per_kanban (one physical kanban unit per line).
// SalesPlan is carried on each item (transient) for response building.
func (s *svc) buildStagePO(ctx context.Context, legacySupID int64, entries []models.POBudgetEntry, req models.GeneratePORequest, createdBy string, stage int) (*models.PurchaseOrder, []models.PurchaseOrderItem, error) {
	poNumber, err := s.repo.NextPONumber(ctx, req.PoType, req.Period)
	if err != nil {
		return nil, nil, err
	}

	var firstEntryID *int64
	if len(entries) > 0 {
		id := entries[0].ID
		firstEntryID = &id
	}

	now := time.Now()
	createdByStr := createdBy

	items := make([]models.PurchaseOrderItem, 0, len(entries))
	lineNo := 1
	var totalWeight float64

	for _, e := range entries {
		// Pick raw qty for this stage (pre-calculated as purchase_request * pct / 100).
		var rawQty float64
		if stage == 1 {
			rawQty = e.Po1Qty
		} else {
			rawQty = e.Po2Qty
		}
		if rawQty <= 0 {
			continue
		}

		// Expand into N kanban lines. Each line = one full kanban unit (pcs_per_kanban qty).
		// If pcs_per_kanban is not set, treat the whole rawQty as one line.
		numKanbans := 1
		kanbanQty := rawQty
		if e.PcsPerKanban != nil && *e.PcsPerKanban > 0 {
			k := float64(*e.PcsPerKanban)
			numKanbans = int(math.Ceil(rawQty / k))
			kanbanQty = k
		}

		if e.WeightKg != nil {
			totalWeight += *e.WeightKg * kanbanQty * float64(numKanbans)
		}

		for i := 0; i < numKanbans; i++ {
			entryID := e.ID
			items = append(items, models.PurchaseOrderItem{
				LineNo:          lineNo,
				ItemUniqCode:    e.UniqCode,
				ProductModel:    e.ProductModel,
				MaterialType:    e.MaterialType,
				PartName:        e.PartName,
				PartNumber:      e.PartNumber,
				Uom:             e.Uom,
				WeightKg:        e.WeightKg,
				OrderedQty:      kanbanQty,
				PcsPerKanban:    e.PcsPerKanban,
				PoBudgetEntryID: &entryID,
				SalesPlan:       e.SalesPlan,
				Status:          "open",
			})
			lineNo++
		}
	}

	po := &models.PurchaseOrder{
		PoType:           req.PoType,
		PoStage:          &stage,
		Period:           req.Period,
		PoNumber:         poNumber,
		PoBudgetEntryID:  firstEntryID,
		SupplierID:       &legacySupID,
		Status:           "pending",
		PoDate:           &now,
		TotalWeight:      &totalWeight,
		ExternalSystem:   strPtrIfNonEmpty(req.ExternalSystem),
		ExternalPoNumber: strPtrIfNonEmpty(req.ExternalPoNumber),
		CreatedBy:        &createdByStr,
		UpdatedBy:        &createdByStr,
	}

	return po, items, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildBudgetRef generates display string "POB-{YYYY}-{TYPE}-{id}".
func buildBudgetRef(poType, period string, entryID *int64) string {
	year := extractYear(period)
	t := strings.ToUpper(poType)
	if entryID == nil {
		return fmt.Sprintf("POB-%s-%s-?", year, t)
	}
	return fmt.Sprintf("POB-%s-%s-%d", year, t, *entryID)
}

func buildBudgetRefFromEntry(poType, period string, entryID int64) string {
	year := extractYear(period)
	t := strings.ToUpper(poType)
	return fmt.Sprintf("POB-%s-%s-%d", year, t, entryID)
}

func extractYear(period string) string {
	// period can be "2024-01" or "October 2025"
	parts := strings.Fields(period)
	for _, p := range parts {
		if len(p) == 4 {
			return p
		}
	}
	if len(period) >= 4 {
		return period[:4]
	}
	return time.Now().Format("2006")
}

func strPtr(s string) *string { return &s }

func strPtrIfNonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func normalizeBudgetType(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "_")
	for strings.Contains(v, "__") {
		v = strings.ReplaceAll(v, "__", "_")
	}
	return v
}
