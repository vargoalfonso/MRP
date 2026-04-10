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
	GetDNDetail(ctx context.Context, dnID string) (*models.DNDetailResponse, error)

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
			DnCreated:     r.DnCreated,
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
		DnCreated:        po.DnCreated,
		DnIncoming:       po.DnIncoming,
		TotalIncoming:    po.TotalIncoming,
		Status:           po.Status,
		ExternalSystem:   po.ExternalSystem,
		ExternalPoNumber: po.ExternalPoNumber,
	}

	itemDetails := make([]models.POItemDetail, 0, len(items))
	for _, it := range items {
		itemDetails = append(itemDetails, models.POItemDetail{
			ID:            it.ID,
			LineNo:        it.LineNo,
			UniqCode:      it.ItemUniqCode,
			PartNumber:    it.PartNumber,
			PartName:      it.PartName,
			Model:         it.ProductModel,
			Qty:           it.OrderedQty,
			Uom:           it.Uom,
			PackingNumber: it.PackingNumber,
			PcsPerKanban:  it.PcsPerKanban,
			UnitPrice:     it.UnitPrice,
			Amount:        it.Amount,
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

func (s *svc) GetDNDetail(ctx context.Context, dnID string) (*models.DNDetailResponse, error) {
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
			PackingNumber: it.PackingNumber,
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
		budgetMap[k].TotalQuantity += e.PurchaseRequest
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
	// req.PoType IS the budget_type (raw_material|indirect|subcon) — no conversion needed

	// Resolve split %
	po1Pct, po2Pct, err := s.repo.GetSplitSetting(ctx, req.PoType)
	if err != nil {
		return nil, err
	}

	// Determine supplier UUID filter (for budget entry lookup via supplier_legacy_map).
	// If generate_mode = "bulk_all_suppliers" we don't filter by supplier.
	var supplierUUIDs []string
	if req.GenerateMode != "bulk_all_suppliers" && req.SupplierID > 0 {
		// Resolve legacy → UUID via supplier_legacy_map.
		// If not found in map, fall back to treating supplier_id as the UUID string
		// (best-effort; legacy table has no UUID).
		supplierUUIDs = []string{} // empty = no filter; handled by entryIDs below
	}

	entries, err := s.repo.ListBudgetEntriesForGenerate(ctx, req.PoType, req.Period, supplierUUIDs, req.PoBudgetEntryIDs)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no approved budget entries found for type=%s period=%s", req.PoType, req.Period)
	}

	// ── Type-safety: reject mixed budget_type in one generate call ──────────
	for _, e := range entries {
		if e.BudgetType != req.PoType {
			return nil, fmt.Errorf(
				"type mismatch: po_type=%s but entry id=%d has budget_type=%s; cannot mix types in one PO",
				req.PoType, e.ID, e.BudgetType,
			)
		}
	}

	// ── Determine stages to generate ────────────────────────────────────────
	var stages []int
	switch req.GenerateMode {
	case "stage_only":
		if req.Stage != 1 && req.Stage != 2 {
			return nil, fmt.Errorf("generate_mode=stage_only requires stage=1 or stage=2")
		}
		stages = []int{req.Stage}
	default: // "both_stages" and "bulk_all_suppliers"
		stages = []int{1, 2}
	}

	// ── Group entries by supplier (for bulk) ────────────────────────────────
	// For single-supplier mode, all entries go into one group keyed by req.SupplierID.
	type supplierGroup struct {
		legacyID int64
		entries  []models.POBudgetEntry
	}

	groupMap := map[int64]*supplierGroup{}

	if req.GenerateMode == "bulk_all_suppliers" {
		// Group by supplier_name / supplier_id (best effort via entry.SupplierName)
		// We group by the entry.SupplierID (UUID string) and resolve legacy ID separately.
		// Simplified: group by SupplierName if SupplierID UUID not in legacy map.
		for _, e := range entries {
			// Use 0 as placeholder key per supplier name.
			// In practice you'd resolve UUID→legacy BIGINT via supplier_legacy_map.
			// Here we use a hash of supplier name as the key.
			key := hashSupplierKey(e.SupplierID, e.SupplierName)
			if _, ok := groupMap[key]; !ok {
				groupMap[key] = &supplierGroup{legacyID: key}
			}
			groupMap[key].entries = append(groupMap[key].entries, e)
		}
	} else {
		groupMap[req.SupplierID] = &supplierGroup{
			legacyID: req.SupplierID,
			entries:  entries,
		}
	}

	// ── Resolve supplier names ────────────────────────────────────────────
	supplierNames := map[int64]string{}
	for legacyID := range groupMap {
		if legacyID > 0 {
			if sup, serr := s.repo.GetLegacySupplier(ctx, legacyID); serr == nil {
				supplierNames[legacyID] = sup.SupplierName
			}
		}
	}

	// ── Build line strategy helper ────────────────────────────────────────
	lineStrategy := req.LineStrategy
	if lineStrategy == "" {
		lineStrategy = "keep_granular"
	}

	// ── Generate PO for each supplier × stage ────────────────────────────
	var result models.GeneratePOResponse

	for _, grp := range groupMap {
		for _, stage := range stages {
			po, items, err := s.buildPO(ctx, buildPOParams{
				req:          req,
				stage:        stage,
				legacySupID:  grp.legacyID,
				supplierName: supplierNames[grp.legacyID],
				entries:      grp.entries,
				po1Pct:       po1Pct,
				po2Pct:       po2Pct,
				lineStrategy: lineStrategy,
				createdBy:    createdBy,
			})
			if err != nil {
				return nil, err
			}

			// Persist
			if err := s.repo.CreatePO(ctx, po); err != nil {
				return nil, err
			}
			for i := range items {
				items[i].PoID = po.PoID
			}
			if err := s.repo.CreatePOItems(ctx, items); err != nil {
				return nil, err
			}
			logEntry := &models.PurchaseOrderLog{
				PoID:     po.PoID,
				Action:   "Created",
				Notes:    strPtr(fmt.Sprintf("Generated from budget — stage %d", stage)),
				Username: &createdBy,
			}
			_ = s.repo.CreatePOLog(ctx, logEntry)

			// Build response group
			supName := supplierNames[grp.legacyID]
			budgetRef := buildBudgetRef(req.PoType, req.Period, po.PoBudgetEntryID)
			var totalQty float64
			itemDetails := make([]models.POItemDetail, 0, len(items))
			for ln, it := range items {
				totalQty += it.OrderedQty
				itemDetails = append(itemDetails, models.POItemDetail{
					ID:            it.ID,
					LineNo:        ln + 1,
					UniqCode:      it.ItemUniqCode,
					PartNumber:    it.PartNumber,
					PartName:      it.PartName,
					Model:         it.ProductModel,
					Qty:           it.OrderedQty,
					Uom:           it.Uom,
					PackingNumber: it.PackingNumber,
					PcsPerKanban:  it.PcsPerKanban,
					UnitPrice:     it.UnitPrice,
				})
			}

			result.Pos = append(result.Pos, models.GeneratedPOGroup{
				Stage: stage,
				PO: models.POHeaderDetail{
					PoID:             po.PoID,
					PoType:           po.PoType,
					PoStage:          po.PoStage,
					Period:           po.Period,
					PoNumber:         po.PoNumber,
					PoBudgetRef:      budgetRef,
					TotalBudgetPo:    totalQty,
					SupplierID:       &grp.legacyID,
					SupplierName:     &supName,
					TotalQuantity:    totalQty,
					DnCreated:        po.DnCreated,
					DnIncoming:       po.DnIncoming,
					TotalIncoming:    po.TotalIncoming,
					Status:           po.Status,
					ExternalSystem:   po.ExternalSystem,
					ExternalPoNumber: po.ExternalPoNumber,
				},
				Items: itemDetails,
			})
		}
	}

	return &result, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

type buildPOParams struct {
	req          models.GeneratePORequest
	stage        int
	legacySupID  int64
	supplierName string
	entries      []models.POBudgetEntry
	po1Pct       float64
	po2Pct       float64
	lineStrategy string
	createdBy    string
}

func (s *svc) buildPO(ctx context.Context, p buildPOParams) (*models.PurchaseOrder, []models.PurchaseOrderItem, error) {
	poNumber, err := s.repo.NextPONumber(ctx, p.req.PoType, p.req.Period)
	if err != nil {
		return nil, nil, err
	}

	pct := p.po1Pct
	if p.stage == 2 {
		pct = p.po2Pct
	}

	// Use per-entry pct if available (po_budget_entries stores individual pcts).
	// Merge lines
	type lineKey struct{ uniqCode string }
	type lineAgg struct {
		entry   models.POBudgetEntry
		qty     float64
		entryID int64
	}

	lineMap := map[lineKey]*lineAgg{}
	lineOrder := []lineKey{}

	for _, e := range p.entries {
		// Per-entry pct override if non-zero
		stagePct := pct
		if p.stage == 1 && e.Po1Pct > 0 {
			stagePct = e.Po1Pct
		} else if p.stage == 2 && e.Po2Pct > 0 {
			stagePct = e.Po2Pct
		}

		stageQty := e.PurchaseRequest * stagePct / 100

		key := lineKey{uniqCode: e.UniqCode}
		if p.lineStrategy == "aggregate_by_uniq" {
			if _, ok := lineMap[key]; !ok {
				lineMap[key] = &lineAgg{entry: e, qty: 0, entryID: e.ID}
				lineOrder = append(lineOrder, key)
			}
			lineMap[key].qty += stageQty
		} else {
			// keep_granular: use entry ID in key to avoid merging
			key = lineKey{uniqCode: fmt.Sprintf("%s::%d", e.UniqCode, e.ID)}
			lineMap[key] = &lineAgg{entry: e, qty: stageQty, entryID: e.ID}
			lineOrder = append(lineOrder, key)
		}
	}

	// Find the first entry ID for budget_ref
	var firstEntryID *int64
	if len(p.entries) > 0 {
		id := p.entries[0].ID
		firstEntryID = &id
	}

	now := time.Now()
	createdByStr := p.createdBy
	supplierID := p.legacySupID

	po := &models.PurchaseOrder{
		PoType:          p.req.PoType,
		Period:          p.req.Period,
		PoNumber:        poNumber,
		PoStage:         &p.stage,
		PoBudgetEntryID: firstEntryID,
		SupplierID:      &supplierID,
		TotalIncoming:   p.req.TotalIncoming, // rencana dari Step 1 wizard
		// DnCreated + DnIncoming: default 0 (DB), diupdate saat DN dibuat/diterima
		Status:           "draft",
		PoDate:           &now,
		ExternalSystem:   strPtrIfNonEmpty(p.req.ExternalSystem),
		ExternalPoNumber: strPtrIfNonEmpty(p.req.ExternalPoNumber),
		CreatedBy:        &createdByStr,
		UpdatedBy:        &createdByStr,
	}

	items := make([]models.PurchaseOrderItem, 0, len(lineOrder))
	for i, key := range lineOrder {
		agg := lineMap[key]
		entryID := agg.entryID
		packingNumber := agg.entry.PackingNumber
		pcsPerKanban := agg.entry.PcsPerKanban
		// If packing_number isn't explicitly provided, derive it from Kanban master data.
		// packing_number here represents how many kanban packs are needed for the ordered qty.
		if packingNumber == nil && pcsPerKanban != nil && *pcsPerKanban > 0 {
			packs := int(math.Ceil(agg.qty / float64(*pcsPerKanban)))
			s := fmt.Sprintf("%d", packs)
			packingNumber = &s
		}
		items = append(items, models.PurchaseOrderItem{
			LineNo:          i + 1,
			ItemUniqCode:    agg.entry.UniqCode,
			ProductModel:    agg.entry.ProductModel,
			MaterialType:    agg.entry.MaterialType,
			PartName:        agg.entry.PartName,
			PartNumber:      agg.entry.PartNumber,
			Uom:             agg.entry.Uom,
			WeightKg:        agg.entry.WeightKg,
			OrderedQty:      agg.qty,
			PackingNumber:   packingNumber,
			PcsPerKanban:    pcsPerKanban,
			PoBudgetEntryID: &entryID,
			Status:          "open",
		})
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

// hashSupplierKey produces a stable int64 key from supplier_id (UUID string) or supplier_name.
func hashSupplierKey(supplierIDStr *string, supplierName *string) int64 {
	if supplierIDStr != nil && *supplierIDStr != "" {
		var h int64
		for _, c := range *supplierIDStr {
			h = h*31 + int64(c)
		}
		if h < 0 {
			h = -h
		}
		return h
	}
	if supplierName != nil {
		var h int64
		for _, c := range *supplierName {
			h = h*31 + int64(c)
		}
		if h < 0 {
			h = -h
		}
		return h
	}
	return 0
}

func strPtr(s string) *string { return &s }

func strPtrIfNonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
