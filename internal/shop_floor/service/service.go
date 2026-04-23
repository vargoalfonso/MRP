package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/shop_floor/models"
	"github.com/ganasa18/go-template/internal/shop_floor/repository"
)

type IService interface {
	GetLiveProductionSummary(ctx context.Context, limit, staleMinutes int) (*models.LiveProductionSummary, error)
	GetDeliveryReadinessSummary(ctx context.Context, limit int) (*models.DeliveryReadinessSummary, error)
	GetProductionIssuesSummary(ctx context.Context, limit, windowHours int) (*models.ProductionIssuesSummary, error)
	GetScanEventsSummary(ctx context.Context, limit, windowHours int) (*models.ScanEventsSummary, error)
}

type service struct {
	repo repository.IRepository
}

func New(repo repository.IRepository) IService {
	return &service{repo: repo}
}

func (s *service) GetLiveProductionSummary(ctx context.Context, limit, staleMinutes int) (*models.LiveProductionSummary, error) {
	return s.repo.GetLiveProductionSummary(ctx, repository.Filter{Limit: limit, StaleMinutes: staleMinutes})
}

func (s *service) GetDeliveryReadinessSummary(ctx context.Context, limit int) (*models.DeliveryReadinessSummary, error) {
	return s.repo.GetDeliveryReadinessSummary(ctx, repository.Filter{Limit: limit})
}

func (s *service) GetProductionIssuesSummary(ctx context.Context, limit, windowHours int) (*models.ProductionIssuesSummary, error) {
	return s.repo.GetProductionIssuesSummary(ctx, repository.Filter{Limit: limit, WindowHours: windowHours})
}

func (s *service) GetScanEventsSummary(ctx context.Context, limit, windowHours int) (*models.ScanEventsSummary, error) {
	return s.repo.GetScanEventsSummary(ctx, repository.Filter{Limit: limit, WindowHours: windowHours})
}
