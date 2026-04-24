package repository

import (
	"context"

	"github.com/ganasa18/go-template/internal/product_return/models"
	"gorm.io/gorm"
)

type IProductReturnRepository interface {
	Create(ctx context.Context, req models.CreateProductReturnRequest) (*models.ProductReturn, error)
	FindAll(ctx context.Context, page, limit int) ([]models.ProductReturn, int64, error)
	FindByID(ctx context.Context, id int64) (*models.ProductReturn, error)
	Update(ctx context.Context, id int64, req models.UpdateProductReturnRequest) (*models.ProductReturn, error)
	Delete(ctx context.Context, id int64) error
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IProductReturnRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, req models.CreateProductReturnRequest) (*models.ProductReturn, error) {
	data := models.ProductReturn{
		Uniq:           req.Uniq,
		DNNumber:       req.DNNumber,
		QuantityScrap:  req.QuantityScrap,
		QuantityRework: req.QuantityRework,
		Status:         req.Status,
	}

	err := r.db.WithContext(ctx).Create(&data).Error
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (r *repository) FindAll(ctx context.Context, page, limit int) ([]models.ProductReturn, int64, error) {
	var data []models.ProductReturn
	var total int64

	if page <= 0 {
		page = 1
	}

	if limit <= 0 {
		limit = 10
	}

	offset := (page - 1) * limit

	db := r.db.WithContext(ctx)

	// =====================================
	// COUNT TOTAL
	// =====================================
	err := db.
		Table("product_returns pr").
		Joins("LEFT JOIN delivery_note_items kp ON kp.item_uniq_code = pr.uniq").
		Joins("LEFT JOIN items i ON i.uniq_code = pr.uniq").
		Count(&total).Error

	if err != nil {
		return nil, 0, err
	}

	// =====================================
	// GET DATA
	// =====================================
	err = db.
		Table("product_returns pr").
		Select(`
			pr.*,
			kp.packing_number,
			i.part_name,
			i.part_number
		`).
		Joins("LEFT JOIN delivery_note_items kp ON kp.item_uniq_code = pr.uniq").
		Joins("LEFT JOIN items i ON i.uniq_code = pr.uniq").
		Order("pr.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&data).Error

	if err != nil {
		return nil, 0, err
	}

	return data, total, nil
}

func (r *repository) FindByID(ctx context.Context, id int64) (*models.ProductReturn, error) {
	var data models.ProductReturn
	err := r.db.WithContext(ctx).First(&data, id).Error
	return &data, err
}

func (r *repository) Update(ctx context.Context, id int64, req models.UpdateProductReturnRequest) (*models.ProductReturn, error) {
	var data models.ProductReturn

	err := r.db.WithContext(ctx).First(&data, id).Error
	if err != nil {
		return nil, err
	}

	data.Uniq = req.Uniq
	data.DNNumber = req.DNNumber
	data.QuantityScrap = req.QuantityScrap
	data.QuantityRework = req.QuantityRework
	data.Status = req.Status

	err = r.db.WithContext(ctx).Save(&data).Error
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.ProductReturn{}, id).Error
}
