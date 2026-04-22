package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/admin_jobs/repository"
	"github.com/google/uuid"
)

const (
	defaultLogicVersion   = "v1"
	defaultFormulaOTD     = "(on_time_deliveries / total_deliveries) * 100"
	defaultFormulaQuality = "(accepted_quantity / (accepted_quantity + rejected_quantity)) * 100"
	defaultFormulaGrade   = "(quality_percentage * 0.5) + (otd_percentage * 0.5)"
	defaultFormulaNotes   = "OTD based on receipt_date <= period end; grade A >= 90, B 80-89.99, C < 80"
)

// RecomputeSupplierPerformanceRequest is the minimal payload for the recompute job.
// period_value is auto-resolved from the current date based on period_type.
type RecomputeSupplierPerformanceRequest struct {
	PeriodType string `json:"period_type"`
}

// supplierPerformanceAggregateRow is the internal service representation of aggregated data.
type supplierPerformanceAggregateRow struct {
	SupplierUUID           string
	SupplierCode           string
	SupplierName           string
	TotalPurchaseValue     float64
	OnTimeDeliveries       int
	LateDeliveries         int
	AverageDelayDays       float64
	QualityInspectionCount int
	AcceptedQuantity       float64
	RejectedQuantity       float64
}

// supplierPerformanceSnapshotRow is the fully-computed row before mapping to repository type.
type supplierPerformanceSnapshotRow struct {
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

func (s *service) RecomputeSupplierPerformance(ctx context.Context, req RecomputeSupplierPerformanceRequest) (int64, error) {
	periodValue := resolvePeriodValue(req.PeriodType)

	repoRows, err := s.repo.ListSupplierPerformanceAggregates(ctx, req.PeriodType, periodValue)
	if err != nil {
		return 0, err
	}

	snapshots := make([]repository.SupplierPerformanceSnapshotRow, 0, len(repoRows))
	for _, r := range repoRows {
		agg := supplierPerformanceAggregateRow{
			SupplierUUID:           r.SupplierUUID,
			SupplierCode:           r.SupplierCode,
			SupplierName:           r.SupplierName,
			TotalPurchaseValue:     r.TotalPurchaseValue,
			OnTimeDeliveries:       r.OnTimeDeliveries,
			LateDeliveries:         r.LateDeliveries,
			AverageDelayDays:       r.AverageDelayDays,
			QualityInspectionCount: r.QualityInspectionCount,
			AcceptedQuantity:       r.AcceptedQuantity,
			RejectedQuantity:       r.RejectedQuantity,
		}
		snap := buildSupplierPerformanceSnapshot(agg, req.PeriodType, periodValue)
		snapshots = append(snapshots, toRepositorySnapshotRow(snap))
	}

	return s.repo.UpsertSupplierPerformanceSnapshots(ctx, snapshots)
}

// resolvePeriodValue derives the period_value string from the current date.
// monthly → "2026-04", quarterly → "2026-Q2", yearly → "2026"
func resolvePeriodValue(periodType string) string {
	now := time.Now().UTC()
	switch strings.ToLower(periodType) {
	case "quarterly":
		q := (int(now.Month())-1)/3 + 1
		return fmt.Sprintf("%d-Q%d", now.Year(), q)
	case "yearly":
		return fmt.Sprintf("%d", now.Year())
	default: // monthly
		return now.Format("2006-01")
	}
}

func buildSupplierPerformanceSnapshot(row supplierPerformanceAggregateRow, periodType, periodValue string) supplierPerformanceSnapshotRow {
	total := row.OnTimeDeliveries + row.LateDeliveries

	var otdPct float64
	if total > 0 {
		otdPct = round2sp(float64(row.OnTimeDeliveries) / float64(total) * 100)
	}

	accepted := row.AcceptedQuantity
	rejected := row.RejectedQuantity
	inspected := accepted + rejected

	var qualityPct float64
	if inspected > 0 {
		qualityPct = round2sp(accepted / inspected * 100)
	}

	score := round2sp((qualityPct * 0.5) + (otdPct * 0.5))

	grade := "C"
	if score >= 90 {
		grade = "A"
	} else if score >= 80 {
		grade = "B"
	}

	statusLabel := spGradeToStatusLabel(grade)

	logicVersion := defaultLogicVersion
	formulaOTD := defaultFormulaOTD
	formulaQuality := defaultFormulaQuality
	formulaGrade := defaultFormulaGrade
	formulaNotes := defaultFormulaNotes

	evalDate := periodEndDate(periodType, periodValue)

	return supplierPerformanceSnapshotRow{
		SnapshotUUID:            uuid.New().String(),
		SupplierUUID:            row.SupplierUUID,
		SupplierCode:            row.SupplierCode,
		SupplierName:            row.SupplierName,
		EvaluationPeriodType:    periodType,
		EvaluationPeriodValue:   periodValue,
		EvaluationDate:          evalDate,
		TotalDeliveries:         total,
		OnTimeDeliveries:        row.OnTimeDeliveries,
		LateDeliveries:          row.LateDeliveries,
		OTDPercentage:           otdPct,
		AverageDelayDays:        row.AverageDelayDays,
		QualityInspectionCount:  row.QualityInspectionCount,
		AcceptedQuantity:        accepted,
		RejectedQuantity:        rejected,
		InspectedQuantity:       inspected,
		QualityPercentage:       qualityPct,
		TotalPurchaseValue:      row.TotalPurchaseValue,
		ComputedScore:           score,
		SystemGrade:             grade,
		FinalGrade:              grade,
		StatusLabel:             statusLabel,
		PoorDeliveryPerformance: otdPct < 80,
		QCAlert:                 qualityPct < 90,
		SupplierReviewRequired:  grade == "C",
		LogicVersion:            logicVersion,
		FormulaOTD:              formulaOTD,
		FormulaQuality:          formulaQuality,
		FormulaGrade:            formulaGrade,
		FormulaNotes:            formulaNotes,
	}
}

func toRepositorySnapshotRow(s supplierPerformanceSnapshotRow) repository.SupplierPerformanceSnapshotRow {
	return repository.SupplierPerformanceSnapshotRow{
		SnapshotUUID:            s.SnapshotUUID,
		SupplierUUID:            s.SupplierUUID,
		SupplierCode:            s.SupplierCode,
		SupplierName:            s.SupplierName,
		EvaluationPeriodType:    s.EvaluationPeriodType,
		EvaluationPeriodValue:   s.EvaluationPeriodValue,
		EvaluationDate:          s.EvaluationDate,
		TotalDeliveries:         s.TotalDeliveries,
		OnTimeDeliveries:        s.OnTimeDeliveries,
		LateDeliveries:          s.LateDeliveries,
		OTDPercentage:           s.OTDPercentage,
		AverageDelayDays:        s.AverageDelayDays,
		QualityInspectionCount:  s.QualityInspectionCount,
		AcceptedQuantity:        s.AcceptedQuantity,
		RejectedQuantity:        s.RejectedQuantity,
		InspectedQuantity:       s.InspectedQuantity,
		QualityPercentage:       s.QualityPercentage,
		TotalPurchaseValue:      s.TotalPurchaseValue,
		ComputedScore:           s.ComputedScore,
		SystemGrade:             s.SystemGrade,
		FinalGrade:              s.FinalGrade,
		StatusLabel:             s.StatusLabel,
		PoorDeliveryPerformance: s.PoorDeliveryPerformance,
		QCAlert:                 s.QCAlert,
		SupplierReviewRequired:  s.SupplierReviewRequired,
		LogicVersion:            s.LogicVersion,
		FormulaOTD:              s.FormulaOTD,
		FormulaQuality:          s.FormulaQuality,
		FormulaGrade:            s.FormulaGrade,
		FormulaNotes:            s.FormulaNotes,
	}
}

func spGradeToStatusLabel(grade string) string {
	switch strings.ToUpper(grade) {
	case "A":
		return "Excellent"
	case "B":
		return "Good"
	default:
		return "Review Required"
	}
}

func round2sp(v float64) float64 {
	return math.Round(v*100) / 100
}

// periodEndDate returns the last day of the period as time.Time.
func periodEndDate(periodType, periodValue string) time.Time {
	switch strings.ToLower(periodType) {
	case "monthly":
		t, err := time.Parse("2006-01", periodValue)
		if err != nil {
			return time.Now().UTC()
		}
		return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	case "quarterly":
		var year, quarter int
		if _, err := fmt.Sscanf(periodValue, "%d-Q%d", &year, &quarter); err != nil {
			return time.Now().UTC()
		}
		endMonth := time.Month(quarter * 3)
		return time.Date(year, endMonth+1, 0, 0, 0, 0, 0, time.UTC)
	case "yearly":
		t, err := time.Parse("2006", periodValue)
		if err != nil {
			return time.Now().UTC()
		}
		return time.Date(t.Year(), 12, 31, 0, 0, 0, 0, time.UTC)
	}
	return time.Now().UTC()
}
