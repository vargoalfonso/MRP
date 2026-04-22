package repository

import (
	"context"

	"gorm.io/gorm"
)

type IRepository interface {
	// GetGlobalActivePeriode returns the period from global_parameters where
	// parameter_group = 'working_days' and status = 'active'.
	GetGlobalActivePeriode(ctx context.Context) (string, error)
	// GetWorkingDaysWithFallback returns (workingDays, periodeUsed).
	// First tries activePeriode; falls back to the latest available active period.
	GetWorkingDaysWithFallback(ctx context.Context, activePeriode string) (int, string, error)
	// RebuildDemandPeriodeSummaries upserts inventory_demand_periode_summaries for today
	// using the given active_periode and resolved working-days values.
	RebuildDemandPeriodeSummaries(ctx context.Context, activePeriode string, workingDays int, workingDaysPeriodeUsed string) (int64, error)
	// ListSupplierPerformanceAggregates reads delivery, QC, and PO data per active supplier
	// for the given period and returns one aggregate row per supplier.
	ListSupplierPerformanceAggregates(ctx context.Context, periodType, periodValue string) ([]SupplierPerformanceAggregateRow, error)
	// UpsertSupplierPerformanceSnapshots inserts or updates computed snapshot rows.
	UpsertSupplierPerformanceSnapshots(ctx context.Context, rows []SupplierPerformanceSnapshotRow) (int64, error)
}

type repo struct{ db *gorm.DB }

func New(db *gorm.DB) IRepository { return &repo{db: db} }

func (r *repo) GetGlobalActivePeriode(ctx context.Context) (string, error) {
	var result struct {
		Period string `gorm:"column:period"`
	}
	err := r.db.WithContext(ctx).
		Table("global_parameters").
		Select("period").
		Where("parameter_group = 'working_days' AND status = 'active'").
		Order("id DESC").
		Limit(1).
		Scan(&result).Error
	return result.Period, err
}

func (r *repo) GetWorkingDaysWithFallback(ctx context.Context, activePeriode string) (int, string, error) {
	var result struct {
		Period      string `gorm:"column:period"`
		WorkingDays int    `gorm:"column:working_days"`
	}

	// Try the active periode first.
	err := r.db.WithContext(ctx).
		Table("global_parameters").
		Select("period, working_days").
		Where("parameter_group = 'working_days' AND status = 'active' AND period = ? AND working_days > 0", activePeriode).
		Order("id DESC").
		Limit(1).
		Scan(&result).Error
	if err != nil {
		return 0, "", err
	}
	if result.WorkingDays > 0 {
		return result.WorkingDays, result.Period, nil
	}

	// Fallback: latest previous active period with working_days > 0.
	err = r.db.WithContext(ctx).
		Table("global_parameters").
		Select("period, working_days").
		Where("parameter_group = 'working_days' AND status = 'active' AND working_days > 0").
		Order("id DESC").
		Limit(1).
		Scan(&result).Error
	if err != nil {
		return 0, "", err
	}
	return result.WorkingDays, result.Period, nil
}

func (r *repo) RebuildDemandPeriodeSummaries(ctx context.Context, activePeriode string, workingDays int, workingDaysPeriodeUsed string) (int64, error) {
	result := r.db.WithContext(ctx).Exec(`
		INSERT INTO inventory_demand_periode_summaries (
			uniq_code,
			active_periode,
			snapshot_date,
			working_days_periode_used,
			working_days_used,
			safety_stock_calc_type_active,
			safety_stock_constanta_active,
			stockdays_calc_type_active,
			stockdays_constanta_active,
			prl_sum,
			po_customer_sum,
			total_demand_sum,
			rebuilt_at
		)
		SELECT
			rm.uniq_code,
			? AS active_periode,
			CURRENT_DATE AS snapshot_date,
			? AS working_days_periode_used,
			? AS working_days_used,
			ss.calculation_type  AS safety_stock_calc_type_active,
			ss.constanta         AS safety_stock_constanta_active,
			sd.calculation_type  AS stockdays_calc_type_active,
			sd.constanta         AS stockdays_constanta_active,
			COALESCE(prl.prl_sum, 0)        AS prl_sum,
			COALESCE(poc.po_customer_sum, 0) AS po_customer_sum,
			COALESCE(prl.prl_sum, 0) + COALESCE(poc.po_customer_sum, 0) AS total_demand_sum,
			NOW() AS rebuilt_at
		FROM raw_materials rm
		LEFT JOIN safety_stock_parameters ss
			ON ss.item_uniq_code = rm.uniq_code AND ss.inventory_type = 'raw_material'
		LEFT JOIN stockdays_parameters sd
			ON sd.item_uniq_code = rm.uniq_code AND sd.inventory_type = 'raw_material'
		LEFT JOIN (
			SELECT uniq_code, SUM(quantity) AS prl_sum
			FROM prls
			WHERE forecast_period = ?
			  AND status          = 'approved'
			  AND deleted_at      IS NULL
			GROUP BY uniq_code
		) prl ON prl.uniq_code = rm.uniq_code
		LEFT JOIN (
			SELECT i.item_uniq_code, SUM(i.quantity) AS po_customer_sum
			FROM customer_order_document_items i
			JOIN customer_order_documents d ON d.id = i.document_id
			WHERE d.period_schedule = ?
			  AND d.document_type   = 'PO'
			  AND d.status NOT IN ('cancelled', 'draft')
			  AND d.deleted_at      IS NULL
			GROUP BY i.item_uniq_code
		) poc ON poc.item_uniq_code = rm.uniq_code
		WHERE rm.deleted_at IS NULL
		ON CONFLICT (uniq_code, active_periode, snapshot_date)
		DO UPDATE SET
			working_days_periode_used     = EXCLUDED.working_days_periode_used,
			working_days_used             = EXCLUDED.working_days_used,
			safety_stock_calc_type_active = EXCLUDED.safety_stock_calc_type_active,
			safety_stock_constanta_active = EXCLUDED.safety_stock_constanta_active,
			stockdays_calc_type_active    = EXCLUDED.stockdays_calc_type_active,
			stockdays_constanta_active    = EXCLUDED.stockdays_constanta_active,
			prl_sum                       = EXCLUDED.prl_sum,
			po_customer_sum               = EXCLUDED.po_customer_sum,
			total_demand_sum              = EXCLUDED.total_demand_sum,
			rebuilt_at                    = EXCLUDED.rebuilt_at
	`, activePeriode, workingDaysPeriodeUsed, workingDays, activePeriode, activePeriode)

	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
