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
	// req.PoType IS the budget_type (raw_material|indirect|subcon) — no conversion needed

	// Generate PO must follow active configuration from po_split_settings.
	policy, err := s.repo.GetSplitPolicy(ctx, req.PoType)
	if err != nil {
		return nil, fmt.Errorf("generate requires active po_split_settings for po_type=%s: %w", req.PoType, err)
	}

	if policy.MinOrderQty > 0 && req.TotalIncoming < policy.MinOrderQty {
		return nil, fmt.Errorf(
			"total_incoming (%d) is below min_order_qty (%d) from po_split_settings",
			req.TotalIncoming,
			policy.MinOrderQty,
		)
	}

	// Resolve split %
	po1Pct, po2Pct := policy.Po1Pct, policy.Po2Pct
	if po1Pct+po2Pct <= 0 {
		return nil, fmt.Errorf("invalid split percentage in po_split_settings for po_type=%s", req.PoType)
	}

	// Determine supplier UUID filter (for budget entry lookup via supplier_legacy_map).
	// If generate_mode = "bulk_all_suppliers" we don't filter by supplier.
	var supplierUUIDs []string
	if req.GenerateMode != "bulk_all_suppliers" && req.SupplierID > 0 {
		uuid, mapErr := s.repo.GetSupplierUUIDByLegacyID(ctx, req.SupplierID)
		if mapErr != nil {
			return nil, mapErr
		}
		if uuid != nil {
			supplierUUIDs = []string{*uuid}
		}
	}

	entries, err := s.repo.ListBudgetEntriesForGenerate(ctx, req.PoType, req.Period, supplierUUIDs, req.PoBudgetEntryIDs)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no approved budget entries found for type=%s period=%s", req.PoType, req.Period)
	}

	// ── Type-safety: reject mixed budget_type in one generate call ──────────
	reqBudgetType := normalizeBudgetType(req.PoType)
	for _, e := range entries {
		if normalizeBudgetType(e.BudgetType) != reqBudgetType {
			return nil, fmt.Errorf(
				"type mismatch: po_type=%s but entry id=%d has budget_type=%s; cannot mix types in one PO",
				req.PoType, e.ID, e.BudgetType,
			)
		}
	}

	// Generate single PO per supplier with po1_qty + po2_qty items

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
	legacyIDs := make([]int64, 0, len(groupMap))
	for legacyID := range groupMap {
		if legacyID > 0 {
			legacyIDs = append(legacyIDs, legacyID)
		}
	}
	if len(legacyIDs) > 0 {
		suppliers, serr := s.repo.ListLegacySuppliersByIDs(ctx, legacyIDs)
		if serr != nil {
			return nil, serr
		}
		for _, sup := range suppliers {
			supplierNames[sup.SupplierID] = sup.SupplierName
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
		po, poItems, err := s.buildSimplePO(ctx, grp.legacyID, grp.entries, supplierNames[grp.legacyID], req, createdBy, policy.Po1Pct, policy.Po2Pct)
		if err != nil {
			return nil, err
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

		logEntry := &models.PurchaseOrderLog{
			PoID:     po.PoID,
			Action:   "Created",
			Notes:    strPtr(fmt.Sprintf("Generated from budget")),
			Username: &createdBy,
		}
		_ = s.repo.CreatePOLog(ctx, logEntry)

		supName := supplierNames[grp.legacyID]
		budgetRef := buildBudgetRef(req.PoType, req.Period, po.PoBudgetEntryID)
		var totalQty float64
		itemDetails := make([]models.POItemDetail, 0, len(poItems))
		for _, it := range poItems {
			totalQty += it.OrderedQty
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
			})
		}

		result.Pos = append(result.Pos, models.GeneratedPOGroup{
			Stage: 0,
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

	return &result, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (s *svc) buildSimplePO(ctx context.Context, legacySupID int64, entries []models.POBudgetEntry, supplierName string, req models.GeneratePORequest, createdBy string, po1Pct, po2Pct float64) (*models.PurchaseOrder, []models.PurchaseOrderItem, error) {
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

	po := &models.PurchaseOrder{
		PoType:           req.PoType,
		Period:           req.Period,
		PoNumber:         poNumber,
		PoBudgetEntryID:  firstEntryID,
		SupplierID:       &legacySupID,
		TotalIncoming:    req.TotalIncoming,
		Status:           "pending",
		PoDate:           &now,
		ExternalSystem:   strPtrIfNonEmpty(req.ExternalSystem),
		ExternalPoNumber: strPtrIfNonEmpty(req.ExternalPoNumber),
		CreatedBy:        &createdByStr,
		UpdatedBy:        &createdByStr,
	}

	items := make([]models.PurchaseOrderItem, 0, 2*len(entries))
	lineNo := 1

	for _, e := range entries {
		if e.Po1Qty > 0 {
			items = append(items, models.PurchaseOrderItem{
				LineNo:          lineNo,
				ItemUniqCode:    e.UniqCode,
				ProductModel:    e.ProductModel,
				MaterialType:    e.MaterialType,
				PartName:        e.PartName,
				PartNumber:      e.PartNumber,
				Uom:             e.Uom,
				WeightKg:        e.WeightKg,
				OrderedQty:      e.Po1Qty,
				PackingNumber:   firstNonEmptyStringPtr(e.KanbanNumber, e.PackingNumber),
				PcsPerKanban:    e.PcsPerKanban,
				PoBudgetEntryID: &e.ID,
				Status:          "open",
			})
			lineNo++
		}

		if e.Po2Qty > 0 {
			items = append(items, models.PurchaseOrderItem{
				LineNo:          lineNo,
				ItemUniqCode:    e.UniqCode,
				ProductModel:    e.ProductModel,
				MaterialType:    e.MaterialType,
				PartName:        e.PartName,
				PartNumber:      e.PartNumber,
				Uom:             e.Uom,
				WeightKg:        e.WeightKg,
				OrderedQty:      e.Po2Qty,
				PackingNumber:   firstNonEmptyStringPtr(e.KanbanNumber, e.PackingNumber),
				PcsPerKanban:    e.PcsPerKanban,
				PoBudgetEntryID: &e.ID,
				Status:          "open",
			})
			lineNo++
		}
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

func firstNonEmptyStringPtr(candidates ...*string) *string {
	for _, c := range candidates {
		if c != nil {
			v := strings.TrimSpace(*c)
			if v != "" {
				return &v
			}
		}
	}
	return nil
}


func splitPOItemsByMaxLines(items []models.PurchaseOrderItem, maxLines int) [][]models.PurchaseOrderItem {
	if len(items) == 0 {
		return [][]models.PurchaseOrderItem{}
	}
	if maxLines <= 0 || len(items) <= maxLines {
		return [][]models.PurchaseOrderItem{items}
	}

	chunks := make([][]models.PurchaseOrderItem, 0, int(math.Ceil(float64(len(items))/float64(maxLines))))
	for i := 0; i < len(items); i += maxLines {
		end := i + maxLines
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[i:end])
	}
	return chunks
}

func normalizeBudgetType(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "_")
	for strings.Contains(v, "__") {
		v = strings.ReplaceAll(v, "__", "_")
	}
	return v
}
