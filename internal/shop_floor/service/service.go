package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/shop_floor/models"
	"github.com/ganasa18/go-template/internal/shop_floor/repository"
	"github.com/ganasa18/go-template/pkg/concurrency"
)

type IService interface {
	GetLiveProductionSummary(ctx context.Context, limit, staleMinutes int) (*models.LiveProductionSummary, error)
	GetDeliveryReadinessSummary(ctx context.Context, limit int) (*models.DeliveryReadinessSummary, error)
	GetProductionIssuesSummary(ctx context.Context, limit, windowHours int) (*models.ProductionIssuesSummary, error)
	GetScanEventsSummary(ctx context.Context, limit, windowHours int) (*models.ScanEventsSummary, error)
}

type service struct {
	repo   repository.IRepository
	fanout concurrency.FanoutOptions
}

func New(repo repository.IRepository, fanout concurrency.FanoutOptions) IService {
	return &service{repo: repo, fanout: fanout}
}

func (s *service) GetLiveProductionSummary(ctx context.Context, limit, staleMinutes int) (*models.LiveProductionSummary, error) {
	filter := repository.Filter{Limit: limit, StaleMinutes: staleMinutes}
	var (
		totals *models.LiveProductionSummary
		items  []models.LiveProduction
	)
	err := concurrency.Run(ctx, []concurrency.Task{
		func(c context.Context) error {
			result, err := s.repo.GetLiveProductionTotals(c, filter)
			if err != nil {
				return err
			}
			totals = result
			return nil
		},
		func(c context.Context) error {
			result, err := s.repo.GetLiveProductionItems(c, filter)
			if err != nil {
				return err
			}
			items = result
			return nil
		},
	}, s.fanout)
	if err != nil {
		return nil, err
	}
	totals.Items = items
	return totals, nil
}

func (s *service) GetDeliveryReadinessSummary(ctx context.Context, limit int) (*models.DeliveryReadinessSummary, error) {
	filter := repository.Filter{Limit: limit}
	var (
		totals *models.DeliveryReadinessSummary
		items  []models.DeliveryReadinessItem
	)
	err := concurrency.Run(ctx, []concurrency.Task{
		func(c context.Context) error {
			result, err := s.repo.GetDeliveryReadinessTotals(c, filter)
			if err != nil {
				return err
			}
			totals = result
			return nil
		},
		func(c context.Context) error {
			result, err := s.repo.GetDeliveryReadinessItems(c, filter)
			if err != nil {
				return err
			}
			items = result
			return nil
		},
	}, s.fanout)
	if err != nil {
		return nil, err
	}
	totals.Items = items
	return totals, nil
}

func (s *service) GetProductionIssuesSummary(ctx context.Context, limit, windowHours int) (*models.ProductionIssuesSummary, error) {
	return s.repo.GetProductionIssuesSummary(ctx, repository.Filter{Limit: limit, WindowHours: windowHours})
}

func (s *service) GetScanEventsSummary(ctx context.Context, limit, windowHours int) (*models.ScanEventsSummary, error) {
	return s.repo.GetScanEventsSummary(ctx, repository.Filter{Limit: limit, WindowHours: windowHours})
}
