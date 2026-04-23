package repository

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/supplier_performance/models"
	"github.com/ganasa18/go-template/pkg/pagination"
	"gorm.io/gorm"
)

type IRepository interface {
	ResolveLatestPeriodValue(ctx context.Context, periodType string) (string, error)
	ListSnapshots(ctx context.Context, p pagination.SupplierPerformancePaginationInput) ([]models.Snapshot, int64, error)
	GetSummary(ctx context.Context, periodType, periodValue string) (*models.SummaryResponse, error)
	GetCharts(ctx context.Context, periodType, periodValue string) (*models.ChartsResponse, error)
	ApplyOverride(ctx context.Context, supplierUUID, periodType, periodValue, grade, remarks, actor string) error
}

type repo struct{ db *gorm.DB }

type aggregatedSnapshotRow struct {
	SupplierUUID            string     `gorm:"column:supplier_uuid"`
	SupplierCode            string     `gorm:"column:supplier_code"`
	SupplierName            string     `gorm:"column:supplier_name"`
	EvaluationDate          time.Time  `gorm:"column:evaluation_date"`
	TotalDeliveries         int        `gorm:"column:total_deliveries"`
	OnTimeDeliveries        int        `gorm:"column:on_time_deliveries"`
	LateDeliveries          int        `gorm:"column:late_deliveries"`
	AverageDelayDays        float64    `gorm:"column:average_delay_days"`
	QualityInspectionCount  int        `gorm:"column:quality_inspection_count"`
	AcceptedQuantity        float64    `gorm:"column:accepted_quantity"`
	RejectedQuantity        float64    `gorm:"column:rejected_quantity"`
	TotalPurchaseValue      float64    `gorm:"column:total_purchase_value"`
	FinalGrade             string     `gorm:"column:final_grade"`
	StatusLabel            string     `gorm:"column:status_label"`
	PoorDeliveryPerformance bool      `gorm:"column:poor_delivery_performance"`
	QCAlert                bool       `gorm:"column:qc_alert"`
	SupplierReviewRequired bool       `gorm:"column:supplier_review_required"`
	IsGradeOverridden      bool       `gorm:"column:is_grade_overridden"`
	OverrideGrade          *string    `gorm:"column:override_grade"`
	OverrideRemarks        *string    `gorm:"column:override_remarks"`
	OverrideBy             *string    `gorm:"column:override_by"`
	OverrideAt             *time.Time `gorm:"column:override_at"`
	ComputedAt             time.Time  `gorm:"column:computed_at"`
	LogicVersion           string     `gorm:"column:logic_version"`
	FormulaOTD             string     `gorm:"column:formula_otd"`
	FormulaQuality         string     `gorm:"column:formula_quality"`
	FormulaGrade           string     `gorm:"column:formula_grade"`
	FormulaNotes           *string    `gorm:"column:formula_notes"`
}

func New(db *gorm.DB) IRepository { return &repo{db: db} }

func (r *repo) ResolveLatestPeriodValue(ctx context.Context, periodType string) (string, error) {
	var value string

	switch strings.ToLower(strings.TrimSpace(periodType)) {
	case "date":
		if err := r.db.WithContext(ctx).Raw(`
			SELECT COALESCE(MAX(evaluation_period_value), '')
			FROM supplier_performance_snapshots
			WHERE deleted_at IS NULL
			  AND evaluation_period_type = 'daily'
		`).Scan(&value).Error; err != nil {
			return "", fmt.Errorf("ResolveLatestPeriodValue date: %w", err)
		}
	case "yearly":
		if err := r.db.WithContext(ctx).Raw(`
			SELECT COALESCE(MAX(LEFT(evaluation_period_value, 4)), '')
			FROM supplier_performance_snapshots
			WHERE deleted_at IS NULL
			  AND evaluation_period_type = 'daily'
		`).Scan(&value).Error; err != nil {
			return "", fmt.Errorf("ResolveLatestPeriodValue yearly: %w", err)
		}
	default:
		if err := r.db.WithContext(ctx).Raw(`
			SELECT COALESCE(MAX(LEFT(evaluation_period_value, 7)), '')
			FROM supplier_performance_snapshots
			WHERE deleted_at IS NULL
			  AND evaluation_period_type = 'daily'
		`).Scan(&value).Error; err != nil {
			return "", fmt.Errorf("ResolveLatestPeriodValue monthly: %w", err)
		}
	}

	return value, nil
}

