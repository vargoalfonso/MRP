package service

import (
	"context"
	"strings"

	"github.com/ganasa18/go-template/internal/admin_jobs/repository"
)

type IService interface {
	// RebuildPRLPeriodSummaries recomputes prl_item_period_summaries for the given period.
	// If forecastPeriod is empty, the latest globally approved period is used.
	// Returns the number of rows upserted.
	RebuildPRLPeriodSummaries(ctx context.Context, forecastPeriod string) (int64, string, error)
}

type service struct {
	repo repository.IRepository
}

func New(repo repository.IRepository) IService { return &service{repo: repo} }

func (s *service) RebuildPRLPeriodSummaries(ctx context.Context, forecastPeriod string) (int64, string, error) {
	forecastPeriod = strings.TrimSpace(forecastPeriod)
	if forecastPeriod == "" {
		latest, err := s.repo.GetLatestApprovedPRLPeriod(ctx)
		if err != nil {
			return 0, "", err
		}
		forecastPeriod = latest
	}

	if forecastPeriod == "" {
		return 0, "", nil
	}

	n, err := s.repo.RebuildPRLPeriodSummaries(ctx, forecastPeriod)
	return n, forecastPeriod, err
}
