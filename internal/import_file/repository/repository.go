package repository

import (
	"context"

	"github.com/ganasa18/go-template/internal/import_file/models"
	"gorm.io/gorm"
)

type ImportRepository interface {
	GetLatestCustomerByName(ctx context.Context, name string) (*models.Customer, error)
	CountPRL(ctx context.Context) (int64, error)
	InsertPRL(ctx context.Context, prl *models.PRL) error
	InsertPRLBulk(ctx context.Context, prls []models.PRL) error
	GetItemByUniqCode(ctx context.Context, uniqCode string) (*models.Item, error)
	GetMaxPRLNumber(ctx context.Context, year string) (int64, error)
}

type importRepository struct {
	db *gorm.DB
}

func New(db *gorm.DB) ImportRepository {
	return &importRepository{db: db}
}

func (r *importRepository) GetLatestCustomerByName(ctx context.Context, name string) (*models.Customer, error) {
	var customer models.Customer

	err := r.db.WithContext(ctx).
		Where("customer_name = ? AND deleted_at IS NULL", name).
		Order("created_at DESC").
		Limit(1).
		Find(&customer).Error

	if err != nil {
		return nil, err
	}

	// kalau tidak ditemukan
	if customer.ID == 0 {
		return nil, nil
	}

	return &customer, nil
}

func (r *importRepository) CountPRL(ctx context.Context) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&models.PRL{}).
		Where("deleted_at IS NULL").
		Count(&count).Error

	return count, err
}

func (r *importRepository) InsertPRL(ctx context.Context, prl *models.PRL) error {
	return r.db.WithContext(ctx).
		Create(prl).
		Error
}

func (r *importRepository) InsertPRLBulk(ctx context.Context, prls []models.PRL) error {
	if len(prls) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).
		CreateInBatches(prls, 500).
		Error
}

func (r *importRepository) GetItemByUniqCode(ctx context.Context, uniqCode string) (*models.Item, error) {
	var item models.Item

	err := r.db.WithContext(ctx).
		Where("uniq_code = ? AND deleted_at IS NULL", uniqCode).
		Order("created_at DESC").
		First(&item).Error

	if err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *importRepository) GetMaxPRLNumber(ctx context.Context, year string) (int64, error) {
	var max int64

	err := r.db.WithContext(ctx).
		Raw(`
			SELECT COALESCE(MAX(
				CAST(SPLIT_PART(prl_id, '-', 3) AS BIGINT)
			), 0)
			FROM prls
			WHERE prl_id LIKE ?
		`, "PRL-"+year+"-%").
		Scan(&max).Error

	return max, err
}