func (r *repo) ListSnapshots(ctx context.Context, p pagination.SupplierPerformancePaginationInput) ([]models.Snapshot, int64, error) {
	snapshots, err := r.listAggregatedSnapshots(ctx, p.PeriodType, p.PeriodValue)
	if err != nil {
		return nil, 0, err
	}

	filtered := filterSnapshots(snapshots, p.Search, p.Status)
	sortSnapshots(filtered, p.SortBy, p.SortDirection)

	total := int64(len(filtered))
	start := p.Offset()
	if start >= len(filtered) {
		return []models.Snapshot{}, total, nil
	}

	end := start + p.Limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], total, nil
}

func (r *repo) GetSummary(ctx context.Context, periodType, periodValue string) (*models.SummaryResponse, error) {
	snapshots, err := r.listAggregatedSnapshots(ctx, periodType, periodValue)
	if err != nil {
		return nil, err
	}

	resp := &models.SummaryResponse{}
	for _, snap := range snapshots {
		resp.TotalPurchaseValue += snap.TotalPurchaseValue
		resp.TotalSuppliersEvaluated++
		if snap.ComputedAt.After(resp.ComputedAt) {
			resp.ComputedAt = snap.ComputedAt
			resp.LogicVersion = snap.LogicVersion
			resp.FormulaGrade = snap.FormulaGrade
		}
		switch snap.StatusLabel {
		case "Excellent":
			resp.ExcellentSuppliers++
		case "Good":
			resp.GoodSuppliers++
		case "Review Required":
			resp.ReviewRequiredSuppliers++
		}
	}

	return resp, nil
}

func (r *repo) GetCharts(ctx context.Context, periodType, periodValue string) (*models.ChartsResponse, error) {
	snapshots, err := r.listAggregatedSnapshots(ctx, periodType, periodValue)
	if err != nil {
		return nil, err
	}

	trend, err := r.listTrend(ctx, periodType, periodValue, snapshots)
	if err != nil {
		return nil, err
	}

	scatter := make([]models.ChartScatterPoint, 0, len(snapshots))
	for _, snap := range snapshots {
		scatter = append(scatter, models.ChartScatterPoint{
			SupplierID:        snap.SupplierUUID,
			SupplierName:      snap.SupplierName,
			OTDPercentage:     snap.OTDPercentage,
			QualityPercentage: snap.QualityPercentage,
			StatusLabel:       snap.StatusLabel,
		})
	}

	latestValue, err := r.ResolveLatestPeriodValue(ctx, periodType)
	if err != nil {
		return nil, err
	}

	latestSnapshots := snapshots
	if latestValue != "" && latestValue != periodValue {
		latestSnapshots, err = r.listAggregatedSnapshots(ctx, periodType, latestValue)
		if err != nil {
			return nil, err
		}
	}

	return &models.ChartsResponse{
		Trend:         trend,
		Scatter:       scatter,
		Top5:          rankSnapshots(snapshots, true),
		Bottom5:       rankSnapshots(snapshots, false),
		Top5Latest:    rankSnapshots(latestSnapshots, true),
		Bottom5Latest: rankSnapshots(latestSnapshots, false),
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
			    override_by         = ?,
			    override_at         = NOW(),
			    final_grade         = ?,
			    status_label        = ?,
			    updated_at          = NOW()
			WHERE supplier_uuid = ?
			  AND evaluation_period_type = ?
			  AND evaluation_period_value = ?
			  AND deleted_at IS NULL
		`, grade, remarks, actor, grade, statusLabel, supplierUUID, periodType, periodValue).Error; err != nil {
			return fmt.Errorf("update snapshot: %w", err)
		}

		return nil
	})
}

func (r *repo) listAggregatedSnapshots(ctx context.Context, periodType, periodValue string) ([]models.Snapshot, error) {
	if strings.TrimSpace(periodValue) == "" {
		return []models.Snapshot{}, nil
	}

	rows, err := r.listAggregateRows(ctx, periodType, periodValue)
	if err != nil {
		return nil, err
	}

	snapshots := make([]models.Snapshot, 0, len(rows))
	for _, row := range rows {
		snapshots = append(snapshots, aggregateRowToSnapshot(periodType, periodValue, row))
	}

	return snapshots, nil
}

func (r *repo) listAggregateRows(ctx context.Context, periodType, periodValue string) ([]aggregatedSnapshotRow, error) {
	whereClause, args, err := periodFilterClause(periodType, periodValue)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		SELECT
			supplier_uuid,
			MAX(supplier_code)                                                                  AS supplier_code,
			MAX(supplier_name)                                                                  AS supplier_name,
			MAX(evaluation_date)                                                                AS evaluation_date,
			COALESCE(SUM(total_deliveries), 0)                                                  AS total_deliveries,
			COALESCE(SUM(on_time_deliveries), 0)                                                AS on_time_deliveries,
			COALESCE(SUM(late_deliveries), 0)                                                   AS late_deliveries,
			COALESCE(SUM(average_delay_days * late_deliveries) / NULLIF(SUM(late_deliveries), 0), 0) AS average_delay_days,
			COALESCE(SUM(quality_inspection_count), 0)                                          AS quality_inspection_count,
			COALESCE(SUM(accepted_quantity), 0)                                                 AS accepted_quantity,
			COALESCE(SUM(rejected_quantity), 0)                                                 AS rejected_quantity,
			COALESCE(SUM(total_purchase_value), 0)                                              AS total_purchase_value,
			MAX(final_grade)                                                                    AS final_grade,
			MAX(status_label)                                                                   AS status_label,
			BOOL_OR(poor_delivery_performance)                                                  AS poor_delivery_performance,
			BOOL_OR(qc_alert)                                                                   AS qc_alert,
			BOOL_OR(supplier_review_required)                                                   AS supplier_review_required,
			BOOL_OR(is_grade_overridden)                                                        AS is_grade_overridden,
			MAX(override_grade)                                                                 AS override_grade,
			MAX(override_remarks)                                                               AS override_remarks,
			MAX(override_by)                                                                    AS override_by,
			MAX(override_at)                                                                    AS override_at,
			MAX(computed_at)                                                                    AS computed_at,
			MAX(logic_version)                                                                  AS logic_version,
			MAX(formula_otd)                                                                    AS formula_otd,
			MAX(formula_quality)                                                                AS formula_quality,
			MAX(formula_grade)                                                                  AS formula_grade,
			MAX(formula_notes)                                                                  AS formula_notes
		FROM supplier_performance_snapshots
		WHERE deleted_at IS NULL
		  AND evaluation_period_type = 'daily'
		  AND %s
		GROUP BY supplier_uuid
	`, whereClause)

	var rows []aggregatedSnapshotRow
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("listAggregateRows: %w", err)
	}

	return rows, nil
}

