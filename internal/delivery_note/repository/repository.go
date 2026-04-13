package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ganasa18/go-template/internal/delivery_note/models"
	"gorm.io/gorm"
)

type IDeliveryNoteRepository interface {
	UomByCode(ctx context.Context, itemUniqCode string) (string, error)

	FindLastDNNumber(ctx context.Context, tx *gorm.DB, prefix string) (string, error)
	GetPOByPONumber(ctx context.Context, poNumber string) (*models.PurchaseOrder, error)
	GetPOItemsByPOID(ctx context.Context, poID int64) ([]models.PurchaseOrderItem, error)
	GetTotalQtyByPOID(ctx context.Context, poID int64) (int, error)
	CountDNIncomingByPONumber(ctx context.Context, poNumber string) (int64, error)
	GetKanbanByItemCode(ctx context.Context, itemCode string) (*models.KanbanParameter, error)

	Create(ctx context.Context, tx *gorm.DB, dn *models.DeliveryNote) error
	CreateItems(ctx context.Context, tx *gorm.DB, items []models.DeliveryNoteItem) error

	GetTotalDNCreatedByDNID(ctx context.Context, dnID int64) (int64, error)
	GetSupplierByID(ctx context.Context, supplierID int64) (*models.Supplier, error)
	CountDNByPrefix(ctx context.Context, prefix string) (int64, error)
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IDeliveryNoteRepository {
	return &repository{db: db}
}

func (r *repository) UomByCode(ctx context.Context, itemUniqCode string) (string, error) {
	var uomName string

	err := r.db.WithContext(ctx).
		Table("items").
		Select("uom_parameters.name").
		Joins("JOIN uom_parameters ON uom_parameters.id = items.uom_id").
		Where("items.uniq_code = ?", itemUniqCode).
		Scan(&uomName).Error

	if err != nil {
		return "", err
	}

	return uomName, nil
}

func (r *repository) FindLastDNNumber(ctx context.Context, tx *gorm.DB, prefix string) (string, error) {
	var dn models.DeliveryNote

	err := tx.WithContext(ctx).
		Where("dn_number LIKE ?", prefix+"%").
		Order("dn_number DESC").
		First(&dn).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}

	return dn.DNNumber, nil
}

func (r *repository) GetPOByPONumber(ctx context.Context, poNumber string) (*models.PurchaseOrder, error) {
	var po models.PurchaseOrder

	err := r.db.WithContext(ctx).
		Table("purchase_orders").
		Where("po_number = ?", poNumber).
		First(&po).Error

	if err != nil {
		return nil, fmt.Errorf("purchase order with po_number %s not found: %w", poNumber, err)
	}

	return &po, nil
}

func (r *repository) GetPOItemsByPOID(ctx context.Context, poID int64) ([]models.PurchaseOrderItem, error) {
	var items []models.PurchaseOrderItem

	err := r.db.WithContext(ctx).
		Table("purchase_order_items").
		Where("po_id = ?", poID).
		Find(&items).Error

	if err != nil {
		return nil, err
	}

	return items, nil
}

func (r *repository) GetTotalQtyByPOID(ctx context.Context, poID int64) (int, error) {
	var total int

	err := r.db.WithContext(ctx).
		Table("purchase_order_items").
		Select("COALESCE(ROUND(SUM(ordered_qty)), 0)::int").
		Where("po_id = ?", poID).
		Scan(&total).Error

	if err != nil {
		return 0, err
	}

	return total, nil
}

func (r *repository) CountDNIncomingByPONumber(ctx context.Context, poNumber string) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Table("delivery_notes").
		Where("po_number = ? AND status = ?", poNumber, "incoming").
		Count(&count).Error

	return count, err
}

func (r *repository) GetKanbanByItemCode(ctx context.Context, itemCode string) (*models.KanbanParameter, error) {
	var kb models.KanbanParameter

	err := r.db.WithContext(ctx).
		Table("kanban_parameters").
		Where("item_uniq_code = ?", itemCode).
		First(&kb).Error

	if err != nil {
		return nil, err
	}

	return &kb, nil
}

func (r *repository) Create(ctx context.Context, tx *gorm.DB, dn *models.DeliveryNote) error {
	return tx.WithContext(ctx).Create(dn).Error
}

func (r *repository) CreateItems(ctx context.Context, tx *gorm.DB, items []models.DeliveryNoteItem) error {
	return tx.WithContext(ctx).Create(&items).Error
}

func (r *repository) GetTotalDNCreatedByDNID(ctx context.Context, dnID int64) (int64, error) {
	var total int64

	err := r.db.WithContext(ctx).
		Table("delivery_note_items").
		Select("COALESCE(SUM(quantity), 0)").
		Where("dn_id = ?", dnID).
		Scan(&total).Error

	if err != nil {
		return 0, err
	}

	return total, nil
}

func (r *repository) GetSupplierByID(ctx context.Context, supplierID int64) (*models.Supplier, error) {
	var supplier models.Supplier
	err := r.db.WithContext(ctx).
		Table("suppliers").
		Where("id = ?", supplierID).
		First(&supplier).Error
	if err != nil {
		return nil, err
	}
	return &supplier, nil
}

func (r *repository) CountDNByPrefix(ctx context.Context, prefix string) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&models.DeliveryNote{}).
		Where("dn_number LIKE ?", prefix+"%").
		Count(&count).Error

	if err != nil {
		return 0, err
	}

	return count, nil
}
