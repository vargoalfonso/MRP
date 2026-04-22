package repository

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/supplier_performance/models"
	"github.com/ganasa18/go-template/pkg/pagination"
	"gorm.io/gorm"
)

type IRepository interface {
	ListSnapshots(ctx context.Context, p pagination.SupplierPerformancePaginationInput) ([]models.Snapshot, int64, error)
	GetSummary(ctx context.Context, periodType, periodValue string) (*models.SummaryResponse, error)
	GetCharts(ctx context.Context, periodType, periodValue string) (*models.ChartsResponse, error)
	ApplyOverride(ctx context.Context, supplierUUID, periodType, periodValue, grade, remarks, actor string) error
	ListAuditLogs(ctx context.Context, supplierUUID, periodType, periodValue string) ([]models.AuditLog, error)
}

type repo struct{ db *gorm.DB }

func New(db *gorm.DB) IRepository { return &repo{db: db} }

func (r *repo) ListSnapshots(ctx context.Context, p pagination.SupplierPerformancePaginationInput) ([]models.Snapshot, int64, error) {
	q := r.db.WithContext(ctx).Table("supplier_performance_snapshots").
		Where("deleted_at IS NULL")

	if p.PeriodType != "" {
		q = q.Where("evaluation_period_type = ?", p.PeriodType)
	}
	if p.PeriodValue != "" {
		q = q.Where("evaluation_period_value = ?", p.PeriodValue)
	}
	if p.Search != "" {
		like := "%" + strings.ToLower(p.Search) + "%"
		q = q.Where("(LOWER(supplier_name) LIKE ? OR LOWER(supplier_code) LIKE ?)", like, like)
	}
	if p.Status != "" {
		// status filter maps to status_label values
		switch strings.ToLower(p.Status) {
		case "excellent":
			q = q.Where("status_label = ?", "Excellent")
		case "good":
			q = q.Where("status_label = ?", "Good")
		case "review_required":
			q = q.Where("status_label = ?", "Review Required")
		}
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("ListSnapshots count: %w", err)
	}

	sortBy := "computed_at"
	allowed := map[string]bool{
		"supplier_name": true, "otd_percentage": true,
		"quality_percentage": true, "computed_score": true, "computed_at": true,
	}
	if allowed[p.SortBy] {
		sortBy = p.SortBy
	}
	dir := "DESC"
	if strings.ToLower(p.SortDirection) == "asc" {
		dir = "ASC"
	}

	limit := p.Limit
	if limit <= 0 {
		limit = 20
	}

	var rows []models.Snapshot
	if err := q.Order(sortBy + " " + dir).
		Limit(limit).
		Offset(p.Offset()).
		Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("ListSnapshots scan: %w", err)
	}
	return rows, total, nil
}

func (r *repo) GetSummary(ctx context.Context, periodType, periodValue string) (*models.SummaryResponse, error) {
	type raw struct {
		StatusLabel        string  `gorm:"column:status_label"`
		Count              int     `gorm:"column:cnt"`
		TotalPurchaseValue float64 `gorm:"column:total_pv"`
		LogicVersion       string  `gorm:"column:logic_version"`
		FormulaGrade       string  `gorm:"column:formula_grade"`
		ComputedAt         time.Time `gorm:"column:computed_at"`
	}

	var rows []raw
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			status_label,
			COUNT(*) AS cnt,
			SUM(total_purchase_value) AS total_pv,
			MAX(logic_version) AS logic_version,
			MAX(formula_grade) AS formula_grade,
			MAX(computed_at) AS computed_at
		FROM supplier_performance_snapshots
		WHERE deleted_at IS NULL
		  AND evaluation_period_type = ?
		  AND evaluation_period_value = ?
		GROUP BY status_label
	`, periodType, periodValue).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("GetSummary: %w", err)
	}

	resp := &models.SummaryResponse{}
	for _, row := range rows {
		resp.TotalPurchaseValue += row.TotalPurchaseValue
		resp.TotalSuppliersEvaluated += row.Count
		if resp.LogicVersion == "" {
			resp.LogicVersion = row.LogicVersion
			resp.FormulaGrade = row.FormulaGrade
			resp.ComputedAt = row.ComputedAt
		}
		switch row.StatusLabel {
		case "Excellent":
			resp.ExcellentSuppliers = row.Count
		case "Good":
			resp.GoodSuppliers = row.Count
		case "Review Required":
			resp.ReviewRequiredSuppliers = row.Count
		}
	}
	return resp, nil
}

func (r *repo) GetCharts(ctx context.Context, periodType, periodValue string) (*models.ChartsResponse, error) {
	// trend: last 6 periods of the same period_type, ordered chronologically
	var trend []models.ChartTrendPoint
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			evaluation_period_value                      AS period,
			ROUND(AVG(otd_percentage)::numeric, 2)      AS avg_otd_percentage,
			ROUND(AVG(quality_percentage)::numeric, 2)  AS avg_quality_percentage
		FROM supplier_performance_snapshots
		WHERE deleted_at IS NULL
		  AND evaluation_period_type = ?
		GROUP BY evaluation_period_value
		ORDER BY evaluation_period_value DESC
		LIMIT 6
	`, periodType).Scan(&trend).Error
	if err != nil {
		return nil, fmt.Errorf("GetCharts trend: %w", err)
	}
	// reverse to chronological order
	for i, j := 0, len(trend)-1; i < j; i, j = i+1, j-1 {
		trend[i], trend[j] = trend[j], trend[i]
	}

	base := r.db.WithContext(ctx).Table("supplier_performance_snapshots").
		Where("deleted_at IS NULL AND evaluation_period_type = ?", periodType)
	if periodValue != "" {
		base = base.Where("evaluation_period_value = ?", periodValue)
	}

	var scatter []models.ChartScatterPoint
	if err := base.Select("supplier_uuid, supplier_name, otd_percentage, quality_percentage, status_label").
		Scan(&scatter).Error; err != nil {
		return nil, fmt.Errorf("GetCharts scatter: %w", err)
	}

	var top5 []models.ChartRankPoint
	if err := base.Select("supplier_uuid, supplier_name, final_grade, status_label, computed_score").
		Order("computed_score DESC").Limit(5).Scan(&top5).Error; err != nil {
		return nil, fmt.Errorf("GetCharts top5: %w", err)
	}

	var bottom5 []models.ChartRankPoint
	if err := base.Select("supplier_uuid, supplier_name, final_grade, status_label, computed_score").
		Order("computed_score ASC").Limit(5).Scan(&bottom5).Error; err != nil {
		return nil, fmt.Errorf("GetCharts bottom5: %w", err)
	}

	if trend == nil {
		trend = []models.ChartTrendPoint{}
	}
	if scatter == nil {
		scatter = []models.ChartScatterPoint{}
	}
	if top5 == nil {
		top5 = []models.ChartRankPoint{}
	}
	if bottom5 == nil {
		bottom5 = []models.ChartRankPoint{}
	}

	return &models.ChartsResponse{
		Trend:   trend,
		Scatter: scatter,
		Top5:    top5,
		Bottom5: bottom5,
	}, nil
}

