package repository

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SupplierPerformanceAggregateRow is the raw aggregated result from SQL per supplier.
type SupplierPerformanceAggregateRow struct {
	SupplierUUID           string  `gorm:"column:supplier_uuid"`
	SupplierCode           string  `gorm:"column:supplier_code"`
	SupplierName           string  `gorm:"column:supplier_name"`
	TotalPurchaseValue     float64 `gorm:"column:total_purchase_value"`
	OnTimeDeliveries       int     `gorm:"column:on_time_deliveries"`
	LateDeliveries         int     `gorm:"column:late_deliveries"`
	AverageDelayDays       float64 `gorm:"column:average_delay_days"`
	QualityInspectionCount int     `gorm:"column:quality_inspection_count"`
	AcceptedQuantity       float64 `gorm:"column:accepted_quantity"`
	RejectedQuantity       float64 `gorm:"column:rejected_quantity"`
}

// SupplierPerformanceSnapshotRow is the fully-computed row ready for upsert.
type SupplierPerformanceSnapshotRow struct {
	SnapshotUUID            string
	SupplierUUID            string
	SupplierCode            string
	SupplierName            string
	EvaluationPeriodType    string
	EvaluationPeriodValue   string
	EvaluationDate          time.Time
	TotalDeliveries         int
	OnTimeDeliveries        int
	LateDeliveries          int
	OTDPercentage           float64
	AverageDelayDays        float64
	QualityInspectionCount  int
	AcceptedQuantity        float64
	RejectedQuantity        float64
	InspectedQuantity       float64
	QualityPercentage       float64
	TotalPurchaseValue      float64
	ComputedScore           float64
	SystemGrade             string
	FinalGrade              string
	StatusLabel             string
	PoorDeliveryPerformance bool
	QCAlert                 bool
	SupplierReviewRequired  bool
	LogicVersion            string
	FormulaOTD              string
	FormulaQuality          string
	FormulaGrade            string
	FormulaNotes            string
}

func (r *repo) ListSupplierPerformanceAggregates(ctx context.Context, periodType, periodValue string) ([]SupplierPerformanceAggregateRow, error) {
	periodStart, periodEnd, err := periodDateRange(periodType, periodValue)
	if err != nil {
		return nil, err
	}

	var rows []SupplierPerformanceAggregateRow
	queryErr := r.db.WithContext(ctx).Raw(`
		SELECT
			s.uuid                                                                                  AS supplier_uuid,
			s.supplier_code,
			s.supplier_name,
			COALESCE(del.on_time_deliveries, 0)                                                     AS on_time_deliveries,
			COALESCE(del.late_deliveries, 0)                                                        AS late_deliveries,
			COALESCE(del.average_delay_days, 0)                                                     AS average_delay_days,
			COALESCE(qc.quality_inspection_count, 0)                                                AS quality_inspection_count,
			COALESCE(qc.accepted_quantity, 0)                                                       AS accepted_quantity,
			COALESCE(qc.rejected_quantity, 0)                                                       AS rejected_quantity,
			COALESCE(po.total_purchase_value, 0)                                                    AS total_purchase_value
		FROM suppliers s
		LEFT JOIN (
			SELECT
				dn.supplier_id,
				COUNT(DISTINCT dn.id) AS on_time_deliveries,
				0                     AS late_deliveries,
				0.0                   AS average_delay_days
			FROM delivery_notes dn
			JOIN delivery_note_items dni ON dni.dn_id = dn.id
			WHERE dn.supplier_id IS NOT NULL
			  AND dni.date_incoming IS NOT NULL
			  AND dni.date_incoming >= ?
			  AND dni.date_incoming <= ?
			  AND dn.status NOT IN ('draft', 'cancelled')
			GROUP BY dn.supplier_id
		) del ON del.supplier_id = s.id
		LEFT JOIN (
			SELECT
				dn.supplier_id,
				COUNT(DISTINCT qt.id)              AS quality_inspection_count,
				COALESCE(SUM(qt.good_quantity), 0) AS accepted_quantity,
				COALESCE(SUM(qt.ng_quantity), 0)   AS rejected_quantity
			FROM qc_tasks qt
			JOIN delivery_note_items dni ON dni.id = qt.incoming_dn_item_id
			JOIN delivery_notes dn ON dn.id = dni.dn_id
			WHERE qt.status = 'approved'
			  AND qt.date_checked IS NOT NULL
			  AND qt.date_checked >= ?
			  AND qt.date_checked <= ?
			  AND dn.supplier_id IS NOT NULL
			GROUP BY dn.supplier_id
		) qc ON qc.supplier_id = s.id
		LEFT JOIN (
			SELECT
				po.supplier_id,
				COALESCE(SUM(po.total_amount), 0) AS total_purchase_value
			FROM purchase_orders po
			WHERE po.supplier_id IS NOT NULL
			  AND po.period = ?
			  AND po.status NOT IN ('draft', 'cancelled')
			GROUP BY po.supplier_id
		) po ON po.supplier_id = s.id
		WHERE s.deleted_at IS NULL
		  AND s.status = 'Active'
	`, periodStart, periodEnd, periodStart, periodEnd, periodValue).Scan(&rows).Error

	return rows, queryErr
}

