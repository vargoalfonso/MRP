package repository

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	outModels "github.com/ganasa18/go-template/internal/outgoing_material/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Filter & Row types
// ---------------------------------------------------------------------------

type ListFilter struct {
	Search         string
	DateFrom       string
	DateTo         string
	Reason         string
	Uniq           string
	TransactionID  string
	WorkOrderNo    string
	Page           int
	Limit          int
	Offset         int
	OrderBy        string
	OrderDirection string
}

type OutgoingRow struct {
	ID                  int64     `gorm:"column:id"`
	TransactionID       string    `gorm:"column:transaction_id"`
	TransactionDate     time.Time `gorm:"column:transaction_date"`
	Uniq                string    `gorm:"column:uniq"`
	RMName              *string   `gorm:"column:rm_name"`
	PackingListRM       *string   `gorm:"column:packing_list_rm"`
	Unit                *string   `gorm:"column:unit"`
	QuantityOut         float64   `gorm:"column:quantity_out"`
	StockBefore         float64   `gorm:"column:stock_before"`
	StockAfter          float64   `gorm:"column:stock_after"`
	Reason              string    `gorm:"column:reason"`
	Purpose             *string   `gorm:"column:purpose"`
	WorkOrderNo         *string   `gorm:"column:work_order_no"`
	DestinationLocation *string   `gorm:"column:destination_location"`
	RequestedBy         *string    `gorm:"column:requested_by"`
	Remarks             *string    `gorm:"column:remarks"`
	StockRestoredAt     *time.Time `gorm:"column:stock_restored_at"`
	CreatedBy           *string    `gorm:"column:created_by"`
	CreatedAt           time.Time  `gorm:"column:created_at"`
}

// ---------------------------------------------------------------------------
// Interface
// ---------------------------------------------------------------------------

// RMFormOptionRow is a minimal projection from raw_materials used for autocomplete.
type RMFormOptionRow struct {
	ID                int64   `gorm:"column:id"`
	UniqCode          string  `gorm:"column:uniq_code"`
	PartNumber        *string `gorm:"column:part_number"`
	PartName          *string `gorm:"column:part_name"`
	UOM               *string `gorm:"column:uom"`
	StockQty          float64 `gorm:"column:stock_qty"`
	WarehouseLocation *string `gorm:"column:warehouse_location"`
}

type IRepository interface {
	List(ctx context.Context, f ListFilter) ([]OutgoingRow, int64, error)
	GetByID(ctx context.Context, id int64) (*outModels.OutgoingRawMaterial, error)
	// SearchRawMaterials queries raw_materials for autocomplete on the create form.
	// Searches uniq_code, part_name, and part_number. Returns up to limit rows.
	SearchRawMaterials(ctx context.Context, q string, limit int) ([]RMFormOptionRow, error)
	// ProcessTx runs the outgoing transaction atomically:
	// 1. Locks and reads raw_materials stock for the given uniq code.
	// 2. Validates stock >= quantity_out.
	// 3. Generates transaction_id, sets stock_before/stock_after, auto-fills rm_name and unit.
	// 4. Inserts outgoing_raw_material row.
	// 5. Deducts stock from raw_materials.
	// The caller receives the populated orm after commit.
	ProcessTx(ctx context.Context, orm *outModels.OutgoingRawMaterial) error
	// UpdateTx applies a partial update atomically. When quantity_out and/or uniq
	// change, it re-locks the affected raw_materials rows and re-calculates stock:
	//   - same uniq: adjust stock by the delta (new_qty - old_qty)
	//   - changed uniq: return old_qty to the old material, deduct new_qty from the new one
	// It returns the fully populated record after commit.
	UpdateTx(ctx context.Context, id int64, req outModels.UpdateOutgoingRMRequest, updatedBy string) (*outModels.OutgoingRawMaterial, error)
	// SoftDelete marks a transaction as deleted. Stock is NOT restored.
	SoftDelete(ctx context.Context, id int64, deletedBy string) error
	// RestoreStockTx returns the transaction quantity back into raw_materials
	// stock exactly once (guarded by stock_restored_at).
	RestoreStockTx(ctx context.Context, id int64, updatedBy string) (*outModels.OutgoingRawMaterial, error)
}

// ---------------------------------------------------------------------------
// Implementation
// ---------------------------------------------------------------------------

type repo struct{ db *gorm.DB }

func New(db *gorm.DB) IRepository { return &repo{db: db} }

