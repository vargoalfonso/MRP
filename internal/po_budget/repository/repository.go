// Package repository provides data-access for the PO Budget module.
package repository

import (
	"context"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/po_budget/models"
	"gorm.io/gorm"
)

// ListFilter carries all query params for listing entries.
type ListFilter struct {
	BudgetType     string
	UniqCode       string
	CustomerID     int64
	Period         string
	Status         string
	Search         string
	Page           int
	Limit          int
	OrderBy        string
	OrderDirection string
}

// AggFilter carries filter params for the aggregated view.
type AggFilter struct {
	BudgetType string
	UniqCode   string
	CustomerID int64
	Period     string
	Page       int
	Limit      int
}

// IRepository is the data-access contract for the PO Budget module.
type IRepository interface {
	// Entries CRUD
	CreateEntry(ctx context.Context, e *models.POBudgetEntry) error
	GetEntryByID(ctx context.Context, id int64) (*models.POBudgetEntry, error)
	UpdateEntry(ctx context.Context, e *models.POBudgetEntry) error
	DeleteEntry(ctx context.Context, id int64) error
	DeleteEntriesByIDs(ctx context.Context, ids []int64) error
	DeleteEntriesByFilter(ctx context.Context, budgetType, period string) error
	ListEntries(ctx context.Context, f ListFilter) ([]models.POBudgetEntry, int64, error)

	// Aggregated view
	ListAggregated(ctx context.Context, f AggFilter) ([]models.AggregatedRow, int64, error)
	// Summary cards
	GetSummary(ctx context.Context, budgetType, period string) (*models.SummaryResponse, error)

	// Split settings
	GetSplitSetting(ctx context.Context, budgetType string) (*models.POSplitSetting, error)
	ListSplitSettings(ctx context.Context) ([]models.POSplitSetting, error)
	UpdateSplitSetting(ctx context.Context, s *models.POSplitSetting) error

	// Bulk
	BulkCreateEntries(ctx context.Context, entries []models.POBudgetEntry) error

	// Schema checks (for better error messages)
	HasColumn(ctx context.Context, table, column string) (bool, error)

	// PRL
	// Data asli: table "prls". PRL document key is prl_id (varchar(32)).
	ListPRL(ctx context.Context, customerCode string, period string, page, limit int) ([]models.PRLRow, int64, error)
	GetPRLDocByPrlID(ctx context.Context, prlID string) (*models.PRLRow, error)
	GetPRLRowsByPrlID(ctx context.Context, prlID string) ([]models.PRLRow, error)
	GetPRLRowsByIDs(ctx context.Context, ids []int64) ([]models.PRLRow, error)

	// SumAllocatedQty returns how much of prl_row_id's quantity is already in po_budget_entries
	// for the given budget_type. Used to enforce the ceiling constraint.
	SumAllocatedQty(ctx context.Context, prlRowID int64, budgetType string) (float64, error)
	// SumAllocatedQtyBatch returns allocated totals for a slice of prl_row_ids in one query.
	SumAllocatedQtyBatch(ctx context.Context, prlRowIDs []int64, budgetType string) (map[int64]float64, error)
}

type repo struct{ db *gorm.DB }

func New(db *gorm.DB) IRepository { return &repo{db: db} }

func (r *repo) HasColumn(ctx context.Context, table, column string) (bool, error) {
	// Postgres: information_schema is portable.
	// Limit to current schema (public by default).
	var exists bool
	err := r.db.WithContext(ctx).Raw(
		`SELECT EXISTS(
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = ?
			  AND column_name = ?
		)`,
		table, column,
	).Scan(&exists).Error
	return exists, err
}

// ---------------------------------------------------------------------------
// Entries CRUD
// ---------------------------------------------------------------------------

func (r *repo) CreateEntry(ctx context.Context, e *models.POBudgetEntry) error {
	// Manual entry creation must not depend on PRL-linkage columns.
	// Bulk-from-PRL uses BulkCreateEntries (no omit) and will set linkage fields.
	return r.db.WithContext(ctx).
		Omit("PrlID", "PrlItemID", "PrlRef", "PrlRowID", "BudgetQty", "BudgetSubtype").
		Create(e).Error
}

