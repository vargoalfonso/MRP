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

// RecomputeSupplierPerformanceRequest is the payload for the recompute job.
//
// Ada dua mode:
//
//  1. Satu hari  : snapshot_date (opsional, default hari ini UTC)
//  2. Backfill   : date_from + date_to, mengisi setiap hari dalam rentang
//
// Mode backfill diperlukan karena snapshot disimpan per hari, sedangkan
// halaman Supplier Performance menampilkan periode bulanan/kuartalan/tahunan.
// Tanpa backfill, tampilan bulanan hanya berisi satu hari saja.
type RecomputeSupplierPerformanceRequest struct {
	PeriodType   string `json:"period_type,omitempty"`
	SnapshotDate string `json:"snapshot_date,omitempty"`
	DateFrom     string `json:"date_from,omitempty"`
	DateTo       string `json:"date_to,omitempty"`
}

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
	dates, err := resolveSnapshotDates(req)
	if err != nil {
		return 0, err
	}

	var affected int64

	for _, snapshotDate := range dates {
		repoRows, err := s.repo.ListSupplierPerformanceAggregates(ctx, snapshotDate)
		if err != nil {
			return affected, fmt.Errorf("aggregate %s: %w", snapshotDate, err)
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
			snap := buildSupplierPerformanceSnapshot(agg, snapshotDate)
			snapshots = append(snapshots, toRepositorySnapshotRow(snap))
		}

		n, err := s.repo.UpsertSupplierPerformanceSnapshots(ctx, snapshots)
		if err != nil {
			return affected, fmt.Errorf("upsert %s: %w", snapshotDate, err)
		}
		affected += n
	}

	return affected, nil
}

// maxBackfillDays membatasi satu panggilan backfill agar tidak berjalan
// berjam-jam tanpa disadari. Sekitar dua tahun sudah lebih dari cukup.
const maxBackfillDays = 800

// resolveSnapshotDates menentukan daftar tanggal yang akan dihitung.
func resolveSnapshotDates(req RecomputeSupplierPerformanceRequest) ([]string, error) {
	from := strings.TrimSpace(req.DateFrom)
	to := strings.TrimSpace(req.DateTo)

	if from == "" && to == "" {
		day, err := resolveSnapshotDate(req.SnapshotDate)
		if err != nil {
			return nil, err
		}
		return []string{day}, nil
	}

	if from == "" || to == "" {
		return nil, fmt.Errorf("date_from and date_to must be provided together")
	}

	start, err := time.Parse("2006-01-02", from)
	if err != nil {
		return nil, fmt.Errorf("invalid date_from %q: %w", from, err)
	}

	end, err := time.Parse("2006-01-02", to)
	if err != nil {
		return nil, fmt.Errorf("invalid date_to %q: %w", to, err)
	}

	if end.Before(start) {
		return nil, fmt.Errorf("date_to %q is before date_from %q", to, from)
	}

	dates := make([]string, 0)
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if len(dates) >= maxBackfillDays {
			return nil, fmt.Errorf("range too large: maximum %d days per call", maxBackfillDays)
		}
		dates = append(dates, d.Format("2006-01-02"))
	}

	return dates, nil
}

func resolveSnapshotDate(snapshotDate string) (string, error) {
	if snapshotDate == "" {
		return time.Now().UTC().Format("2006-01-02"), nil
	}
	if _, err := time.Parse("2006-01-02", snapshotDate); err != nil {
		return "", fmt.Errorf("invalid snapshot_date %q: %w", snapshotDate, err)
	}
	return snapshotDate, nil
}

func buildSupplierPerformanceSnapshot(row supplierPerformanceAggregateRow, snapshotDate string) supplierPerformanceSnapshotRow {
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

	evalDate, _ := time.Parse("2006-01-02", snapshotDate)

	return supplierPerformanceSnapshotRow{
		SnapshotUUID:            uuid.New().String(),
		SupplierUUID:            row.SupplierUUID,
		SupplierCode:            row.SupplierCode,
		SupplierName:            row.SupplierName,
		EvaluationPeriodType:    "daily",
		EvaluationPeriodValue:   snapshotDate,
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
		StatusLabel:             spGradeToStatusLabel(grade),
		PoorDeliveryPerformance: otdPct < 80,
		QCAlert:                 qualityPct < 90,
		SupplierReviewRequired:  grade == "C",
		LogicVersion:            defaultLogicVersion,
		FormulaOTD:              defaultFormulaOTD,
		FormulaQuality:          defaultFormulaQuality,
		FormulaGrade:            defaultFormulaGrade,
		FormulaNotes:            defaultFormulaNotes,
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
	switch grade {
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