func (r *repo) ApplyOverride(ctx context.Context, supplierUUID, periodType, periodValue, grade, remarks, actor string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var snap models.Snapshot
		if err := tx.Table("supplier_performance_snapshots").
			Where("supplier_uuid = ? AND evaluation_period_type = ? AND evaluation_period_value = ? AND deleted_at IS NULL",
				supplierUUID, periodType, periodValue).
			First(&snap).Error; err != nil {
			return fmt.Errorf("snapshot not found: %w", err)
		}

		statusLabel := statusLabelFromGrade(grade)
		if err := tx.Exec(`
			UPDATE supplier_performance_snapshots
			SET is_grade_overridden = TRUE,
			    override_grade      = ?,
			    override_remarks    = ?,
			    override_by        = ?,
			    override_at        = NOW(),
			    final_grade        = ?,
			    status_label       = ?,
			    updated_at         = NOW()
			WHERE supplier_uuid = ?
			  AND evaluation_period_type = ?
			  AND evaluation_period_value = ?
			  AND deleted_at IS NULL
		`, grade, remarks, actor, grade, statusLabel, supplierUUID, periodType, periodValue).Error; err != nil {
			return fmt.Errorf("update snapshot: %w", err)
		}

		return tx.Exec(`
			INSERT INTO supplier_performance_audit_logs (
				snapshot_uuid, supplier_uuid,
				evaluation_period_type, evaluation_period_value,
				action, old_grade, new_grade, remarks, actor, occurred_at
			) VALUES (?,?,?,?,?,?,?,?,?,NOW())
		`, snap.SnapshotUUID, supplierUUID, periodType, periodValue,
			"override_grade", snap.PerformanceGrade, grade, remarks, actor).Error
	})
}

func (r *repo) ListAuditLogs(ctx context.Context, supplierUUID, periodType, periodValue string) ([]models.AuditLog, error) {
	var rows []models.AuditLog
	q := r.db.WithContext(ctx).Table("supplier_performance_audit_logs").
		Where("supplier_uuid = ?", supplierUUID).
		Order("occurred_at DESC")

	if periodType != "" {
		q = q.Where("evaluation_period_type = ?", periodType)
	}
	if periodValue != "" {
		q = q.Where("evaluation_period_value = ?", periodValue)
	}

	if err := q.Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListAuditLogs: %w", err)
	}
	return rows, nil
}

func statusLabelFromGrade(grade string) string {
	switch strings.ToUpper(strings.TrimSpace(grade)) {
	case "A":
		return "Excellent"
	case "B":
		return "Good"
	default:
		return "Review Required"
	}
}

func newPaginationMeta(total int64, p pagination.SupplierPerformancePaginationInput) models.PaginationMeta {
	pages := 0
	if p.Limit > 0 {
		pages = int(math.Ceil(float64(total) / float64(p.Limit)))
	}
	return models.PaginationMeta{
		Total:      total,
		Page:       p.Page,
		Limit:      p.Limit,
		TotalPages: pages,
	}
}

// NewPaginationMeta is exported for service use.
func NewPaginationMeta(total int64, p pagination.SupplierPerformancePaginationInput) models.PaginationMeta {
	return newPaginationMeta(total, p)
}