func (r *repo) UpsertSupplierPerformanceSnapshots(ctx context.Context, rows []SupplierPerformanceSnapshotRow) (int64, error) {
	var affected int64
	for _, row := range rows {
		result := r.db.WithContext(ctx).Exec(`
			INSERT INTO supplier_performance_snapshots (
				snapshot_uuid, supplier_uuid, supplier_code, supplier_name,
				evaluation_period_type, evaluation_period_value, evaluation_date,
				total_deliveries, on_time_deliveries, late_deliveries, otd_percentage,
				average_delay_days, quality_inspection_count, accepted_quantity,
				rejected_quantity, inspected_quantity, quality_percentage,
				total_purchase_value, computed_score, system_grade, final_grade,
				status_label, poor_delivery_performance, qc_alert,
				supplier_review_required, is_grade_overridden,
				logic_version, formula_otd, formula_quality, formula_grade,
				formula_notes, computed_at, data_through_at, created_at, updated_at
			) VALUES (
				?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,NOW(),NOW(),NOW(),NOW()
			)
			ON CONFLICT (supplier_uuid, evaluation_period_type, evaluation_period_value, logic_version)
			DO UPDATE SET
				total_deliveries          = EXCLUDED.total_deliveries,
				on_time_deliveries        = EXCLUDED.on_time_deliveries,
				late_deliveries           = EXCLUDED.late_deliveries,
				otd_percentage            = EXCLUDED.otd_percentage,
				average_delay_days        = EXCLUDED.average_delay_days,
				quality_inspection_count  = EXCLUDED.quality_inspection_count,
				accepted_quantity         = EXCLUDED.accepted_quantity,
				rejected_quantity         = EXCLUDED.rejected_quantity,
				inspected_quantity        = EXCLUDED.inspected_quantity,
				quality_percentage        = EXCLUDED.quality_percentage,
				total_purchase_value      = EXCLUDED.total_purchase_value,
				computed_score            = EXCLUDED.computed_score,
				system_grade              = EXCLUDED.system_grade,
				final_grade               = CASE
					WHEN supplier_performance_snapshots.is_grade_overridden
						THEN supplier_performance_snapshots.final_grade
					ELSE EXCLUDED.final_grade
				END,
				status_label              = CASE
					WHEN supplier_performance_snapshots.is_grade_overridden
						THEN supplier_performance_snapshots.status_label
					ELSE EXCLUDED.status_label
				END,
				poor_delivery_performance = EXCLUDED.poor_delivery_performance,
				qc_alert                  = EXCLUDED.qc_alert,
				supplier_review_required  = EXCLUDED.supplier_review_required,
				computed_at               = NOW(),
				data_through_at           = NOW(),
				updated_at                = NOW()
		`,
			row.SnapshotUUID, row.SupplierUUID, row.SupplierCode, row.SupplierName,
			row.EvaluationPeriodType, row.EvaluationPeriodValue, row.EvaluationDate,
			row.TotalDeliveries, row.OnTimeDeliveries, row.LateDeliveries, row.OTDPercentage,
			row.AverageDelayDays, row.QualityInspectionCount, row.AcceptedQuantity,
			row.RejectedQuantity, row.InspectedQuantity, row.QualityPercentage,
			row.TotalPurchaseValue, row.ComputedScore, row.SystemGrade, row.FinalGrade,
			row.StatusLabel, row.PoorDeliveryPerformance, row.QCAlert,
			row.SupplierReviewRequired, false,
			row.LogicVersion, row.FormulaOTD, row.FormulaQuality, row.FormulaGrade, row.FormulaNotes,
		)
		if result.Error != nil {
			return affected, fmt.Errorf("upsert snapshot for %s: %w", row.SupplierUUID, result.Error)
		}
		affected++
	}
	return affected, nil
}

// periodDateRange returns (start, end time.Time) for the given period.
// monthly: "2026-04" → 2026-04-01 .. 2026-04-30
// quarterly: "2026-Q1" → 2026-01-01 .. 2026-03-31
// yearly: "2026" → 2026-01-01 .. 2026-12-31
func periodDateRange(periodType, periodValue string) (time.Time, time.Time, error) {
	switch strings.ToLower(periodType) {
	case "monthly":
		t, err := time.Parse("2006-01", periodValue)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid period_value %q for monthly: %w", periodValue, err)
		}
		start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, -1)
		return start, end, nil
	case "quarterly":
		var year, quarter int
		if _, err := fmt.Sscanf(periodValue, "%d-Q%d", &year, &quarter); err != nil || quarter < 1 || quarter > 4 {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid period_value %q for quarterly, expected YYYY-QN", periodValue)
		}
		startMonth := time.Month((quarter-1)*3 + 1)
		start := time.Date(year, startMonth, 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 3, -1)
		return start, end, nil
	case "yearly":
		t, err := time.Parse("2006", periodValue)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid period_value %q for yearly: %w", periodValue, err)
		}
		start := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(t.Year(), 12, 31, 0, 0, 0, 0, time.UTC)
		return start, end, nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("unsupported period_type %q", periodType)
	}
}
