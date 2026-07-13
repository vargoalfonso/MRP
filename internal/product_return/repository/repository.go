package repository

import (
	"context"
	"time"

	"github.com/ganasa18/go-template/internal/product_return/models"
	"gorm.io/gorm"
)

type IProductReturnRepository interface {
	Create(ctx context.Context, req models.CreateProductReturnRequest) (*models.ProductReturn, error)
	FindAll(ctx context.Context, page, limit int) ([]ProductReturn, int64, error)
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

type ProductReturn struct {
	ID             uint      `gorm:"column:id" json:"id"`
	Uniq           string    `gorm:"column:uniq" json:"uniq"`
	DNNumber       string    `gorm:"column:dn_number" json:"dn_number"`
	QuantityScrap  int       `gorm:"column:quantity_scrap" json:"quantity_scrap"`
	QuantityRework int       `gorm:"column:quantity_rework" json:"quantity_rework"`
	Status         string    `gorm:"column:status" json:"status"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`

	// BRD fields (kolom hasil migration 0078) — ikut ter-select via pr.*
	Weight       float64    `gorm:"column:weight" json:"weight"`
	UOM          string     `gorm:"column:uom" json:"uom"`
	DateReceived *time.Time `gorm:"column:date_received" json:"date_received"`
	ScrapType    string     `gorm:"column:scrap_type" json:"scrap_type"`

	// Hasil JOIN
	PackingNumber string `gorm:"column:packing_number" json:"packing_number"`
	PartName      string `gorm:"column:part_name" json:"part_name"`
	PartNumber    string `gorm:"column:part_number" json:"part_number"`
}

func (r *repository) FindAll(ctx context.Context, page, limit int) ([]ProductReturn, int64, error) {
	var (
		data  []ProductReturn
		total int64
	)

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
	if err := db.
		Table("product_returns").
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// =====================================
	// GET DATA
	// =====================================
	if err := db.
		Table("product_returns pr").
		Select(`
			pr.*,
			kp.packing_number,
			i.part_name,
			i.part_number
		`).
		Joins(`
			LEFT JOIN (
				SELECT DISTINCT ON (item_uniq_code)
					item_uniq_code,
					packing_number
				FROM delivery_note_items
				ORDER BY item_uniq_code, id DESC
			) kp ON kp.item_uniq_code = pr.uniq
		`).
		Joins(`
			LEFT JOIN (
				SELECT DISTINCT ON (uniq_code)
					uniq_code,
					part_name,
					part_number
				FROM items
				ORDER BY uniq_code, id DESC
			) i ON i.uniq_code = pr.uniq
		`).
		Order("pr.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&data).Error; err != nil {
		return nil, 0, err
	}

	return data, total, nil
}

func (r *repository) FindByID(ctx context.Context, id int64) (*models.ProductReturn, error) {
	var data models.ProductReturn
	err := r.db.WithContext(ctx).
		Table("product_returns pr").
		Select(`
			pr.*,
			kp.packing_number,
			i.part_name,
			i.part_number,
			i.model
		`).
		Joins(`
			LEFT JOIN (
				SELECT DISTINCT ON (item_uniq_code)
					item_uniq_code,
					packing_number
				FROM delivery_note_items
				ORDER BY item_uniq_code, id DESC
			) kp ON kp.item_uniq_code = pr.uniq
		`).
		Joins(`
			LEFT JOIN (
				SELECT DISTINCT ON (uniq_code)
					uniq_code,
					part_name,
					part_number,
					model
				FROM items
				ORDER BY uniq_code, id DESC
			) i ON i.uniq_code = pr.uniq
		`).
		Where("pr.id = ?", id).
		Scan(&data).Error
	if err != nil {
		return nil, err
	}
	if data.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &data, nil
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
