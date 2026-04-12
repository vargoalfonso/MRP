package repository

import (
	"context"

	"github.com/ganasa18/go-template/internal/delivery_note/models"
	"gorm.io/gorm"
)

type IDeliveryNoteRepository interface {
	Create(ctx context.Context, tx *gorm.DB, dn *models.DeliveryNote) error
	CreateItems(ctx context.Context, tx *gorm.DB, items []models.DeliveryNoteItem) error
	FindLastDNNumber(ctx context.Context, tx *gorm.DB, prefix string) (string, error)
	UomByCode(ctx context.Context, itemUniqCode string) (string, error)
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IDeliveryNoteRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, tx *gorm.DB, dn *models.DeliveryNote) error {
	return tx.WithContext(ctx).Create(dn).Error
}

func (r *repository) CreateItems(ctx context.Context, tx *gorm.DB, items []models.DeliveryNoteItem) error {
	return tx.WithContext(ctx).Create(&items).Error
}

func (r *repository) FindLastDNNumber(ctx context.Context, tx *gorm.DB, prefix string) (string, error) {
	var last string

	err := tx.WithContext(ctx).
		Model(&models.DeliveryNote{}).
		Select("dn_number").
		Where("dn_number LIKE ?", prefix+"%").
		Order("dn_number DESC").
		Limit(1).
		Scan(&last).Error

	return last, err
}

func (r *repository) UomByCode(ctx context.Context, itemUniqCode string) (string, error) {
	var uomName string

	err := r.db.WithContext(ctx).
		Table("items").
		Select("uom").
		Where("uniq_code = ?", itemUniqCode).
		Scan(&uomName).Error

	if err != nil {
		return "", err
	}

	return uomName, nil
}
