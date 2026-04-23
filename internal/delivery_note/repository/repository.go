package repository

import (
	"context"
	"errors"

	"github.com/ganasa18/go-template/internal/delivery_note/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IDeliveryNoteRepository interface {
	UomByCode(ctx context.Context, itemUniqCode string) (string, error)

	FindLastDNNumber(ctx context.Context, tx *gorm.DB, prefix string) (string, error)
	GetPOByPONumber(ctx context.Context, poNumber string) (*models.PurchaseOrder, error)
	GetPOItemsByPOID(ctx context.Context, poID int64) ([]models.PurchaseOrderItem, error)
	GetTotalQtyByPOID(ctx context.Context, poID int64) (int64, error)
	CountDNIncomingByPONumber(ctx context.Context, poNumber string) (int64, error)

	Create(ctx context.Context, tx *gorm.DB, dn *models.DeliveryNote) error
	CreateItems(ctx context.Context, tx *gorm.DB, items []models.DeliveryNoteItem) error

	GetTotalDNCreatedByDNID(ctx context.Context, dnID int64) (int64, error)
	GetSupplierByID(ctx context.Context, supplierID int64) (*models.Supplier, error)
	CountDNByPrefix(ctx context.Context, prefix string) (int64, error)

	CountDNByPONumber(ctx context.Context, poNumber string) (int64, error)
	GetDNSummaryByPO(ctx context.Context, poNumber string) (*DNCountSummary, error)
	GetUsedQtyByItem(ctx context.Context, itemCode string) (int64, error)
	GetKanbanByItemCode(ctx context.Context, code string) (*models.KanbanParameter, error)

	GetPOItemByPackingNumber(ctx context.Context, packing string) (*models.DeliveryNoteItem, error)
	GetDNByID(ctx context.Context, id int64) (*models.DeliveryNote, error)
	CheckItemExistsInDN(ctx context.Context, itemCode string) (int64, error)

	WithTx(ctx context.Context, fn func(tx *gorm.DB) error) error

	// DN
	FindDNByNumber(ctx context.Context, tx *gorm.DB, dnNumber string) (*models.DeliveryNoteSupplier, error)
	CreateDN(ctx context.Context, tx *gorm.DB, dn *models.DeliveryNoteSupplier) error
	AddDNQty(ctx context.Context, tx *gorm.DB, dnID int64, qty float64) error

	// DN ITEM
	InsertDNItem(ctx context.Context, tx *gorm.DB, item *models.DeliveryNoteSupplierItem) error

	// STOCK
	GetFinishedGoodsForUpdate(ctx context.Context, tx *gorm.DB, uniq_code string) (*models.FinishedGoods, error)
	ReduceStockTx(ctx context.Context, tx *gorm.DB, fgID int64, qty float64) error

	// OPTIONAL
	IsDNItemDuplicate(ctx context.Context, tx *gorm.DB, dnID int64, kanban string) (bool, error)

	InsertHeaderTx(tx *gorm.DB, data *models.DeliveryScheduleCustomer) error
	InsertItemTx(tx *gorm.DB, data *models.DeliveryScheduleItemCustomer) error

	GetFGForUpdate(tx *gorm.DB, uniq string) (*models.FinishedGoods, error)
	ReduceFGStock(tx *gorm.DB, fgID int64, qty float64) error
}

type DNCountSummary struct {
	Total    int64
	Incoming int64
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IDeliveryNoteRepository {
	return &repository{db: db}
}

func (r *repository) GetUsedQtyByItem(ctx context.Context, itemCode string) (int64, error) {
	var total int64

	err := r.db.WithContext(ctx).
		Model(&models.DeliveryNoteItem{}).
		Select("COALESCE(SUM(quantity),0)").
		Where("item_uniq_code = ?", itemCode).
		Scan(&total).Error

	return total, err
}

func (r *repository) GetKanbanByItemCode(ctx context.Context, code string) (*models.KanbanParameter, error) {
	var k models.KanbanParameter

	err := r.db.WithContext(ctx).
		Where("item_uniq_code = ?", code).
		First(&k).Error

	return &k, err
}

func (r *repository) GetDNSummaryByPO(ctx context.Context, poNumber string) (*DNCountSummary, error) {
	var result DNCountSummary

	err := r.db.WithContext(ctx).
		Model(&models.DeliveryNote{}).
		Select(`
			COUNT(*) as total,
			COUNT(CASE WHEN status != 'draft' THEN 1 END) as incoming
		`).
		Where("po_number = ?", poNumber).
		Scan(&result).Error

	return &result, err
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
		Where("po_number = ?", poNumber).
		First(&po).Error

	if err != nil {
		return nil, err
	}

	return &po, nil
}

func (r *repository) GetPOItemsByPOID(ctx context.Context, poID int64) ([]models.PurchaseOrderItem, error) {
	var items []models.PurchaseOrderItem

	err := r.db.WithContext(ctx).
		Where("po_id = ?", poID).
		Find(&items).Error

	return items, err
}

func (r *repository) GetTotalQtyByPOID(ctx context.Context, poID int64) (int64, error) {
	var total float64

	err := r.db.WithContext(ctx).
		Model(&models.PurchaseOrderItem{}).
		Select("COALESCE(SUM(ordered_qty),0)").
		Where("po_id = ?", poID).
		Scan(&total).Error

	return int64(total), err
}

func (r *repository) CountDNIncomingByPONumber(ctx context.Context, poNumber string) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Table("delivery_notes").
		Where("po_number = ? AND status = ?", poNumber, "incoming").
		Count(&count).Error

	return count, err
}

