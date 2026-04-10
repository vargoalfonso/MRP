// Package repository handles all DB queries for the Procurement module.
package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/procurement/models"
	"gorm.io/gorm"
)

// IRepository is the data-access contract for the Procurement module.
type IRepository interface {
	// Summary KPI
	GetSummary(ctx context.Context, poType, period string) (models.POSummaryRow, error)

	// PO Board list
	ListPOBoard(ctx context.Context, f POBoardFilter) ([]models.POBoardRow, int64, error)

	// PO Detail
	GetPOByID(ctx context.Context, poID int64) (*models.PurchaseOrder, error)
	GetPOItems(ctx context.Context, poID int64) ([]models.PurchaseOrderItem, error)
	GetPOLogs(ctx context.Context, poID int64) ([]models.PurchaseOrderLog, error)

	// DN queries
	ListDNs(ctx context.Context, f models.DNListFilter) ([]models.IncomingDN, int64, error)
	GetDNByID(ctx context.Context, dnID string) (*models.IncomingDN, error)
	GetDNItems(ctx context.Context, dnID string) ([]models.IncomingDNItem, error)

	// Supplier lookup (legacy)
	GetLegacySupplier(ctx context.Context, supplierID int64) (*models.LegacySupplier, error)
	ListLegacySuppliersForBudget(ctx context.Context, budgetType, period string) ([]models.LegacySupplier, error)

	// Budget entries (read-only from po_budget_entries for form_options / generate)
	ListBudgetEntriesForGenerate(ctx context.Context, budgetType, period string, supplierUUIDs []string, entryIDs []int64) ([]models.POBudgetEntry, error)
	GetSplitSetting(ctx context.Context, budgetType string) (float64, float64, error) // po1Pct, po2Pct

	// PO generation writes
	CreatePO(ctx context.Context, po *models.PurchaseOrder) error
	CreatePOItems(ctx context.Context, items []models.PurchaseOrderItem) error
	CreatePOLog(ctx context.Context, log *models.PurchaseOrderLog) error

	// PO number sequence helper
	NextPONumber(ctx context.Context, poType, period string) (string, error)
}

// POBoardFilter holds all filter options for the PO board list query.
type POBoardFilter struct {
	PoType     string
	Period     string
	SupplierID int64
	UniqCode   string
	Status     string
	LateOnly   bool
	Search     string
	Page       int
	Limit      int
	OrderBy    string
	OrderDir   string
}

func poTypeCode(poType string) string {
	switch strings.ToLower(strings.TrimSpace(poType)) {
	case "raw_material":
		return "RM"
	case "subcon":
		return "SB"
	case "indirect":
		return "IB"
	default:
		return strings.ToUpper(strings.TrimSpace(poType))
	}
}

