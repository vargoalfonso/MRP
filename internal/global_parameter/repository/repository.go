package repository

import (
	"context"
	"errors"

	"github.com/ganasa18/go-template/internal/global_parameter/models"
	"gorm.io/gorm"
)

type IGlobalParameterRepository interface {
	FindAll(ctx context.Context) ([]models.GlobalParameter, error)
	FindByID(ctx context.Context, id int64) (*models.GlobalParameter, error)
	Create(ctx context.Context, req models.CreateGlobalParameterRequest) (*models.GlobalParameter, error)
	Update(ctx context.Context, id int64, req models.UpdateGlobalParameterRequest) (*models.GlobalParameter, error)
	Delete(ctx context.Context, id int64) error
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IGlobalParameterRepository {
	return &repository{db: db}
}

func (r *repository) FindAll(ctx context.Context) ([]models.GlobalParameter, error) {
	var data []models.GlobalParameter

	err := r.db.WithContext(ctx).
		Order("id DESC").
		Find(&data).Error

	if err != nil {
		return nil, err
	}

	return data, nil
}

func (r *repository) FindByID(ctx context.Context, id int64) (*models.GlobalParameter, error) {
	var data models.GlobalParameter

	err := r.db.WithContext(ctx).
		First(&data, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("global parameter not found")
		}
		return nil, err
	}

	return &data, nil
}

func (r *repository) Create(ctx context.Context, req models.CreateGlobalParameterRequest) (*models.GlobalParameter, error) {
	data := models.GlobalParameter{
		ParameterGroup: req.ParameterGroup,
		Period:         req.Period,
		WorkingDays:    req.WorkingDays,
		Status:         req.Status,
	}

	if err := r.db.WithContext(ctx).
		Create(&data).Error; err != nil {
		return nil, err
	}

	return &data, nil
}

func (r *repository) Update(ctx context.Context, id int64, req models.UpdateGlobalParameterRequest) (*models.GlobalParameter, error) {
	var data models.GlobalParameter

	// cek data
	if err := r.db.WithContext(ctx).
		First(&data, id).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("global parameter not found")
		}
		return nil, err
	}

	// mapping update
	updateData := map[string]interface{}{
		"parameter_group": req.ParameterGroup,
		"period":          req.Period,
		"working_days":    req.WorkingDays,
		"status":          req.Status,
	}

	if err := r.db.WithContext(ctx).
		Model(&data).
		Updates(updateData).Error; err != nil {
		return nil, err
	}

	// ambil data terbaru
	if err := r.db.WithContext(ctx).
		First(&data, id).Error; err != nil {
		return nil, err
	}

	return &data, nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).
		Delete(&models.GlobalParameter{}, id)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("global parameter not found")
	}

	return nil
}
