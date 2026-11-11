package repository

import (
	"context"
	"strings"

	"github.com/ganasa18/go-template/internal/type_parameter/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

func wrapTypeParameterDBError(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "description") && strings.Contains(msg, "does not exist") {
		return apperror.BadRequest("kolom description pada type_parameters belum ada; jalankan migration 0067_add_description_to_type_parameters_up.sql")
	}
	if strings.Contains(msg, "unique_type_code") || strings.Contains(msg, "duplicate key value") {
		return apperror.Conflict("type_code sudah digunakan")
	}
	return apperror.InternalWrap("type parameter db error", err)
}

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
	return wrapTypeParameterDBError(r.db.WithContext(ctx).Create(data).Error)
}

func (r *repository) FindAll(ctx context.Context) ([]models.TypeParameter, error) {
	var res []models.TypeParameter
	err := r.db.WithContext(ctx).Find(&res).Error
	return res, wrapTypeParameterDBError(err)
}

func (r *repository) FindByID(ctx context.Context, id int64) (*models.TypeParameter, error) {
	var data models.TypeParameter
	err := r.db.WithContext(ctx).First(&data, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.NotFound("type parameter tidak ditemukan")
	}
	return &data, wrapTypeParameterDBError(err)
}

func (r *repository) Update(ctx context.Context, id int64, data map[string]interface{}) error {
	return wrapTypeParameterDBError(r.db.WithContext(ctx).
		Model(&models.TypeParameter{}).
		Where("id = ?", id).
		Updates(data).Error)
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	return wrapTypeParameterDBError(r.db.WithContext(ctx).Delete(&models.TypeParameter{}, id).Error)
}
