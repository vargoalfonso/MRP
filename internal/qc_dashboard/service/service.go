package service

import (
	"context"
	"time"

	"github.com/ganasa18/go-template/internal/qc_dashboard/models"
	"github.com/ganasa18/go-template/internal/qc_dashboard/repository"
	"github.com/ganasa18/go-template/pkg/concurrency"
)

type IService interface {
	GetOverview(ctx context.Context, filter repository.Filter) (*models.OverviewResponse, error)
	ListProductionQC(ctx context.Context, filter repository.Filter) (*models.ProductionQCListResponse, error)
	ListIncomingQC(ctx context.Context, filter repository.Filter) (*models.IncomingQCListResponse, error)
	ListProductReturnQC(ctx context.Context, filter repository.Filter) (*models.ProductReturnQCListResponse, error)
	ListDefects(ctx context.Context, filter repository.Filter) (*models.DefectListResponse, error)
	ListIssueTypes() []models.IssueListItem
	CreateManualQCReport(ctx context.Context, req models.CreateManualQCReportRequest, performedBy string) error
	CreateReworkTask(ctx context.Context, defectID int64, performedBy string) error
}

type service struct {
	repo   repository.IRepository
	fanout concurrency.FanoutOptions
}

func New(repo repository.IRepository, fanout concurrency.FanoutOptions) IService {
	return &service{repo: repo, fanout: fanout}
}

func (s *service) GetOverview(ctx context.Context, filter repository.Filter) (*models.OverviewResponse, error) {
	var (
		cards         models.OverviewCards
		asOf          = filterDateFallback()
		bySource      []models.SourceSummary
		topIssues     []models.IssueSummary
		pendingRework int64
	)

	err := concurrency.Run(ctx, []concurrency.Task{
		func(c context.Context) error {
			result, resultAsOf, err := s.repo.GetOverviewCards(c, filter)
			if err != nil {
				return err
			}
			cards = result
			asOf = resultAsOf
			return nil
		},
		func(c context.Context) error {
			result, err := s.repo.GetOverviewBySource(c, filter)
			if err != nil {
				return err
			}
			bySource = result
			return nil
		},
		func(c context.Context) error {
			result, err := s.repo.GetTopIssues(c, filter, 5)
			if err != nil {
				return err
			}
			topIssues = result
			return nil
		},
		func(c context.Context) error {
			result, err := s.repo.CountPendingRework(c)
			if err != nil {
				return err
			}
			pendingRework = result
			return nil
		},
	}, s.fanout)
	if err != nil {
		return nil, err
	}

	cards.PendingRework = pendingRework
	return &models.OverviewResponse{
		AsOf:               asOf,
		WindowHours:        filter.WindowHours,
		Cards:              cards,
		BySource:           bySource,
		TopIssues:          topIssues,
		ImplementationNote: "Overview sudah mencakup production, incoming, dan product return QC bila datanya sudah tercatat.",
	}, nil
}

func (s *service) ListProductionQC(ctx context.Context, filter repository.Filter) (*models.ProductionQCListResponse, error) {
	return s.repo.ListProductionQC(ctx, filter)
}

func (s *service) ListIncomingQC(ctx context.Context, filter repository.Filter) (*models.IncomingQCListResponse, error) {
	return s.repo.ListIncomingQC(ctx, filter)
}

func (s *service) ListProductReturnQC(ctx context.Context, filter repository.Filter) (*models.ProductReturnQCListResponse, error) {
	return s.repo.ListProductReturnQC(ctx, filter)
}

func (s *service) ListDefects(ctx context.Context, filter repository.Filter) (*models.DefectListResponse, error) {
	return s.repo.ListDefects(ctx, filter)
}

func (s *service) ListIssueTypes() []models.IssueListItem { return models.IssueList }

func (s *service) CreateManualQCReport(ctx context.Context, req models.CreateManualQCReportRequest, performedBy string) error {
	return s.repo.CreateManualQCReport(ctx, req, performedBy)
}

func (s *service) CreateReworkTask(ctx context.Context, defectID int64, performedBy string) error {
	return s.repo.CreateReworkTask(ctx, defectID, performedBy)
}

func filterDateFallback() time.Time {
	return time.Now().UTC()
}
