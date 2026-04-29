package repository

import (
	"context"
	"strings"

	"github.com/ganasa18/go-template/internal/machine_pattern/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

type IRepository interface {
	List(ctx context.Context, q ListQuery) ([]models.MachinePattern, int64, error)
	GetByID(ctx context.Context, id int64) (*models.MachinePattern, error)
	GetByUniqCode(ctx context.Context, uniqCode string) ([]models.MachinePattern, error)
	GetByMachineID(ctx context.Context, machineID int64) ([]models.MachinePattern, error)
	GetByUniqAndMachine(ctx context.Context, uniqCode string, machineID int64) (*models.MachinePattern, error)
	Create(ctx context.Context, p *models.MachinePattern) error
	Update(ctx context.Context, p *models.MachinePattern) error
	Delete(ctx context.Context, id int64) error
	ValidateMachineExists(ctx context.Context, machineID int64) (bool, error)
	GetSummary(ctx context.Context) (*models.MachinePatternSummary, error)
}

type repository struct{ db *gorm.DB }

func New(db *gorm.DB) IRepository { return &repository{db: db} }

type ListQuery struct {
	Page       int
	Limit      int
	Offset     int
	Search     string
	MachineID  int64
	MovingType string
	Status     string
	UniqCode   string
}

func (r *repository) List(ctx context.Context, q ListQuery) ([]models.MachinePattern, int64, error) {
	var items []models.MachinePattern
	var total int64

	base := r.db.WithContext(ctx).Model(&models.MachinePattern{}).Where("deleted_at IS NULL")

	if search := strings.TrimSpace(q.Search); search != "" {
		base = base.Where("uniq_code ILIKE ?", "%"+search+"%")
	}
	if q.MachineID > 0 {
		base = base.Where("machine_id = ?", q.MachineID)
	}
	if movingType := strings.TrimSpace(q.MovingType); movingType != "" {
		base = base.Where("moving_type = ?", movingType)
	}
	if status := strings.TrimSpace(q.Status); status != "" {
		base = base.Where("status = ?", status)
	}
	if uniqCode := strings.TrimSpace(q.UniqCode); uniqCode != "" {
		base = base.Where("uniq_code = ?", uniqCode)
	}

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, apperror.InternalWrap("list machine patterns count", err)
	}

	if err := base.Order("id ASC").Offset(q.Offset).Limit(q.Limit).Find(&items).Error; err != nil {
		return nil, 0, apperror.InternalWrap("list machine patterns", err)
	}

	return items, total, nil
}

func (r *repository) GetByID(ctx context.Context, id int64) (*models.MachinePattern, error) {
	var p models.MachinePattern
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		First(&p, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("machine pattern not found")
		}
		return nil, apperror.InternalWrap("get machine pattern by id", err)
	}
	return &p, nil
}

func (r *repository) GetByUniqCode(ctx context.Context, uniqCode string) ([]models.MachinePattern, error) {
	var items []models.MachinePattern
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL AND uniq_code = ?", uniqCode).
		Order("machine_id ASC").
		Find(&items).Error
	if err != nil {
		return nil, apperror.InternalWrap("get machine patterns by uniq code", err)
	}
	return items, nil
}

func (r *repository) GetByMachineID(ctx context.Context, machineID int64) ([]models.MachinePattern, error) {
	var items []models.MachinePattern
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL AND machine_id = ?", machineID).
		Order("uniq_code ASC").
		Find(&items).Error
	if err != nil {
		return nil, apperror.InternalWrap("get machine patterns by machine id", err)
	}
	return items, nil
}

func (r *repository) GetByUniqAndMachine(ctx context.Context, uniqCode string, machineID int64) (*models.MachinePattern, error) {
	var p models.MachinePattern
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL AND uniq_code = ? AND machine_id = ?", uniqCode, machineID).
		First(&p).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, apperror.InternalWrap("get machine pattern by uniq and machine", err)
	}
	return &p, nil
}

func (r *repository) Create(ctx context.Context, p *models.MachinePattern) error {
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return apperror.InternalWrap("create machine pattern", err)
	}
	return nil
}

func (r *repository) Update(ctx context.Context, p *models.MachinePattern) error {
	if err := r.db.WithContext(ctx).Save(p).Error; err != nil {
		return apperror.InternalWrap("update machine pattern", err)
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	if err := r.db.WithContext(ctx).
		Model(&models.MachinePattern{}).
		Where("id = ?", id).
		Update("deleted_at", gorm.Expr("NOW()")).Error; err != nil {
		return apperror.InternalWrap("delete machine pattern", err)
	}
	return nil
}

func (r *repository) ValidateMachineExists(ctx context.Context, machineID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.MachinePattern{}).
		Raw("SELECT COUNT(*) FROM master_machines WHERE id = ?", machineID).
		Scan(&count).Error
	if err != nil {
		return false, apperror.InternalWrap("validate machine exists", err)
	}
	return count > 0, nil
}

func (r *repository) GetSummary(ctx context.Context) (*models.MachinePatternSummary, error) {
	var result struct {
		TotalPattern int
		FastMoving   int
		SlowMoving   int
		Normal       int
		AvgPattern   float64
	}

	err := r.db.WithContext(ctx).
		Model(&models.MachinePattern{}).
		Where("deleted_at IS NULL").
		Select(`
			COUNT(*) as total_pattern,
			COUNT(CASE WHEN moving_type = 'Fast Moving' THEN 1 END) as fast_moving,
			COUNT(CASE WHEN moving_type = 'Slow Moving' THEN 1 END) as slow_moving,
			COUNT(CASE WHEN moving_type = 'Normal' THEN 1 END) as normal,
			COALESCE(AVG(pattern_value), 0) as avg_pattern
		`).Scan(&result).Error
	if err != nil {
		return nil, apperror.InternalWrap("get machine pattern summary", err)
	}

	return &models.MachinePatternSummary{
		TotalPattern: result.TotalPattern,
		FastMoving:   result.FastMoving,
		SlowMoving:   result.SlowMoving,
		Normal:       result.Normal,
		AvgPattern:   result.AvgPattern,
	}, nil
}