func (r *repo) listTrend(ctx context.Context, periodType, periodValue string, snapshots []models.Snapshot) ([]models.ChartTrendPoint, error) {
	periodType = strings.ToLower(strings.TrimSpace(periodType))

	if periodType == "date" {
		if len(snapshots) == 0 {
			return []models.ChartTrendPoint{}, nil
		}

		var totalOTD float64
		var totalQuality float64
		for _, snap := range snapshots {
			totalOTD += snap.OTDPercentage
			totalQuality += snap.QualityPercentage
		}

		return []models.ChartTrendPoint{{
			Period:               periodValue,
			AvgOTDPercentage:     round2(totalOTD / float64(len(snapshots))),
			AvgQualityPercentage: round2(totalQuality / float64(len(snapshots))),
		}}, nil
	}

	var query string
	var args []interface{}
	if periodType == "yearly" {
		query = `
			SELECT
				LEFT(evaluation_period_value, 7)                  AS period,
				ROUND(AVG(otd_percentage)::numeric, 2)            AS avg_otd_percentage,
				ROUND(AVG(quality_percentage)::numeric, 2)        AS avg_quality_percentage
			FROM supplier_performance_snapshots
			WHERE deleted_at IS NULL
			  AND evaluation_period_type = 'daily'
			  AND LEFT(evaluation_period_value, 4) = ?
			GROUP BY LEFT(evaluation_period_value, 7)
			ORDER BY LEFT(evaluation_period_value, 7)
		`
		args = []interface{}{periodValue}
	} else {
		query = `
			SELECT
				evaluation_period_value                           AS period,
				ROUND(AVG(otd_percentage)::numeric, 2)            AS avg_otd_percentage,
				ROUND(AVG(quality_percentage)::numeric, 2)        AS avg_quality_percentage
			FROM supplier_performance_snapshots
			WHERE deleted_at IS NULL
			  AND evaluation_period_type = 'daily'
			  AND LEFT(evaluation_period_value, 7) = ?
			GROUP BY evaluation_period_value
			ORDER BY evaluation_period_value
		`
		args = []interface{}{periodValue}
	}

	var trend []models.ChartTrendPoint
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&trend).Error; err != nil {
		return nil, fmt.Errorf("listTrend: %w", err)
	}
	if trend == nil {
		return []models.ChartTrendPoint{}, nil
	}
	return trend, nil
}