func extractYear(period string) string {
	// period can be "2025-10" or "October 2025".
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

// ---------------------------------------------------------------------------

type repo struct {
	db *gorm.DB
}

// New returns a new IRepository backed by the given *gorm.DB.
func New(db *gorm.DB) IRepository {
	return &repo{db: db}
}

// ---------------------------------------------------------------------------
// Summary
// ---------------------------------------------------------------------------

func (r *repo) GetSummary(ctx context.Context, poType, period string) (models.POSummaryRow, error) {
	var row models.POSummaryRow

	// Late definition: expected_delivery_date < today AND qty_delivered < total ordered (via items).
	// Simplified: we count POs where status = 'draft' or 'open' and expected_delivery_date < today.
	today := time.Now().Format("2006-01-02")

	query := r.db.WithContext(ctx).
		Table("purchase_orders po").
		Select(`
			COUNT(DISTINCT po.po_id)                          AS total_pos,
			COUNT(DISTINCT po.supplier_id)                    AS active_suppliers,
			COALESCE(SUM(po.total_amount), 0)                 AS total_po_value,
			COUNT(DISTINCT CASE
				WHEN po.expected_delivery_date IS NOT NULL
				 AND po.expected_delivery_date < ?
				 AND po.status NOT IN ('closed','cancelled')
				THEN po.po_id END)                            AS late_deliveries
		`, today)

	if poType != "" {
		query = query.Where("po.po_type = ?", poType)
	}
	if period != "" {
		query = query.Where("po.period = ?", period)
	}

	if err := query.Scan(&row).Error; err != nil {
		return row, fmt.Errorf("GetSummary: %w", err)
	}
	return row, nil
}

// ---------------------------------------------------------------------------
// PO Board list
// ---------------------------------------------------------------------------

func (r *repo) ListPOBoard(ctx context.Context, f POBoardFilter) ([]models.POBoardRow, int64, error) {
	base := r.db.WithContext(ctx).
		Table("purchase_orders po").
		Joins("LEFT JOIN supplier s ON s.supplier_id = po.supplier_id").
		Joins(`LEFT JOIN LATERAL (
			SELECT COALESCE(SUM(poi.ordered_qty), 0) AS total_budget_po
			FROM   purchase_order_items poi
			WHERE  poi.po_id = po.po_id
		) budget ON true`).
		Joins(`LEFT JOIN LATERAL (
			SELECT COALESCE(SUM(idi.qty_received::numeric), 0) AS qty_delivered,
			       MIN(idi.item_uniq_code)                      AS uniq_code
			FROM   incoming_dns idn
			JOIN   incoming_dn_items idi ON idi.incoming_dn_id = idn.id
			WHERE  idn.po_number = po.po_number
		) dn_agg ON true`)

	// Apply filters
	if f.PoType != "" {
		base = base.Where("po.po_type = ?", f.PoType)
	}
	if f.Period != "" {
		base = base.Where("po.period = ?", f.Period)
	}
	if f.SupplierID > 0 {
		base = base.Where("po.supplier_id = ?", f.SupplierID)
	}
	if f.Status != "" {
		base = base.Where("po.status = ?", f.Status)
	}
	if f.LateOnly {
		today := time.Now().Format("2006-01-02")
		base = base.Where("po.expected_delivery_date IS NOT NULL AND po.expected_delivery_date < ? AND po.status NOT IN ('closed','cancelled')", today)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		base = base.Where("po.po_number ILIKE ? OR s.supplier_name ILIKE ?", like, like)
	}

	// Count
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("ListPOBoard count: %w", err)
	}

	// Ordering
	orderBy := "po.created_at"
	if f.OrderBy != "" {
		orderBy = "po." + f.OrderBy
	}
	dir := "DESC"
	if f.OrderDir == "asc" {
		dir = "ASC"
	}

	// Pagination
	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := 0
	if f.Page > 1 {
		offset = (f.Page - 1) * limit
	}

	today := time.Now().Format("2006-01-02")
	var rows []models.POBoardRow
	err := base.
		Select(`
			po.po_id,
			po.po_type,
			po.po_stage,
			po.period,
			po.po_number,
			COALESCE(budget.total_budget_po, 0)              AS total_budget_po,
			COALESCE(dn_agg.qty_delivered, 0)                AS qty_delivered,
			dn_agg.uniq_code,
			po.supplier_id,
			s.supplier_name,
			po.dn_created,
			po.status,
			po.total_amount,
			(po.expected_delivery_date IS NOT NULL
			  AND po.expected_delivery_date < ?
			  AND po.status NOT IN ('closed','cancelled'))   AS is_late
		`, today).
		Order(fmt.Sprintf("%s %s", orderBy, dir)).
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error

	if err != nil {
		return nil, 0, fmt.Errorf("ListPOBoard scan: %w", err)
	}
	return rows, total, nil
}

// ---------------------------------------------------------------------------
// PO Detail
// ---------------------------------------------------------------------------

func (r *repo) GetPOByID(ctx context.Context, poID int64) (*models.PurchaseOrder, error) {
	var po models.PurchaseOrder
	if err := r.db.WithContext(ctx).First(&po, "po_id = ?", poID).Error; err != nil {
		return nil, fmt.Errorf("GetPOByID %d: %w", poID, err)
	}
	return &po, nil
}

