package repository

import (
	"context"
	"strings"

	"github.com/ganasa18/go-template/internal/work_order_logs/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

type IRepository interface {
	Create(ctx context.Context, log *models.WorkOrderAutomationLog) error
	List(ctx context.Context, f models.ListLogFilters) ([]models.WorkOrderAutomationLog, int64, error)
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, log *models.WorkOrderAutomationLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return apperror.InternalWrap("create work order automation log failed", err)
	}
	return nil
}

func (r *repository) List(ctx context.Context, f models.ListLogFilters) ([]models.WorkOrderAutomationLog, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.WorkOrderAutomationLog{})

	if v := strings.TrimSpace(f.WoNumber); v != "" {
		q = q.Where("wo_number = ?", v)
	}
	if v := strings.TrimSpace(f.AutomationJobID); v != "" {
		q = q.Where("automation_job_id = ?", v)
	}
	if v := strings.TrimSpace(f.Search); v != "" {
		like := "%" + v + "%"
		q = q.Where("wo_number ILIKE ? OR item_uniq_code ILIKE ? OR part_name ILIKE ? OR failure_reason ILIKE ?", like, like, like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, apperror.InternalWrap("count work order automation logs failed", err)
	}

	var logs []models.WorkOrderAutomationLog
	err := q.Order("created_at DESC, id DESC").Limit(f.Limit).Offset(f.Offset).Find(&logs).Error
	if err != nil {
		return nil, 0, apperror.InternalWrap("list work order automation logs failed", err)
	}
	return logs, total, nil
}
