package repository

import (
	"context"

	"github.com/ganasa18/go-template/internal/kanban/models"
	"gorm.io/gorm"
)

type IKanbanParameterRepository interface {
	Create(ctx context.Context, data *models.KanbanParameter) error
	FindAll(ctx context.Context) ([]models.KanbanParameter, error)
	FindByID(ctx context.Context, id int64) (*models.KanbanParameter, error)
	FindByItemCode(ctx context.Context, code string) (*models.KanbanParameter, error)
	Update(ctx context.Context, data *models.KanbanParameter) error
	Delete(ctx context.Context, id int64) error
	Count(ctx context.Context) (int64, error)
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IKanbanParameterRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, data *models.KanbanParameter) error {
	return r.db.WithContext(ctx).Create(data).Error
}

func (r *repository) FindAll(ctx context.Context) ([]models.KanbanParameter, error) {
	var result []models.KanbanParameter
	err := r.db.WithContext(ctx).Find(&result).Error
	return result, err
}

func (r *repository) FindByID(ctx context.Context, id int64) (*models.KanbanParameter, error) {
	var data models.KanbanParameter
	err := r.db.WithContext(ctx).First(&data, id).Error
	return &data, err
}

func (r *repository) FindByItemCode(ctx context.Context, code string) (*models.KanbanParameter, error) {
	var data models.KanbanParameter
	err := r.db.WithContext(ctx).
		Where("item_uniq_code = ?", code).
		First(&data).Error
	return &data, err
}

func (r *repository) Update(ctx context.Context, data *models.KanbanParameter) error {
	return r.db.WithContext(ctx).Save(data).Error
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.KanbanParameter{}, id).Error
}

func (r *repository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.KanbanParameter{}).Count(&count).Error
	return count, err
}