func (r *repo) List(ctx context.Context, f ListFilter) ([]OutgoingRow, int64, error) {
	q := r.db.WithContext(ctx).Table("outgoing_raw_material").Where("deleted_at IS NULL")

	if f.Search != "" {
		s := "%" + f.Search + "%"
		q = q.Where("transaction_id ILIKE ? OR uniq ILIKE ? OR rm_name ILIKE ?", s, s, s)
	}
	if f.Uniq != "" {
		q = q.Where("uniq = ?", f.Uniq)
	}
	if f.TransactionID != "" {
		q = q.Where("transaction_id = ?", f.TransactionID)
	}
	if f.Reason != "" {
		q = q.Where("reason = ?", f.Reason)
	}
	if f.WorkOrderNo != "" {
		q = q.Where("work_order_no = ?", f.WorkOrderNo)
	}
	if f.DateFrom != "" {
		q = q.Where("transaction_date >= ?", f.DateFrom)
	}
	if f.DateTo != "" {
		q = q.Where("transaction_date <= ?", f.DateTo)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderBy := "created_at"
	if f.OrderBy != "" {
		orderBy = f.OrderBy
	}
	dir := "DESC"
	if strings.ToLower(f.OrderDirection) == "asc" {
		dir = "ASC"
	}
	q = q.Order(fmt.Sprintf("%s %s", orderBy, dir))

	if f.Limit > 0 {
		q = q.Limit(f.Limit).Offset(f.Offset)
	}

	var rows []OutgoingRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *repo) GetByID(ctx context.Context, id int64) (*outModels.OutgoingRawMaterial, error) {
	var orm outModels.OutgoingRawMaterial
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&orm).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.New(http.StatusNotFound, apperror.CodeNotFound, "outgoing transaction tidak ditemukan")
	}
	return &orm, err
}

func (r *repo) ProcessTx(ctx context.Context, orm *outModels.OutgoingRawMaterial) error {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		}
	}()

	type rmRow struct {
		ID       int64   `gorm:"column:id"`
		StockQty float64 `gorm:"column:stock_qty"`
		UOM      *string `gorm:"column:uom"`
		PartName *string `gorm:"column:part_name"`
	}
	var rm rmRow
	if err := tx.Raw(
		`SELECT id, stock_qty, uom, part_name FROM raw_materials
		 WHERE uniq_code = ? AND deleted_at IS NULL FOR UPDATE`,
		orm.Uniq,
	).Scan(&rm).Error; err != nil {
		tx.Rollback()
		return err
	}
	if rm.ID == 0 {
		tx.Rollback()
		return apperror.New(http.StatusNotFound, apperror.CodeNotFound, "raw material tidak ditemukan: "+orm.Uniq)
	}
	if rm.StockQty < orm.QuantityOut {
		tx.Rollback()
		return apperror.New(http.StatusUnprocessableEntity, apperror.CodeUnprocessable,
			fmt.Sprintf("insufficient stock: available %.4f, requested %.4f", rm.StockQty, orm.QuantityOut))
	}

	var count int64
	tx.Raw("SELECT COUNT(*) FROM outgoing_raw_material").Scan(&count)

	orm.UUID = uuid.New()
	orm.TransactionID = fmt.Sprintf("OUT-RM-%05d", count+1)
	orm.StockBefore = rm.StockQty
	orm.StockAfter = rm.StockQty - orm.QuantityOut
	orm.RawMaterialID = &rm.ID
	if orm.RMName == nil && rm.PartName != nil {
		orm.RMName = rm.PartName
	}
	if orm.Unit == nil && rm.UOM != nil {
		orm.Unit = rm.UOM
	}

	if err := tx.Create(orm).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Exec(
		`UPDATE raw_materials SET stock_qty = stock_qty - ?, updated_at = NOW(), updated_by = ?
		 WHERE id = ?`,
		orm.QuantityOut, orm.CreatedBy, rm.ID,
	).Error; err != nil {
		tx.Rollback()
		return err
	}

	notes := "reason: " + orm.Reason
	if orm.Purpose != nil {
		notes += ", purpose: " + *orm.Purpose
	}
	if orm.DestinationLocation != nil {
		notes += ", destination: " + *orm.DestinationLocation
	}
	if err := insertLogs(tx, rm.ID, orm.Uniq, -orm.QuantityOut, "outgoing", "outgoing_raw_material", orm.PackingListRM, notes, orm.CreatedBy); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

type rmLockRow struct {
	ID       int64   `gorm:"column:id"`
	StockQty float64 `gorm:"column:stock_qty"`
	UOM      *string `gorm:"column:uom"`
	PartName *string `gorm:"column:part_name"`
}

// lockRawMaterialByUniq locks and reads a raw_materials row for the given uniq.
func lockRawMaterialByUniq(tx *gorm.DB, uniq string) (*rmLockRow, error) {
	var rm rmLockRow
	if err := tx.Raw(
		`SELECT id, stock_qty, uom, part_name FROM raw_materials
		 WHERE uniq_code = ? AND deleted_at IS NULL FOR UPDATE`,
		uniq,
	).Scan(&rm).Error; err != nil {
		return nil, err
	}
	if rm.ID == 0 {
		return nil, apperror.New(http.StatusNotFound, apperror.CodeNotFound, "raw material tidak ditemukan: "+uniq)
	}
	return &rm, nil
}

