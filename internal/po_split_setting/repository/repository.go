package repository

import (
	"context"

	"github.com/ganasa18/go-template/internal/po_split_setting/models"
	"gorm.io/gorm"
)

type IPOSplitSettingRepository interface {
	Create(ctx context.Context, data *models.POSplitSetting) error
	FindAll(ctx context.Context) ([]models.POSplitSetting, error)
	FindByID(ctx context.Context, id int64) (*models.POSplitSetting, error)
	Update(ctx context.Context, id int64, data map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IPOSplitSettingRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, data *models.POSplitSetting) error {
	return r.db.WithContext(ctx).Create(data).Error
}

func (r *repository) FindAll(ctx context.Context) ([]models.POSplitSetting, error) {
	var res []models.POSplitSetting
	err := r.db.WithContext(ctx).Find(&res).Error
	return res, err
}

func (r *repository) FindByID(ctx context.Context, id int64) (*models.POSplitSetting, error) {
	var data models.POSplitSetting
	err := r.db.WithContext(ctx).First(&data, id).Error
	return &data, err
}

func (r *repository) Update(ctx context.Context, id int64, data map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&models.POSplitSetting{}).
		Where("id = ?", id).
		Updates(data).Error
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.POSplitSetting{}, id).Error
}
