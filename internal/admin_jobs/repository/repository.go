package repository

import (
	"context"

	"gorm.io/gorm"
)

type IRepository interface {
	// GetLatestApprovedPRLPeriod returns the most recently approved forecast_period across all items.
	GetLatestApprovedPRLPeriod(ctx context.Context) (string, error)
	// RebuildPRLPeriodSummaries upserts prl_item_period_summaries for the given forecast_period.
	RebuildPRLPeriodSummaries(ctx context.Context, forecastPeriod string) (int64, error)
}

type repo struct{ db *gorm.DB }

func New(db *gorm.DB) IRepository { return &repo{db: db} }

func (r *repo) GetLatestApprovedPRLPeriod(ctx context.Context) (string, error) {
	var result struct {
		ForecastPeriod string `gorm:"column:forecast_period"`
	}
	err := r.db.WithContext(ctx).
		Table("prls").
		Select("forecast_period").
		Where("status = 'approved' AND deleted_at IS NULL").
		Order("COALESCE(approved_at, updated_at) DESC").
		Limit(1).
		Scan(&result).Error
	return result.ForecastPeriod, err
}

func (r *repo) RebuildPRLPeriodSummaries(ctx context.Context, forecastPeriod string) (int64, error) {
	result := r.db.WithContext(ctx).Exec(`
		INSERT INTO prl_item_period_summaries (forecast_period, item_uniq_code, prl_total_qty, computed_at)
		SELECT
			p.forecast_period,
			p.uniq_code        AS item_uniq_code,
			SUM(p.quantity)    AS prl_total_qty,
			NOW()              AS computed_at
		FROM prls p
		WHERE p.status      = 'approved'
		  AND p.deleted_at  IS NULL
		  AND p.forecast_period = ?
		GROUP BY p.forecast_period, p.uniq_code
		ON CONFLICT (forecast_period, item_uniq_code)
		DO UPDATE SET
			prl_total_qty = EXCLUDED.prl_total_qty,
			computed_at   = EXCLUDED.computed_at
	`, forecastPeriod)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
