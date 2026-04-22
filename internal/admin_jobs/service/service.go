package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/admin_jobs/repository"
)

type IService interface {
	// RebuildDemandPeriodeSummaries reads the active_periode from global parameters,
	// resolves working days (with fallback), and upserts inventory_demand_periode_summaries
	// for all raw_material items for today's snapshot date.
	// Returns (rowsUpserted, activePeriode, error).
	RebuildDemandPeriodeSummaries(ctx context.Context) (int64, string, error)
}

type service struct {
	repo repository.IRepository
}

func New(repo repository.IRepository) IService { return &service{repo: repo} }

func (s *service) RebuildDemandPeriodeSummaries(ctx context.Context) (int64, string, error) {
	activePeriode, err := s.repo.GetGlobalActivePeriode(ctx)
	if err != nil {
		return 0, "", err
	}
	if activePeriode == "" {
		return 0, "", nil
	}

	workingDays, workingDaysPeriodeUsed, err := s.repo.GetWorkingDaysWithFallback(ctx, activePeriode)
	if err != nil {
		return 0, activePeriode, err
	}

	n, err := s.repo.RebuildDemandPeriodeSummaries(ctx, activePeriode, workingDays, workingDaysPeriodeUsed)
	return n, activePeriode, err
}
