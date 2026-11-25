package repository

import (
	"context"

	"github.com/ganasa18/go-template/internal/unit_measurement/models"
	"gorm.io/gorm"
)

type IUnitMeasurementRepository interface {
	Create(ctx context.Context, data *models.UomParameter) error
	FindAll(ctx context.Context) ([]models.UomParameter, error)
	FindByID(ctx context.Context, id int64) (*models.UomParameter, error)
	Update(ctx context.Context, id int64, data map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IUnitMeasurementRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, data *models.UomParameter) error {
	return r.db.WithContext(ctx).Create(data).Error
}

func (r *repository) FindAll(ctx context.Context) ([]models.UomParameter, error) {
	var res []models.UomParameter
	err := r.db.WithContext(ctx).Find(&res).Error
	return res, err
}

func (r *repository) FindByID(ctx context.Context, id int64) (*models.UomParameter, error) {
	var data models.UomParameter
	err := r.db.WithContext(ctx).First(&data, id).Error
	return &data, err
}

func (r *repository) Update(ctx context.Context, id int64, data map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&models.UomParameter{}).
		Where("id = ?", id).
		Updates(data).Error
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.UomParameter{}, id).Error
}
