package repository

import (
	"context"

	"github.com/ganasa18/go-template/internal/safety_stock_parameter/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ISafetyStockRepository interface {
	Create(ctx context.Context, data *models.SafetyStockParameter) error
	BulkCreate(ctx context.Context, data []models.SafetyStockParameter) error
	FindAll(ctx context.Context) ([]models.SafetyStockParameter, error)
	FindByID(ctx context.Context, id int64) (*models.SafetyStockParameter, error)
	FindByItemCode(ctx context.Context, itemCode string) (*models.SafetyStockParameter, error)
	Update(ctx context.Context, id int64, data map[string]interface{}) error
	Delete(ctx context.Context, id int64) error

	GetForecastByItem(ctx context.Context, itemCode string) (float64, error)
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) ISafetyStockRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, data *models.SafetyStockParameter) error {
	return r.db.WithContext(ctx).Create(data).Error
}

func (r *repository) BulkCreate(ctx context.Context, data []models.SafetyStockParameter) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "inventory_type"},
				{Name: "item_uniq_code"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"calculation_type",
				"constanta",
				"updated_at",
			}),
		}).
		Create(&data).Error
}

func (r *repository) FindAll(ctx context.Context) ([]models.SafetyStockParameter, error) {
	var result []models.SafetyStockParameter
	err := r.db.WithContext(ctx).Find(&result).Error
	return result, err
}

func (r *repository) FindByID(ctx context.Context, id int64) (*models.SafetyStockParameter, error) {
	var result models.SafetyStockParameter
	err := r.db.WithContext(ctx).First(&result, id).Error
	return &result, err
}

func (r *repository) FindByItemCode(ctx context.Context, itemCode string) (*models.SafetyStockParameter, error) {
	var data models.SafetyStockParameter

	err := r.db.WithContext(ctx).
		Where("item_uniq_code = ?", itemCode).
		First(&data).Error

	return &data, err
}

func (r *repository) Update(ctx context.Context, id int64, data map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&models.SafetyStockParameter{}).
		Where("id = ?", id).
		Updates(data).Error
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.SafetyStockParameter{}, id).Error
}

// forecasting (simple table)
func (r *repository) GetForecastByItem(ctx context.Context, itemCode string) (float64, error) {
	var result struct {
		ForecastQty float64
	}

	err := r.db.WithContext(ctx).
		Table("forecast_results").
		Select("forecast_qty").
		Where("item_code = ?", itemCode).
		Order("created_at DESC").
		Limit(1).
		Scan(&result).Error

	return result.ForecastQty, err
}