func periodFilterClause(periodType, periodValue string) (string, []interface{}, error) {
	switch strings.ToLower(strings.TrimSpace(periodType)) {
	case "date":
		return "evaluation_period_value = ?", []interface{}{periodValue}, nil
	case "yearly":
		return "LEFT(evaluation_period_value, 4) = ?", []interface{}{periodValue}, nil
	case "monthly":
		return "LEFT(evaluation_period_value, 7) = ?", []interface{}{periodValue}, nil
	default:
		return "", nil, fmt.Errorf("unsupported period_type %q", periodType)
	}
}

func aggregateRowToSnapshot(periodType, periodValue string, row aggregatedSnapshotRow) models.Snapshot {
	inspectedQuantity := row.AcceptedQuantity + row.RejectedQuantity
	otdPercentage := 0.0
	if row.TotalDeliveries > 0 {
		otdPercentage = round2(float64(row.OnTimeDeliveries) / float64(row.TotalDeliveries) * 100)
	}

	qualityPercentage := 0.0
	if inspectedQuantity > 0 {
		qualityPercentage = round2(row.AcceptedQuantity / inspectedQuantity * 100)
	}

	computedScore := round2((qualityPercentage * 0.5) + (otdPercentage * 0.5))
	performanceGrade := gradeFromScore(computedScore)
	statusLabel := statusLabelFromGrade(performanceGrade)
	poorDeliveryPerformance := otdPercentage < 80
	qcAlert := qualityPercentage < 90
	supplierReviewRequired := performanceGrade == "C"
	isGradeOverridden := false
	var overrideGrade *string
	var overrideRemarks *string
	var overrideBy *string
	var overrideAt *time.Time

	if strings.EqualFold(periodType, "date") {
		performanceGrade = row.FinalGrade
		if performanceGrade == "" {
			performanceGrade = gradeFromScore(computedScore)
		}
		statusLabel = row.StatusLabel
		if statusLabel == "" {
			statusLabel = statusLabelFromGrade(performanceGrade)
		}
		poorDeliveryPerformance = row.PoorDeliveryPerformance
		qcAlert = row.QCAlert
		supplierReviewRequired = row.SupplierReviewRequired
		isGradeOverridden = row.IsGradeOverridden
		overrideGrade = row.OverrideGrade
		overrideRemarks = row.OverrideRemarks
		overrideBy = row.OverrideBy
		overrideAt = row.OverrideAt
	}

	return models.Snapshot{
		SupplierUUID:            row.SupplierUUID,
		SupplierCode:            row.SupplierCode,
		SupplierName:            row.SupplierName,
		EvaluationPeriodType:    periodType,
		EvaluationPeriodValue:   periodValue,
		EvaluationDate:          row.EvaluationDate,
		TotalDeliveries:         row.TotalDeliveries,
		OnTimeDeliveries:        row.OnTimeDeliveries,
		LateDeliveries:          row.LateDeliveries,
		OTDPercentage:           otdPercentage,
		AverageDelayDays:        round2(row.AverageDelayDays),
		QualityInspectionCount:  row.QualityInspectionCount,
		AcceptedQuantity:        row.AcceptedQuantity,
		RejectedQuantity:        row.RejectedQuantity,
		InspectedQuantity:       inspectedQuantity,
		QualityPercentage:       qualityPercentage,
		TotalPurchaseValue:      row.TotalPurchaseValue,
		ComputedScore:           computedScore,
		PerformanceGrade:        performanceGrade,
		StatusLabel:             statusLabel,
		PoorDeliveryPerformance: poorDeliveryPerformance,
		QCAlert:                 qcAlert,
		SupplierReviewRequired:  supplierReviewRequired,
		IsGradeOverridden:       isGradeOverridden,
		OverrideGrade:           overrideGrade,
		OverrideRemarks:         overrideRemarks,
		OverrideBy:              overrideBy,
		OverrideAt:              overrideAt,
		ComputedAt:              row.ComputedAt,
		LogicVersion:            row.LogicVersion,
		FormulaOTD:              row.FormulaOTD,
		FormulaQuality:          row.FormulaQuality,
		FormulaGrade:            row.FormulaGrade,
		FormulaNotes:            row.FormulaNotes,
	}
}