func (r *repo) GetPOItems(ctx context.Context, poID int64) ([]models.PurchaseOrderItem, error) {
	var items []models.PurchaseOrderItem
	if err := r.db.WithContext(ctx).
		Where("po_id = ?", poID).
		Order("line_no ASC").
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("GetPOItems %d: %w", poID, err)
	}
	return items, nil
}

func (r *repo) GetPOLogs(ctx context.Context, poID int64) ([]models.PurchaseOrderLog, error) {
	var logs []models.PurchaseOrderLog
	if err := r.db.WithContext(ctx).
		Where("po_id = ?", poID).
		Order("occurred_at ASC").
		Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("GetPOLogs %d: %w", poID, err)
	}
	return logs, nil
}

// ---------------------------------------------------------------------------
// DN queries
// ---------------------------------------------------------------------------

func (r *repo) ListDNs(ctx context.Context, f models.DNListFilter) ([]models.IncomingDN, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.IncomingDN{})
	if f.PoNumber != "" {
		q = q.Where("po_number = ?", f.PoNumber)
	}
	if f.Period != "" {
		q = q.Where("period = ?", f.Period)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("ListDNs count: %w", err)
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := 0
	if f.Page > 1 {
		offset = (f.Page - 1) * limit
	}

	var dns []models.IncomingDN
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&dns).Error; err != nil {
		return nil, 0, fmt.Errorf("ListDNs scan: %w", err)
	}
	return dns, total, nil
}

func (r *repo) GetDNByID(ctx context.Context, dnID string) (*models.IncomingDN, error) {
	var dn models.IncomingDN
	if err := r.db.WithContext(ctx).First(&dn, "id = ?", dnID).Error; err != nil {
		return nil, fmt.Errorf("GetDNByID %s: %w", dnID, err)
	}
	return &dn, nil
}

func (r *repo) GetDNItems(ctx context.Context, dnID string) ([]models.IncomingDNItem, error) {
	var items []models.IncomingDNItem
	if err := r.db.WithContext(ctx).
		Where("incoming_dn_id = ?", dnID).
		Order("created_at ASC").
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("GetDNItems %s: %w", dnID, err)
	}
	return items, nil
}

// ---------------------------------------------------------------------------
// Supplier
// ---------------------------------------------------------------------------

func (r *repo) GetLegacySupplier(ctx context.Context, supplierID int64) (*models.LegacySupplier, error) {
	var s models.LegacySupplier
	if err := r.db.WithContext(ctx).First(&s, "supplier_id = ?", supplierID).Error; err != nil {
		return nil, fmt.Errorf("GetLegacySupplier %d: %w", supplierID, err)
	}
	return &s, nil
}

// ListLegacySuppliersForBudget returns legacy suppliers that have budget entries
// for the given budget_type + period (via supplier_legacy_map bridge).
func (r *repo) ListLegacySuppliersForBudget(ctx context.Context, budgetType, period string) ([]models.LegacySupplier, error) {
	var suppliers []models.LegacySupplier
	err := r.db.WithContext(ctx).
		Table("supplier s").
		Joins(`JOIN supplier_legacy_map slm ON slm.legacy_supplier_id = s.supplier_id`).
		Joins(`JOIN po_budget_entries pbe ON pbe.supplier_id::text = slm.supplier_uuid::text`).
		Where("pbe.budget_type = ? AND pbe.period ILIKE ?", budgetType, "%"+period+"%").
		Where("pbe.status = 'Approved'").
		Distinct("s.supplier_id, s.supplier_name").
		Select("s.supplier_id, s.supplier_name").
		Find(&suppliers).Error
	if err != nil {
		return nil, fmt.Errorf("ListLegacySuppliersForBudget: %w", err)
	}
	return suppliers, nil
}

// ---------------------------------------------------------------------------
// Budget entries (read-only)
// ---------------------------------------------------------------------------