func (r *repository) CheckItemExistsInDN(ctx context.Context, itemCode string) (int64, error) {
	var total int64

	err := r.db.WithContext(ctx).
		Model(&models.DeliveryNoteItem{}).
		Select("COALESCE(SUM(qty_received), 0)").
		Where("item_uniq_code = ?", itemCode).
		Where(`"check" = ?`, "completed").
		Scan(&total).Error

	return total, err
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

func (r *repository) CountDNByPONumber(ctx context.Context, poNumber string) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&models.DeliveryNote{}).
		Where("po_number = ?", poNumber).
		Count(&count).Error

	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *repository) GetPOItemByPackingNumber(ctx context.Context, packing string) (*models.DeliveryNoteItem, error) {
	var item models.DeliveryNoteItem

	err := r.db.WithContext(ctx).
		Where("packing_number = ?", packing).
		First(&item).Error

	if err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *repository) GetDNByID(ctx context.Context, id int64) (*models.DeliveryNote, error) {
	var dn models.DeliveryNote

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&dn).Error

	if err != nil {
		return nil, err
	}

	return &dn, nil
}

func (r *repository) WithTx(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *repository) FindDNByNumber(ctx context.Context, tx *gorm.DB, dnNumber string) (*models.DeliveryNoteSupplier, error) {
	var dn models.DeliveryNoteSupplier

	err := tx.WithContext(ctx).
		Where("dn_number = ?", dnNumber).
		First(&dn).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return &dn, err
}

func (r *repository) CreateDN(ctx context.Context, tx *gorm.DB, dn *models.DeliveryNoteSupplier) error {
	return tx.WithContext(ctx).Create(dn).Error
}

func (r *repository) AddDNQty(ctx context.Context, tx *gorm.DB, dnID int64, qty float64) error {
	return tx.WithContext(ctx).
		Model(&models.DeliveryNoteSupplier{}).
		Where("id = ?", dnID).
		Update("total_qty", gorm.Expr("total_qty + ?", qty)).Error
}

func (r *repository) InsertDNItem(ctx context.Context, tx *gorm.DB, item *models.DeliveryNoteSupplierItem) error {
	return tx.WithContext(ctx).Create(item).Error
}

func (r *repository) GetFinishedGoodsForUpdate(ctx context.Context, tx *gorm.DB, uniq_code string) (*models.FinishedGoods, error) {
	var fg models.FinishedGoods

	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("uniq_code = ?", uniq_code).
		First(&fg).Error

	return &fg, err
}

func (r *repository) ReduceStockTx(ctx context.Context, tx *gorm.DB, fgID int64, qty float64) error {
	return tx.WithContext(ctx).
		Model(&models.FinishedGoods{}).
		Where("id = ?", fgID).
		Update("stock_qty", gorm.Expr("stock_qty - ?", qty)).Error
}

func (r *repository) IsDNItemDuplicate(ctx context.Context, tx *gorm.DB, dnID int64, kanban string) (bool, error) {
	var count int64

	err := tx.WithContext(ctx).
		Model(&models.DeliveryNoteSupplierItem{}).
		Where("dn_id = ? AND kanban_number = ?", dnID, kanban).
		Count(&count).Error

	return count > 0, err
}

func (r *repository) InsertHeaderTx(tx *gorm.DB, data *models.DeliveryScheduleCustomer) error {
	return tx.Create(data).Error
}

func (r *repository) InsertItemTx(tx *gorm.DB, data *models.DeliveryScheduleItemCustomer) error {
	return tx.Create(data).Error
}

func (r *repository) GetFGForUpdate(tx *gorm.DB, uniq string) (*models.FinishedGoods, error) {
	var fg models.FinishedGoods

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("uniq_code = ?", uniq).
		First(&fg).Error

	return &fg, err
}

func (r *repository) ReduceFGStock(tx *gorm.DB, fgID int64, qty float64) error {
	return tx.Model(&models.FinishedGoods{}).
		Where("id = ?", fgID).
		Update("stock_qty", gorm.Expr("stock_qty - ?", qty)).Error
}
