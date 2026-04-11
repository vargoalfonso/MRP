package repository

import (
	"context"

	"github.com/ganasa18/go-template/internal/type_parameter/models"
	"gorm.io/gorm"
)

type ITypeRepository interface {
	Create(ctx context.Context, data *models.TypeParameter) error
	FindAll(ctx context.Context) ([]models.TypeParameter, error)
	FindByID(ctx context.Context, id int64) (*models.TypeParameter, error)
	Update(ctx context.Context, id int64, data map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) ITypeRepository {
	return &repository{db: db}
}
func (r *repository) Create(ctx context.Context, data *models.TypeParameter) error {
	return r.db.WithContext(ctx).Create(data).Error
}

func (r *repository) FindAll(ctx context.Context) ([]models.TypeParameter, error) {
	var res []models.TypeParameter
	err := r.db.WithContext(ctx).Find(&res).Error
	return res, err
}

func (r *repository) FindByID(ctx context.Context, id int64) (*models.TypeParameter, error) {
	var data models.TypeParameter
	err := r.db.WithContext(ctx).First(&data, id).Error
	return &data, err
}

func (r *repository) Update(ctx context.Context, id int64, data map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&models.TypeParameter{}).
		Where("id = ?", id).
		Updates(data).Error
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.TypeParameter{}, id).Error
}
