// Package repository provides data-access for the Machine Pattern module.
package repository

import (
	"context"

	"github.com/ganasa18/go-template/internal/machinepattern/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

type IRepository interface {
	// MachinePattern CRUD
	Create(ctx context.Context, mp *models.MachinePattern) error
	Update(ctx context.Context, mp *models.MachinePattern) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*models.MachinePattern, error)
	GetByUniqAndMachine(ctx context.Context, uniqCode string, machineID int64) (*models.MachinePattern, error)
	List(ctx context.Context, q models.ListQuery) ([]models.MachinePattern, int64, error)

	// Global params (singleton, id=1)
	GetParams(ctx context.Context) (*models.MachinePatternParam, error)
	UpsertParams(ctx context.Context, p *models.MachinePatternParam) error

	// Aggregates
	Summary(ctx context.Context) (*models.SummaryResponse, error)

	// Cross-module reads
	GetMachineNamesByIDs(ctx context.Context, ids []int64) (map[int64]string, error)
}

type repository struct{ db *gorm.DB }

func New(db *gorm.DB) IRepository { return &repository{db: db} }

// ---------------------------------------------------------------------------
// MachinePattern writes
// ---------------------------------------------------------------------------

func (r *repository) Create(ctx context.Context, mp *models.MachinePattern) error {
	if err := r.db.WithContext(ctx).Create(mp).Error; err != nil {
		return apperror.InternalWrap("Create MachinePattern", err)
	}
	return nil
}

func (r *repository) Update(ctx context.Context, mp *models.MachinePattern) error {
	if err := r.db.WithContext(ctx).Save(mp).Error; err != nil {
		return apperror.InternalWrap("Update MachinePattern", err)
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	if err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.MachinePattern{}).Error; err != nil {
		return apperror.InternalWrap("Delete MachinePattern", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// MachinePattern reads
// ---------------------------------------------------------------------------

func (r *repository) GetByID(ctx context.Context, id int64) (*models.MachinePattern, error) {
	var mp models.MachinePattern
	err := r.db.WithContext(ctx).First(&mp, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.NotFound("machine pattern not found")
	}
	if err != nil {
		return nil, apperror.InternalWrap("GetByID MachinePattern", err)
	}
	return &mp, nil
}

func (r *repository) GetByUniqAndMachine(ctx context.Context, uniqCode string, machineID int64) (*models.MachinePattern, error) {
	var mp models.MachinePattern
	err := r.db.WithContext(ctx).
		Where("uniq_code = ? AND machine_id = ?", uniqCode, machineID).
		First(&mp).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil // not found is ok (used for duplicate check)
	}
	if err != nil {
		return nil, apperror.InternalWrap("GetByUniqAndMachine", err)
	}
	return &mp, nil
}

func (r *repository) List(ctx context.Context, q models.ListQuery) ([]models.MachinePattern, int64, error) {
	db := r.db.WithContext(ctx).Model(&models.MachinePattern{})

	if q.UniqCode != "" {
		db = db.Where("uniq_code ILIKE ?", "%"+q.UniqCode+"%")
	}
	if q.MachineID > 0 {
		db = db.Where("machine_id = ?", q.MachineID)
	}
	if q.MovingType != "" {
		db = db.Where("moving_type = ?", q.MovingType)
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	if q.Search != "" {
		db = db.Where("uniq_code ILIKE ?", "%"+q.Search+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, apperror.InternalWrap("List MachinePattern count", err)
	}

	limit, offset := limitOffset(q.Limit, q.Page)
	var items []models.MachinePattern
	if err := db.Order("id ASC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, apperror.InternalWrap("List MachinePattern", err)
	}
	return items, total, nil
}

// ---------------------------------------------------------------------------
// Global params
// ---------------------------------------------------------------------------

func (r *repository) GetParams(ctx context.Context) (*models.MachinePatternParam, error) {
	var p models.MachinePatternParam
	err := r.db.WithContext(ctx).First(&p).Error
	if err == gorm.ErrRecordNotFound {
		// Return safe defaults when no row exists yet
		return &models.MachinePatternParam{
			FastMovingThreshold: 1000,
			SlowMovingThreshold: 1000,
			PatternMinMinutes:   48,
			DefaultWorkingDays:  25,
		}, nil
	}
	if err != nil {
		return nil, apperror.InternalWrap("GetParams", err)
	}
	return &p, nil
}

func (r *repository) UpsertParams(ctx context.Context, p *models.MachinePatternParam) error {
	if p.ID == 0 {
		p.ID = 1 // singleton
	}
	result := r.db.WithContext(ctx).
		Where(models.MachinePatternParam{ID: 1}).
		Assign(*p).
		FirstOrCreate(p)
	if result.Error != nil {
		return apperror.InternalWrap("UpsertParams", result.Error)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Aggregates
// ---------------------------------------------------------------------------

// Summary aggregates dashboard counts across all machine patterns.
func (r *repository) Summary(ctx context.Context) (*models.SummaryResponse, error) {
	out := &models.SummaryResponse{}

	if err := r.db.WithContext(ctx).Model(&models.MachinePattern{}).
		Count(&out.TotalPattern).Error; err != nil {
		return nil, apperror.InternalWrap("Summary total", err)
	}
	if err := r.db.WithContext(ctx).Model(&models.MachinePattern{}).
		Where("moving_type = ?", "Fast Moving").Count(&out.FastMoving).Error; err != nil {
		return nil, apperror.InternalWrap("Summary fast", err)
	}
	if err := r.db.WithContext(ctx).Model(&models.MachinePattern{}).
		Where("moving_type = ?", "Slow Moving").Count(&out.SlowMoving).Error; err != nil {
		return nil, apperror.InternalWrap("Summary slow", err)
	}
	if err := r.db.WithContext(ctx).Model(&models.MachinePattern{}).
		Where("moving_type = ?", "Normal").Count(&out.Normal).Error; err != nil {
		return nil, apperror.InternalWrap("Summary normal", err)
	}

	var avg struct{ Avg float64 }
	if err := r.db.WithContext(ctx).Model(&models.MachinePattern{}).
		Select("COALESCE(AVG(pattern_value), 0) AS avg").Scan(&avg).Error; err != nil {
		return nil, apperror.InternalWrap("Summary avg", err)
	}
	out.AvgPattern = avg.Avg

	return out, nil
}

// ---------------------------------------------------------------------------
// Cross-module reads
// ---------------------------------------------------------------------------

// GetMachineNamesByIDs reads from master_machines (owned by billmaterial module).
func (r *repository) GetMachineNamesByIDs(ctx context.Context, ids []int64) (map[int64]string, error) {
	result := make(map[int64]string)
	if len(ids) == 0 {
		return result, nil
	}
	type row struct {
		ID          int64
		MachineName string
	}
	var rows []row
	if err := r.db.WithContext(ctx).
		Table("master_machines").
		Select("id, machine_name").
		Where("id IN ?", ids).
		Find(&rows).Error; err != nil {
		return nil, apperror.InternalWrap("GetMachineNamesByIDs", err)
	}
	for _, row := range rows {
		result[row.ID] = row.MachineName
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func limitOffset(limit, page int) (int, int) {
	if limit < 1 || limit > 200 {
		limit = 20
	}
	if page < 1 {
		page = 1
	}
	return limit, (page - 1) * limit
}
