package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ganasa18/go-template/internal/product_return/models"
	woModels "github.com/ganasa18/go-template/internal/work_order/models"
	"github.com/google/uuid"
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

func parseDate(d string) *time.Time {
	if d == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", d)
	if err == nil {
		return &t
	}
	t2, err2 := time.Parse(time.RFC3339, d)
	if err2 == nil {
		return &t2
	}
	return nil
}

func (r *repository) Create(ctx context.Context, req models.CreateProductReturnRequest) (*models.ProductReturn, error) {
	data := models.ProductReturn{
		Uniq:           req.Uniq,
		DNNumber:       req.DNNumber,
		QuantityScrap:  req.QuantityScrap,
		QuantityRework: req.QuantityRework,
		Status:         req.Status,
		Weight:         req.Weight,
		UOM:            req.UOM,
		ScrapType:      req.ScrapType,
		DateReceived:   parseDate(req.DateReceived),
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
	data.Weight = req.Weight
	data.UOM = req.UOM
	data.ScrapType = req.ScrapType
	if pd := parseDate(req.DateReceived); pd != nil {
		data.DateReceived = pd
	}

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&data).Error; err != nil {
			return err
		}

		// Auto-create Rework WO if approved and has rework quantity
		if req.Status == "APPROVED" && req.QuantityRework > 0 {
			woNumber := fmt.Sprintf("WO-RW-%s-%d", time.Now().Format("20060102"), data.ID)
			now := time.Now()

			wo := woModels.WorkOrder{
				UUID:           uuid.New(),
				WoNumber:       woNumber,
				WoType:         "Rework",
				WOKind:         "standard",
				Status:         "New",
				ApprovalStatus: "Approved",
				CreatedDate:    now,
			}
			if err := tx.Create(&wo).Error; err != nil {
				return err
			}

			woItem := woModels.WorkOrderItem{
				UUID:         uuid.New(),
				WoID:         wo.ID,
				ItemUniqCode: data.Uniq,
				Quantity:     float64(req.QuantityRework),
				Status:       "New",
				KanbanNumber: fmt.Sprintf("KBN-RW-%d", data.ID),
			}
			if err := tx.Create(&woItem).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.ProductReturn{}, id).Error
}