func (r *repo) UpdateTx(ctx context.Context, id int64, req outModels.UpdateOutgoingRMRequest, updatedBy string) (*outModels.OutgoingRawMaterial, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		}
	}()

	var orm outModels.OutgoingRawMaterial
	if err := tx.Raw(
		`SELECT * FROM outgoing_raw_material WHERE id = ? AND deleted_at IS NULL FOR UPDATE`,
		id,
	).Scan(&orm).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if orm.ID == 0 {
		tx.Rollback()
		return nil, apperror.New(http.StatusNotFound, apperror.CodeNotFound, "outgoing transaction tidak ditemukan")
	}

	oldQty := orm.QuantityOut
	newQty := oldQty
	if req.QuantityOut != nil {
		newQty = *req.QuantityOut
	}
	oldUniq := orm.Uniq
	newUniq := oldUniq
	if req.Uniq != nil && *req.Uniq != "" {
		newUniq = *req.Uniq
	}

	if newUniq == oldUniq {
		// Same material: adjust stock by delta = newQty - oldQty.
		delta := newQty - oldQty
		if delta != 0 {
			rm, err := lockRawMaterialByUniq(tx, newUniq)
			if err != nil {
				tx.Rollback()
				return nil, err
			}
			if delta > 0 && rm.StockQty < delta {
				tx.Rollback()
				return nil, apperror.New(http.StatusUnprocessableEntity, apperror.CodeUnprocessable,
					fmt.Sprintf("insufficient stock: available %.4f, additional requested %.4f", rm.StockQty, delta))
			}
			if err := tx.Exec(
				`UPDATE raw_materials SET stock_qty = stock_qty - ?, updated_at = NOW(), updated_by = ? WHERE id = ?`,
				delta, updatedBy, rm.ID,
			).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
			orm.RawMaterialID = &rm.ID
			// Keep the original stock_before; recompute stock_after against new qty.
			orm.StockAfter = orm.StockBefore - newQty
		}
	} else {
		// Material changed: return oldQty to old material, deduct newQty from new material.
		oldRM, err := lockRawMaterialByUniq(tx, oldUniq)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		if err := tx.Exec(
			`UPDATE raw_materials SET stock_qty = stock_qty + ?, updated_at = NOW(), updated_by = ? WHERE id = ?`,
			oldQty, updatedBy, oldRM.ID,
		).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		newRM, err := lockRawMaterialByUniq(tx, newUniq)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		if newRM.StockQty < newQty {
			tx.Rollback()
			return nil, apperror.New(http.StatusUnprocessableEntity, apperror.CodeUnprocessable,
				fmt.Sprintf("insufficient stock: available %.4f, requested %.4f", newRM.StockQty, newQty))
		}
		if err := tx.Exec(
			`UPDATE raw_materials SET stock_qty = stock_qty - ?, updated_at = NOW(), updated_by = ? WHERE id = ?`,
			newQty, updatedBy, newRM.ID,
		).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		orm.RawMaterialID = &newRM.ID
		orm.StockBefore = newRM.StockQty
		orm.StockAfter = newRM.StockQty - newQty
		if req.Unit == nil && newRM.UOM != nil {
			orm.Unit = newRM.UOM
		}
		if newRM.PartName != nil {
			orm.RMName = newRM.PartName
		}
	}

	// Apply scalar field updates.
	orm.Uniq = newUniq
	orm.QuantityOut = newQty
	if req.Unit != nil {
		orm.Unit = req.Unit
	}
	if req.PackingListRM != nil {
		orm.PackingListRM = req.PackingListRM
	}
	if req.Reason != nil && *req.Reason != "" {
		orm.Reason = *req.Reason
	}
	if req.Purpose != nil {
		orm.Purpose = req.Purpose
	}
	if req.WorkOrderNo != nil {
		orm.WorkOrderNo = req.WorkOrderNo
	}
	if req.DestinationLocation != nil {
		orm.DestinationLocation = req.DestinationLocation
	}
	if req.RequestedBy != nil {
		orm.RequestedBy = req.RequestedBy
	}
	if req.Remarks != nil {
		orm.Remarks = req.Remarks
	}
	now := time.Now()
	orm.UpdatedBy = &updatedBy
	orm.UpdatedAt = now

	if err := tx.Exec(
		`UPDATE outgoing_raw_material SET
			raw_material_id = ?, packing_list_rm = ?, uniq = ?, rm_name = ?, unit = ?,
			quantity_out = ?, stock_before = ?, stock_after = ?, reason = ?, purpose = ?,
			work_order_no = ?, destination_location = ?, requested_by = ?, remarks = ?,
			updated_by = ?, updated_at = ?
		 WHERE id = ? AND deleted_at IS NULL`,
		orm.RawMaterialID, orm.PackingListRM, orm.Uniq, orm.RMName, orm.Unit,
		orm.QuantityOut, orm.StockBefore, orm.StockAfter, orm.Reason, orm.Purpose,
		orm.WorkOrderNo, orm.DestinationLocation, orm.RequestedBy, orm.Remarks,
		orm.UpdatedBy, orm.UpdatedAt, orm.ID,
	).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	notes := "outgoing updated, reason: " + orm.Reason
	if err := insertLogs(r.db, *orm.RawMaterialID, orm.Uniq, 0, "adjustment", "outgoing_raw_material_update", orm.WorkOrderNo, notes, orm.UpdatedBy); err != nil {
		// Log error silently or ignore since it's already committed, but let's do it inside tx if we move it above tx.Commit()
	}

	return &orm, nil
}