// ListBudgetEntriesForGenerate fetches Approved po_budget_entries for PO generation.
// If entryIDs is non-empty, filters by those IDs (explicit selection).
// Otherwise filters by budgetType+period and optionally by supplier UUID.
func (r *repo) ListBudgetEntriesForGenerate(ctx context.Context, budgetType, period string, supplierUUIDs []string, entryIDs []int64) ([]models.POBudgetEntry, error) {
	q := r.db.WithContext(ctx).
		Table("po_budget_entries pbe").
		Select("pbe.*, kp.kanban_qty AS pcs_per_kanban").
		Joins("LEFT JOIN kanban_parameters kp ON kp.item_uniq_code = pbe.uniq_code AND kp.status ILIKE 'active'").
		Where("pbe.budget_type = ?", budgetType).
		Where("pbe.status = 'Approved'")

	if len(entryIDs) > 0 {
		q = q.Where("pbe.id IN ?", entryIDs)
	} else {
		// Period stored as "October 2025" — allow matching via ILIKE for "2024-01" format too.
		q = q.Where("pbe.period ILIKE ?", "%"+period+"%")
		if len(supplierUUIDs) > 0 {
			q = q.Where("pbe.supplier_id::text IN ?", supplierUUIDs)
		}
	}

	var entries []models.POBudgetEntry
	if err := q.Order("pbe.id ASC").Scan(&entries).Error; err != nil {
		return nil, fmt.Errorf("ListBudgetEntriesForGenerate: %w", err)
	}
	return entries, nil
}

// GetSplitSetting returns po1_pct, po2_pct for the given budget type.
// Falls back to 60/40 if no record found.
func (r *repo) GetSplitSetting(ctx context.Context, budgetType string) (float64, float64, error) {
	type row struct {
		Po1Pct float64 `gorm:"column:po1_pct"`
		Po2Pct float64 `gorm:"column:po2_pct"`
	}
	var s row
	err := r.db.WithContext(ctx).
		Table("po_split_settings").
		Select("po1_pct, po2_pct").
		Where("budget_type = ? AND status = 'Active'", budgetType).
		First(&s).Error
	if err != nil {
		// Fallback to global 60/40
		return 60, 40, nil
	}
	return s.Po1Pct, s.Po2Pct, nil
}

// ---------------------------------------------------------------------------
// PO writes
// ---------------------------------------------------------------------------

func (r *repo) CreatePO(ctx context.Context, po *models.PurchaseOrder) error {
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return fmt.Errorf("CreatePO: %w", err)
	}
	return nil
}

func (r *repo) CreatePOItems(ctx context.Context, items []models.PurchaseOrderItem) error {
	if len(items) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Create(&items).Error; err != nil {
		return fmt.Errorf("CreatePOItems: %w", err)
	}
	return nil
}

func (r *repo) CreatePOLog(ctx context.Context, log *models.PurchaseOrderLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("CreatePOLog: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// PO number sequence
// ---------------------------------------------------------------------------

// NextPONumber generates the next sequential PO number for a given type+period.
// Format: PO-{CODE}-{YYYY}-{XXXX}
// Uses a DB-side MAX to avoid race conditions in low-concurrency scenarios.
// For high-concurrency, consider a dedicated sequence table.
func (r *repo) NextPONumber(ctx context.Context, poType, period string) (string, error) {
	y := extractYear(period)
	code := poTypeCode(poType)
	prefix := fmt.Sprintf("PO-%s-%s-", code, y)

	type result struct {
		MaxNum int `gorm:"column:max_num"`
	}
	var res result
	err := r.db.WithContext(ctx).
		Table("purchase_orders").
		Select("COALESCE(MAX(CAST(NULLIF(substring(po_number from '([0-9]+)$'), '') AS int)), 0) AS max_num").
		Where("po_number LIKE ?", prefix+"%").
		Scan(&res).Error
	if err != nil {
		return "", fmt.Errorf("NextPONumber: %w", err)
	}

	next := res.MaxNum + 1
	return fmt.Sprintf("%s%04d", prefix, next), nil
}