func (r *repo) GetEntryByID(ctx context.Context, id int64) (*models.POBudgetEntry, error) {
	var e models.POBudgetEntry
	if err := r.db.WithContext(ctx).
		Select("*, purchase_request * po1_pct / 100 AS po1_qty, purchase_request * po2_pct / 100 AS po2_qty, purchase_request * (po1_pct + po2_pct) / 100 AS total_po").
		First(&e, id).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *repo) UpdateEntry(ctx context.Context, e *models.POBudgetEntry) error {
	return r.db.WithContext(ctx).
		Omit("PrlID", "PrlItemID", "PrlRef", "PrlRowID", "BudgetQty", "BudgetSubtype").
		Save(e).Error
}

func (r *repo) DeleteEntry(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.POBudgetEntry{}, id).Error
}

func (r *repo) DeleteEntriesByIDs(ctx context.Context, ids []int64) error {
	return r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Delete(&models.POBudgetEntry{}).Error
}

func (r *repo) DeleteEntriesByFilter(ctx context.Context, budgetType, period string) error {
	q := r.db.WithContext(ctx).Where("budget_type = ?", budgetType)
	if period != "" {
		q = q.Where("period = ?", period)
	}
	return q.Delete(&models.POBudgetEntry{}).Error
}

func (r *repo) ListEntries(ctx context.Context, f ListFilter) ([]models.POBudgetEntry, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.POBudgetEntry{})

	if f.BudgetType != "" {
		q = q.Where("budget_type = ?", f.BudgetType)
	}
	if f.UniqCode != "" {
		q = q.Where("uniq_code ILIKE ?", "%"+f.UniqCode+"%")
	}
	if f.CustomerID > 0 {
		q = q.Where("customer_id = ?", f.CustomerID)
	}
	if f.Period != "" {
		q = q.Where("period = ?", f.Period)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Search != "" {
		s := "%" + f.Search + "%"
		q = q.Where("uniq_code ILIKE ? OR customer_name ILIKE ? OR part_name ILIKE ?", s, s, s)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// ordering
	ob := allowedOrderCol(f.OrderBy, "created_at")
	dir := "DESC"
	if strings.ToLower(f.OrderDirection) == "asc" {
		dir = "ASC"
	}
	q = q.Order(ob + " " + dir)

	if f.Limit > 0 {
		offset := 0
		if f.Page > 1 {
			offset = (f.Page - 1) * f.Limit
		}
		q = q.Limit(f.Limit).Offset(offset)
	}

	var rows []models.POBudgetEntry
	if err := q.Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// ---------------------------------------------------------------------------
// Summary
// ---------------------------------------------------------------------------

func (r *repo) GetSummary(ctx context.Context, budgetType, period string) (*models.SummaryResponse, error) {
	// Use generated column total_po.
	type row struct {
		TotalEntries     int64
		TotalSalesPlan   float64
		TotalPR          float64
		TotalPO          float64
		TotalPRL         float64
		DeltaApoPrl      float64
		PendingApprovals int64
	}

	q := r.db.WithContext(ctx).Table("po_budget_entries").Where("budget_type = ?", budgetType)
	if period != "" {
		q = q.Where("period = ?", period)
	}

	var out row
	if err := q.Select(strings.Join([]string{
		"COUNT(*) AS total_entries",
		"COALESCE(SUM(sales_plan), 0) AS total_sales_plan",
		"COALESCE(SUM(purchase_request), 0) AS total_pr",
		"COALESCE(SUM(total_po), 0) AS total_po",
		"COALESCE(SUM(prl), 0) AS total_prl",
		"COALESCE(SUM(total_po), 0) - COALESCE(SUM(prl), 0) AS delta_apo_prl",
		"COALESCE(SUM(CASE WHEN status = 'Pending' THEN 1 ELSE 0 END), 0) AS pending_approvals",
	}, ", ")).Scan(&out).Error; err != nil {
		return nil, err
	}

	return &models.SummaryResponse{
		TotalEntries:     out.TotalEntries,
		TotalSalesPlan:   out.TotalSalesPlan,
		TotalPurchaseReq: out.TotalPR,
		TotalPO:          out.TotalPO,
		TotalPRL:         out.TotalPRL,
		DeltaApoPrl:      out.DeltaApoPrl,
		PendingApprovals: out.PendingApprovals,
	}, nil
}

// ---------------------------------------------------------------------------
// Aggregated view — sum by uniq_code + customer_name + period
// ---------------------------------------------------------------------------

func (r *repo) ListAggregated(ctx context.Context, f AggFilter) ([]models.AggregatedRow, int64, error) {
	q := r.db.WithContext(ctx).
		Table("po_budget_entries").
		Select(`
			uniq_code,
			COALESCE(MAX(customer_name), '') AS customer_name,
			period,
			COALESCE(MAX(product_model), '') AS product_model,
			SUM(sales_plan)                  AS total_sales_plan,
			SUM(purchase_request)            AS total_purchase_request,
			SUM(purchase_request * po1_pct / 100) AS total_po1,
			SUM(purchase_request * po2_pct / 100) AS total_po2,
			SUM(purchase_request * (po1_pct + po2_pct) / 100) AS total_po,
			SUM(prl)                         AS total_prl,
			SUM(purchase_request * (po1_pct + po2_pct) / 100) - SUM(prl) AS delta_apo_prl,
			COUNT(*)                         AS row_count
		`).
		Group("uniq_code, period")

	if f.BudgetType != "" {
		q = q.Where("budget_type = ?", f.BudgetType)
	}
	if f.UniqCode != "" {
		q = q.Where("uniq_code ILIKE ?", "%"+f.UniqCode+"%")
	}
	if f.CustomerID > 0 {
		q = q.Where("customer_id = ?", f.CustomerID)
	}
	if f.Period != "" {
		q = q.Where("period = ?", f.Period)
	}

	// Count distinct (uniq_code, period) groups
	type countResult struct{ N int64 }
	var cnt countResult
	if err := r.db.WithContext(ctx).
		Table("(?) AS sub", q).
		Select("COUNT(*) AS n").
		Scan(&cnt).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if f.Limit > 0 {
		offset := 0
		if f.Page > 1 {
			offset = (f.Page - 1) * f.Limit
		}
		q = q.Order("period DESC, uniq_code ASC").Limit(f.Limit).Offset(offset)
	} else {
		q = q.Order("period DESC, uniq_code ASC")
	}

	var rows []models.AggregatedRow
	if err := q.Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, cnt.N, nil
}

// ---------------------------------------------------------------------------
// Split settings
// ---------------------------------------------------------------------------

func (r *repo) GetSplitSetting(ctx context.Context, budgetType string) (*models.POSplitSetting, error) {
	var s models.POSplitSetting
	if err := r.db.WithContext(ctx).
		Where("budget_type = ?", budgetType).
		First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *repo) ListSplitSettings(ctx context.Context) ([]models.POSplitSetting, error) {
	var rows []models.POSplitSetting
	if err := r.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *repo) UpdateSplitSetting(ctx context.Context, s *models.POSplitSetting) error {
	s.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(s).Error
}

// ---------------------------------------------------------------------------
// Bulk create
// ---------------------------------------------------------------------------

func (r *repo) BulkCreateEntries(ctx context.Context, entries []models.POBudgetEntry) error {
	if len(entries) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(entries, 100).Error
}

// ---------------------------------------------------------------------------
// PRL reads (data asli: table "prls")
// ---------------------------------------------------------------------------

func (r *repo) ListPRL(ctx context.Context, customerCode string, period string, page, limit int) ([]models.PRLRow, int64, error) {
	// We return one representative row per prl_id (doc).
	// Assumption: all rows under the same prl_id share customer/period/status.
	q := r.db.WithContext(ctx).Table("prls").
		// NOTE: Postgres doesn't support MAX(uuid) in some setups,
		// so we avoid aggregating uuid-typed columns entirely.
		Select(strings.Join([]string{
			"MIN(id) AS id",
			"prl_id",
			"MAX(customer_code) AS customer_code",
			"MAX(customer_name) AS customer_name",
			"MAX(forecast_period) AS forecast_period",
			"MAX(status) AS status",
			"MAX(created_at) AS created_at",
			"MAX(updated_at) AS updated_at",
		}, ", ")).
		Where("deleted_at IS NULL").
		Group("prl_id")

	if customerCode != "" {
		q = q.Where("customer_code = ?", customerCode)
	}
	if period != "" {
		q = q.Where("forecast_period = ?", period)
	}

	var total int64
	// Count groups
	if err := r.db.WithContext(ctx).Table("prls").
		Select("prl_id").
		Where("deleted_at IS NULL").
		Scopes(func(db *gorm.DB) *gorm.DB {
			if customerCode != "" {
				db = db.Where("customer_code = ?", customerCode)
			}
			if period != "" {
				db = db.Where("forecast_period = ?", period)
			}
			return db
		}).
		Group("prl_id").
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		offset := 0
		if page > 1 {
			offset = (page - 1) * limit
		}
		q = q.Limit(limit).Offset(offset)
	}

	var rows []models.PRLRow
	if err := q.Order("MAX(created_at) DESC").Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *repo) GetPRLDocByPrlID(ctx context.Context, prlID string) (*models.PRLRow, error) {
	var row models.PRLRow
	if err := r.db.WithContext(ctx).
		Table("prls").
		Where("prl_id = ? AND deleted_at IS NULL", prlID).
		Order("id ASC").
		Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *repo) GetPRLRowsByPrlID(ctx context.Context, prlID string) ([]models.PRLRow, error) {
	var rows []models.PRLRow
	if err := r.db.WithContext(ctx).
		Table("prls").
		Where("prl_id = ? AND deleted_at IS NULL", prlID).
		Order("id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *repo) GetPRLRowsByIDs(ctx context.Context, ids []int64) ([]models.PRLRow, error) {
	if len(ids) == 0 {
		return []models.PRLRow{}, nil
	}
	var rows []models.PRLRow
	if err := r.db.WithContext(ctx).
		Table("prls").
		Where("id IN ? AND deleted_at IS NULL", ids).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// SumAllocatedQty returns how much quantity is already allocated (in po_budget_entries)
// for a given prl_item_id + budget_type combination.
// This is the enforcement gate for "quantity cannot exceed PRL budget".
func (r *repo) SumAllocatedQty(ctx context.Context, prlRowID int64, budgetType string) (float64, error) {
	type result struct{ Total float64 }
	var res result
	err := r.db.WithContext(ctx).
		Table("po_budget_entries").
		Select("COALESCE(SUM(quantity), 0) AS total").
		Where("prl_row_id = ? AND budget_type = ?", prlRowID, budgetType).
		Scan(&res).Error
	return res.Total, err
}

// SumAllocatedQtyBatch returns allocated totals for multiple prl_item_ids in one query.
// Returns map[prlItemID]allocatedQty.
func (r *repo) SumAllocatedQtyBatch(ctx context.Context, prlRowIDs []int64, budgetType string) (map[int64]float64, error) {
	if len(prlRowIDs) == 0 {
		return map[int64]float64{}, nil
	}
	type row struct {
		PrlRowID int64
		Total    float64
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Table("po_budget_entries").
		Select("prl_row_id, COALESCE(SUM(quantity), 0) AS total").
		Where("prl_row_id IN ? AND budget_type = ?", prlRowIDs, budgetType).
		Group("prl_row_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[int64]float64, len(rows))
	for _, r := range rows {
		out[r.PrlRowID] = r.Total
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var allowedCols = map[string]bool{
	"created_at": true, "updated_at": true, "period_date": true,
	"uniq_code": true, "customer_name": true, "total_po": true,
}

func allowedOrderCol(col, fallback string) string {
	if allowedCols[col] {
		return col
	}
	return fallback
}
