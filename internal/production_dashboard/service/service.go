package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/production_dashboard/models"
	"github.com/ganasa18/go-template/internal/production_dashboard/repository"
)

// IService builds the { summary_cards, table_data } payload for each dashboard view.
type IService interface {
	FinishedGoods(ctx context.Context) (*models.DashboardPayload, error)
	Wip(ctx context.Context) (*models.DashboardPayload, error)
	OutputPerMachine(ctx context.Context) (*models.DashboardPayload, error)
	SummaryStroke(ctx context.Context) (*models.DashboardPayload, error)
	Runtime(ctx context.Context) (*models.DashboardPayload, error)
}

type service struct{ repo repository.IRepository }

// New builds the Production Dashboard service.
func New(repo repository.IRepository) IService { return &service{repo: repo} }

func (s *service) FinishedGoods(ctx context.Context) (*models.DashboardPayload, error) {
	cards, err := s.repo.GetSummaryCards(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListFinishedGoods(ctx)
	if err != nil {
		return nil, err
	}
	return &models.DashboardPayload{SummaryCards: cards, TableData: rows}, nil
}

func (s *service) Wip(ctx context.Context) (*models.DashboardPayload, error) {
	cards, err := s.repo.GetSummaryCards(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListWip(ctx)
	if err != nil {
		return nil, err
	}
	return &models.DashboardPayload{SummaryCards: cards, TableData: rows}, nil
}

func (s *service) OutputPerMachine(ctx context.Context) (*models.DashboardPayload, error) {
	cards, err := s.repo.GetSummaryCards(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListOutputPerMachine(ctx)
	if err != nil {
		return nil, err
	}
	return &models.DashboardPayload{SummaryCards: cards, TableData: rows}, nil
}

func (s *service) SummaryStroke(ctx context.Context) (*models.DashboardPayload, error) {
	cards, err := s.repo.GetSummaryCards(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListSummaryStroke(ctx)
	if err != nil {
		return nil, err
	}
	return &models.DashboardPayload{SummaryCards: cards, TableData: rows}, nil
}

func (s *service) Runtime(ctx context.Context) (*models.DashboardPayload, error) {
	cards, err := s.repo.GetSummaryCards(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListRuntime(ctx)
	if err != nil {
		return nil, err
	}
	return &models.DashboardPayload{SummaryCards: cards, TableData: rows}, nil
}