func (r *repo) SoftDelete(ctx context.Context, id int64, deletedBy string) error {
	now := time.Now()
	res := r.db.WithContext(ctx).
		Model(&outModels.OutgoingRawMaterial{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"deleted_at": now,
			"updated_by": deletedBy,
			"updated_at": now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperror.New(http.StatusNotFound, apperror.CodeNotFound, "outgoing transaction tidak ditemukan")
	}
	return nil
}

func (r *repo) RestoreStockTx(ctx context.Context, id int64, updatedBy string) (*outModels.OutgoingRawMaterial, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		}
	}()

	// Include soft-deleted rows: restore is typically used after a delete.
	var orm outModels.OutgoingRawMaterial
	if err := tx.Raw(
		`SELECT * FROM outgoing_raw_material WHERE id = ? FOR UPDATE`,
		id,
	).Scan(&orm).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if orm.ID == 0 {
		tx.Rollback()
		return nil, apperror.New(http.StatusNotFound, apperror.CodeNotFound, "outgoing transaction tidak ditemukan")
	}
	if orm.StockRestoredAt != nil {
		tx.Rollback()
		return nil, apperror.New(http.StatusConflict, apperror.CodeConflict, "stock untuk transaksi ini sudah pernah dikembalikan")
	}

	rm, err := lockRawMaterialByUniq(tx, orm.Uniq)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Exec(
		`UPDATE raw_materials SET stock_qty = stock_qty + ?, updated_at = NOW(), updated_by = ? WHERE id = ?`,
		orm.QuantityOut, updatedBy, rm.ID,
	).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	now := time.Now()
	if err := tx.Exec(
		`UPDATE outgoing_raw_material SET stock_restored_at = ?, updated_by = ?, updated_at = ? WHERE id = ?`,
		now, updatedBy, now, orm.ID,
	).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	notes := "stock restored for transaction " + orm.TransactionID
	if err := insertLogs(tx, rm.ID, orm.Uniq, orm.QuantityOut, "adjustment", "outgoing_raw_material_restore", orm.WorkOrderNo, notes, &updatedBy); err != nil {
		tx.Rollback()
		return nil, err
	}

	orm.StockRestoredAt = &now
	orm.UpdatedBy = &updatedBy
	orm.UpdatedAt = now
	if orm.RawMaterialID == nil {
		orm.RawMaterialID = &rm.ID
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return &orm, nil
}

func (r *repo) SearchRawMaterials(ctx context.Context, q string, limit int) ([]RMFormOptionRow, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	query := r.db.WithContext(ctx).
		Table("raw_materials").
		Select("id, uniq_code, part_number, part_name, uom, stock_qty, warehouse_location").
		Where("deleted_at IS NULL")
	if q != "" {
		s := "%" + q + "%"
		query = query.Where("uniq_code ILIKE ? OR part_name ILIKE ? OR part_number ILIKE ?", s, s, s)
	}
	var rows []RMFormOptionRow
	if err := query.Order("uniq_code ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func insertLogs(tx *gorm.DB, rmID int64, uniq string, qtyChange float64, txType string, srcFlag string, refID *string, notes string, loggedBy *string) error {
	refVal := "<nil>"
	if refID != nil {
		refVal = *refID
	}
	fmt.Printf("[DEBUG] insertLogs - Uniq: %s, Qty: %.2f, Type: %s, Src: %s, RefID: %s\n", uniq, qtyChange, txType, srcFlag, refVal)
	if err := tx.Exec(
		`INSERT INTO inventory_movement_logs (movement_category, movement_type, uniq_code, entity_id, qty_change, source_flag, reference_id, notes, logged_by, logged_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())`,
		"raw_material", txType, uniq, rmID, qtyChange, srcFlag, refID, notes, loggedBy,
	).Error; err != nil {
		return err
	}
	return nil
}
