package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/ganasa18/go-template/internal/mechin_parameter/models"
	"gorm.io/gorm"
)

type ListQuery struct {
	Limit  int
	Offset int
	Search string
	Status string
}

type IMechinParameterRepository interface {
	FindAll(ctx context.Context, q ListQuery) ([]models.MechinParameter, int64, error)
	FindByID(ctx context.Context, id int64) (*models.MechinParameter, error)
	Create(ctx context.Context, req models.CreateMechinParameterRequest) (*models.MechinParameter, error)
	Update(ctx context.Context, id int64, req models.UpdateMechinParameterRequest) (*models.MechinParameter, error)
	Delete(ctx context.Context, id int64) error
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IMechinParameterRepository {
	return &repository{db: db}
}

func (r *repository) FindAll(ctx context.Context, q ListQuery) ([]models.MechinParameter, int64, error) {
	var data []models.MechinParameter
	var total int64

	query := r.db.WithContext(ctx).Model(&models.MechinParameter{})

	if strings.TrimSpace(q.Search) != "" {
		query = query.Where("machine_name ILIKE ?", "%"+strings.TrimSpace(q.Search)+"%")
	}
	if strings.TrimSpace(q.Status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(q.Status))
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("id DESC").Limit(q.Limit).Offset(q.Offset).Find(&data).Error; err != nil {
		return nil, 0, err
	}

	return data, total, nil
}

func (r *repository) FindByID(ctx context.Context, id int64) (*models.MechinParameter, error) {
	var data models.MechinParameter

	err := r.db.WithContext(ctx).First(&data, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("mechin parameter not found")
		}
		return nil, err
	}

	return &data, nil
}

func (r *repository) Create(ctx context.Context, req models.CreateMechinParameterRequest) (*models.MechinParameter, error) {
	data := models.MechinParameter{
		MachineName:    strings.TrimSpace(req.MachineName),
		MachineCount:   req.MachineCount,
		OperatingHours: req.OperatingHours,
		Status:         strings.TrimSpace(req.Status),
	}

	if err := r.db.WithContext(ctx).Create(&data).Error; err != nil {
		return nil, err
	}

	return &data, nil
}

func (r *repository) Update(ctx context.Context, id int64, req models.UpdateMechinParameterRequest) (*models.MechinParameter, error) {
	var data models.MechinParameter
	if err := r.db.WithContext(ctx).First(&data, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("mechin parameter not found")
		}
		return nil, err
	}

	updateData := map[string]interface{}{}
	if strings.TrimSpace(req.MachineName) != "" {
		updateData["machine_name"] = strings.TrimSpace(req.MachineName)
	}
	if req.MachineCount != nil {
		updateData["machine_count"] = *req.MachineCount
	}
	if req.OperatingHours != nil {
		updateData["operating_hours"] = *req.OperatingHours
	}
	if strings.TrimSpace(req.Status) != "" {
		updateData["status"] = strings.TrimSpace(req.Status)
	}

	if len(updateData) > 0 {
		if err := r.db.WithContext(ctx).Model(&data).Updates(updateData).Error; err != nil {
			return nil, err
		}
	}

	if err := r.db.WithContext(ctx).First(&data, id).Error; err != nil {
		return nil, err
	}

	return &data, nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&models.MechinParameter{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("mechin parameter not found")
	}
	return nil
}
