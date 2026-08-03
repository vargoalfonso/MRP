package repository

import (
	"context"
	"strings"

	"github.com/ganasa18/go-template/internal/customer_order_logs/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

type IRepository interface {
	Create(ctx context.Context, log *models.CustomerOrderAutomationLog) error
	List(ctx context.Context, f models.ListLogFilters) ([]models.CustomerOrderAutomationLog, int64, error)
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, log *models.CustomerOrderAutomationLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return apperror.InternalWrap("create customer order automation log failed", err)
	}
	return nil
}

func (r *repository) List(ctx context.Context, f models.ListLogFilters) ([]models.CustomerOrderAutomationLog, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.CustomerOrderAutomationLog{})

	if f.DocumentNumber != "" {
		q = q.Where("document_number = ?", strings.TrimSpace(f.DocumentNumber))
	}
	if f.Search != "" {
		like := "%" + strings.TrimSpace(f.Search) + "%"
		q = q.Where("item_uniq_code ILIKE ? OR part_name ILIKE ? OR failure_reason ILIKE ?", like, like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, apperror.InternalWrap("count customer order automation logs failed", err)
	}

	var logs []models.CustomerOrderAutomationLog
	err := q.Order("row_no ASC, created_at DESC").Limit(f.Limit).Offset(f.Offset).Find(&logs).Error
	if err != nil {
		return nil, 0, apperror.InternalWrap("list customer order automation logs failed", err)
	}
	return logs, total, nil
}