func filterSnapshots(snapshots []models.Snapshot, search, status string) []models.Snapshot {
	search = strings.ToLower(strings.TrimSpace(search))
	status = strings.ToLower(strings.TrimSpace(status))

	filtered := make([]models.Snapshot, 0, len(snapshots))
	for _, snap := range snapshots {
		if search != "" {
			name := strings.ToLower(snap.SupplierName)
			code := strings.ToLower(snap.SupplierCode)
			if !strings.Contains(name, search) && !strings.Contains(code, search) {
				continue
			}
		}

		if !matchesStatusFilter(snap.StatusLabel, status) {
			continue
		}

		filtered = append(filtered, snap)
	}

	return filtered
}

func matchesStatusFilter(statusLabel, status string) bool {
	if status == "" {
		return true
	}

	switch status {
	case "excellent":
		return statusLabel == "Excellent"
	case "good":
		return statusLabel == "Good"
	case "review_required":
		return statusLabel == "Review Required"
	default:
		return true
	}
}

func sortSnapshots(snapshots []models.Snapshot, sortBy, sortDirection string) {
	field := "computed_at"
	if map[string]bool{
		"supplier_name":       true,
		"otd_percentage":      true,
		"quality_percentage":  true,
		"computed_score":      true,
		"computed_at":         true,
		"total_purchase_value": true,
	}[sortBy] {
		field = sortBy
	}

	asc := strings.EqualFold(sortDirection, "asc")
	sort.SliceStable(snapshots, func(i, j int) bool {
		left := snapshots[i]
		right := snapshots[j]

		compare := 0
		switch field {
		case "supplier_name":
			compare = strings.Compare(strings.ToLower(left.SupplierName), strings.ToLower(right.SupplierName))
		case "otd_percentage":
			compare = compareFloat64(left.OTDPercentage, right.OTDPercentage)
		case "quality_percentage":
			compare = compareFloat64(left.QualityPercentage, right.QualityPercentage)
		case "total_purchase_value":
			compare = compareFloat64(left.TotalPurchaseValue, right.TotalPurchaseValue)
		case "computed_score":
			compare = compareFloat64(left.ComputedScore, right.ComputedScore)
		default:
			compare = compareTime(left.ComputedAt, right.ComputedAt)
		}

		if compare == 0 {
			compare = strings.Compare(strings.ToLower(left.SupplierName), strings.ToLower(right.SupplierName))
		}

		if asc {
			return compare < 0
		}
		return compare > 0
	})
}

func rankSnapshots(snapshots []models.Snapshot, descending bool) []models.ChartRankPoint {
	if len(snapshots) == 0 {
		return []models.ChartRankPoint{}
	}

	cloned := append([]models.Snapshot(nil), snapshots...)
	sort.SliceStable(cloned, func(i, j int) bool {
		if descending {
			if cloned[i].ComputedScore == cloned[j].ComputedScore {
				return strings.ToLower(cloned[i].SupplierName) < strings.ToLower(cloned[j].SupplierName)
			}
			return cloned[i].ComputedScore > cloned[j].ComputedScore
		}
		if cloned[i].ComputedScore == cloned[j].ComputedScore {
			return strings.ToLower(cloned[i].SupplierName) < strings.ToLower(cloned[j].SupplierName)
		}
		return cloned[i].ComputedScore < cloned[j].ComputedScore
	})

	limit := 5
	if len(cloned) < limit {
		limit = len(cloned)
	}

	ranks := make([]models.ChartRankPoint, 0, limit)
	for _, snap := range cloned[:limit] {
		ranks = append(ranks, models.ChartRankPoint{
			SupplierID:       snap.SupplierUUID,
			SupplierName:     snap.SupplierName,
			PerformanceGrade: snap.PerformanceGrade,
			StatusLabel:      snap.StatusLabel,
			Score:            snap.ComputedScore,
		})
	}

	return ranks
}

func gradeFromScore(score float64) string {
	if score >= 90 {
		return "A"
	}
	if score >= 80 {
		return "B"
	}
	return "C"
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

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func compareFloat64(left, right float64) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func compareTime(left, right time.Time) int {
	if left.Before(right) {
		return -1
	}
	if left.After(right) {
		return 1
	}
	return 0
}
