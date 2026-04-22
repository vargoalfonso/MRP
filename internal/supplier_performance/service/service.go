package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/supplier_performance/models"
	"github.com/ganasa18/go-template/internal/supplier_performance/repository"
	"github.com/ganasa18/go-template/pkg/pagination"
)

type IService interface {
	List(ctx context.Context, p pagination.SupplierPerformancePaginationInput) (*models.ListResponse, error)
	Summary(ctx context.Context, periodType, periodValue string) (*models.SummaryResponse, error)
	Charts(ctx context.Context, periodType, periodValue string) (*models.ChartsResponse, error)
	Override(ctx context.Context, req models.OverrideRequest, actor string) error
	AuditLogs(ctx context.Context, supplierUUID, periodType, periodValue string) ([]models.AuditLog, error)
	Export(ctx context.Context, p pagination.SupplierPerformancePaginationInput) (*models.ExportResponse, error)
}

type service struct {
	repo repository.IRepository
}

func New(repo repository.IRepository) IService { return &service{repo: repo} }

func (s *service) List(ctx context.Context, p pagination.SupplierPerformancePaginationInput) (*models.ListResponse, error) {
	p = normalizeInput(p)
	rows, total, err := s.repo.ListSnapshots(ctx, p)
	if err != nil {
		return nil, err
	}

	items := make([]models.SnapshotResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, toResponse(r))
	}

	pages := 0
	if p.Limit > 0 {
		pages = int((total + int64(p.Limit) - 1) / int64(p.Limit))
	}

	return &models.ListResponse{
		Items: items,
		Pagination: models.PaginationMeta{
			Total:      total,
			Page:       p.Page,
			Limit:      p.Limit,
			TotalPages: pages,
		},
	}, nil
}

func (s *service) Summary(ctx context.Context, periodType, periodValue string) (*models.SummaryResponse, error) {
	if periodValue == "" {
		periodValue = resolvePeriodValue(periodType)
	}
	return s.repo.GetSummary(ctx, periodType, periodValue)
}

func (s *service) Charts(ctx context.Context, periodType, periodValue string) (*models.ChartsResponse, error) {
	if periodValue == "" {
		periodValue = resolvePeriodValue(periodType)
	}
	return s.repo.GetCharts(ctx, periodType, periodValue)
}

func (s *service) Override(ctx context.Context, req models.OverrideRequest, actor string) error {
	return s.repo.ApplyOverride(ctx, req.SupplierUUID, req.PeriodType, req.PeriodValue, req.OverrideGrade, req.Remarks, actor)
}

func (s *service) AuditLogs(ctx context.Context, supplierUUID, periodType, periodValue string) ([]models.AuditLog, error) {
	return s.repo.ListAuditLogs(ctx, supplierUUID, periodType, periodValue)
}

func (s *service) Export(ctx context.Context, p pagination.SupplierPerformancePaginationInput) (*models.ExportResponse, error) {
	p = normalizeInput(p)
	p.Limit = 10000
	p.Page = 1

	rows, _, err := s.repo.ListSnapshots(ctx, p)
	if err != nil {
		return nil, err
	}

	items := make([]models.SnapshotResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, toResponse(r))
	}

	resp := &models.ExportResponse{
		PeriodType:  p.PeriodType,
		PeriodValue: p.PeriodValue,
		ExportedAt:  time.Now().UTC(),
		Items:       items,
	}
	if len(rows) > 0 {
		resp.LogicVersion = rows[0].LogicVersion
		resp.FormulaOTD = rows[0].FormulaOTD
		resp.FormulaQuality = rows[0].FormulaQuality
		resp.FormulaGrade = rows[0].FormulaGrade
	}
	return resp, nil
}

// --- helpers ---

func resolvePeriodValue(periodType string) string {
	now := time.Now().UTC()
	switch strings.ToLower(periodType) {
	case "quarterly":
		q := (int(now.Month())-1)/3 + 1
		return fmt.Sprintf("%d-Q%d", now.Year(), q)
	case "yearly":
		return fmt.Sprintf("%d", now.Year())
	default:
		return now.Format("2006-01")
	}
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

func normalizeInput(p pagination.SupplierPerformancePaginationInput) pagination.SupplierPerformancePaginationInput {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.Limit <= 0 {
		p.Limit = 20
	}
	if p.Limit > 200 {
		p.Limit = 200
	}
	return p
}

func toResponse(s models.Snapshot) models.SnapshotResponse {
	return models.SnapshotResponse{
		SupplierID:             s.SupplierUUID,
		SupplierCode:           s.SupplierCode,
		SupplierName:           s.SupplierName,
		EvaluationPeriodType:   s.EvaluationPeriodType,
		EvaluationPeriodValue:  s.EvaluationPeriodValue,
		TotalDeliveries:        s.TotalDeliveries,
		OnTimeDeliveries:       s.OnTimeDeliveries,
		LateDeliveries:         s.LateDeliveries,
		OTDPercentage:          s.OTDPercentage,
		AverageDelayDays:       s.AverageDelayDays,
		QualityInspectionCount: s.QualityInspectionCount,
		AcceptedQuantity:       s.AcceptedQuantity,
		RejectedQuantity:       s.RejectedQuantity,
		InspectedQuantity:      s.InspectedQuantity,
		QualityPercentage:      s.QualityPercentage,
		TotalPurchaseValue:     s.TotalPurchaseValue,
		PerformanceGrade:       s.PerformanceGrade,
		StatusLabel:            s.StatusLabel,
		Flags:                  s.Flags(),
		SupplierReviewRequired: s.SupplierReviewRequired,
		IsGradeOverridden:      s.IsGradeOverridden,
		OverrideGrade:          s.OverrideGrade,
		OverrideRemarks:        s.OverrideRemarks,
		LogicVersion:           s.LogicVersion,
		FormulaOTD:             s.FormulaOTD,
		FormulaQuality:         s.FormulaQuality,
		FormulaGrade:           s.FormulaGrade,
		FormulaNotes:           s.FormulaNotes,
		EvaluationDate:         s.EvaluationDate,
	}
}
