package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ganasa18/go-template/internal/stockdaysparameter/models"
	"gorm.io/gorm"
)

type IStockdaysRepository interface {
	Create(ctx context.Context, data *models.StockdaysParameter) error
	BulkCreate(ctx context.Context, data []models.StockdaysParameter) error
	FindAll(ctx context.Context) ([]models.StockdaysParameter, error)
	FindByID(ctx context.Context, id int64) (*models.StockdaysParameter, error)
	Update(ctx context.Context, id int64, data map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IStockdaysRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, data *models.StockdaysParameter) error {
	return r.db.WithContext(ctx).Create(data).Error
}

func (r *repository) BulkCreate(ctx context.Context, data []models.StockdaysParameter) error {
	return r.db.WithContext(ctx).Create(&data).Error
}

func (r *repository) FindAll(ctx context.Context) ([]models.StockdaysParameter, error) {
	var result []models.StockdaysParameter
	err := r.db.WithContext(ctx).Find(&result).Error
	return result, err
}

func (r *repository) FindByID(ctx context.Context, id int64) (*models.StockdaysParameter, error) {
	var data models.StockdaysParameter
	err := r.db.WithContext(ctx).First(&data, id).Error
	return &data, err
}

func (r *repository) Update(ctx context.Context, id int64, data map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&models.StockdaysParameter{}).
		Where("id = ?", id).
		Updates(data).Error
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.StockdaysParameter{}, id).Error
}

func (r *repository) GetSafetyStockByItem(ctx context.Context, itemUniqCode string) (*models.SafetyStockParameter, error) {
	var data models.SafetyStockParameter

	err := r.db.WithContext(ctx).
		Where("item_uniq_code = ?", itemUniqCode).
		First(&data).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("safety stock parameter with item_uniq_code %s not found", itemUniqCode)
		}
		return nil, err
	}

	return &data, nil
}
