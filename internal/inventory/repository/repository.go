package repository

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	invModels "github.com/ganasa18/go-template/internal/inventory/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Filter types
// ---------------------------------------------------------------------------

type ListFilter struct {
	Search         string
	RMType         string
	RMSource       string
	Status         string
	BuyNotBuy      string
	Page           int
	Limit          int
	Offset         int
	OrderBy        string
	OrderDirection string
}

type SubconListFilter struct {
	Search         string
	PONumber       string
	SupplierID     int64
	Period         string
	Status         string
	Page           int
	Limit          int
	Offset         int
	OrderBy        string
	OrderDirection string
}

type IncomingListFilter struct {
	Search     string
	DNType     string // RM | INDIRECT | SUBCON
	PONumber   string
	Status     string // pending | in_progress | approved | rejected
	SupplierID int64
	Page       int
	Limit      int
	Offset     int
}

// ---------------------------------------------------------------------------
// Row types returned by queries
// ---------------------------------------------------------------------------

type RawMaterialRow struct {
	ID                    int64     `gorm:"column:id"`
	UniqCode              string    `gorm:"column:uniq_code"`
	PartNumber            *string   `gorm:"column:part_number"`
	PartName              *string   `gorm:"column:part_name"`
	RawMaterialType       string    `gorm:"column:raw_material_type"`
	RMSource              string    `gorm:"column:rm_source"`
	WarehouseLocation     *string   `gorm:"column:warehouse_location"`
	UOM                   *string   `gorm:"column:uom"`
	StockQty              float64   `gorm:"column:stock_qty"`
	StockWeightKg         *float64  `gorm:"column:stock_weight_kg"`
	KanbanCount           *int      `gorm:"column:kanban_count"`
	KanbanStandardQty     *int      `gorm:"column:kanban_standard_qty"`
	SafetyStockQty        *float64  `gorm:"column:safety_stock_qty"`
	DailyUsageQty         *float64  `gorm:"column:daily_usage_qty"`
	Status                string    `gorm:"column:status"`
	StockDays             *int      `gorm:"column:stock_days"`
	BuyNotBuy             string    `gorm:"column:buy_not_buy"`
	StockToCompleteKanban *float64  `gorm:"column:stock_to_complete_kanban"`
	CreatedBy             *string   `gorm:"column:created_by"`
	CreatedAt             time.Time `gorm:"column:created_at"`
	UpdatedBy             *string   `gorm:"column:updated_by"`
	UpdatedAt             time.Time `gorm:"column:updated_at"`
}

type IndirectRow struct {
	ID                    int64     `gorm:"column:id"`
	UniqCode              string    `gorm:"column:uniq_code"`
	PartNumber            *string   `gorm:"column:part_number"`
	PartName              *string   `gorm:"column:part_name"`
	WarehouseLocation     *string   `gorm:"column:warehouse_location"`
	UOM                   *string   `gorm:"column:uom"`
	StockQty              float64   `gorm:"column:stock_qty"`
	StockWeightKg         *float64  `gorm:"column:stock_weight_kg"`
	KanbanCount           *int      `gorm:"column:kanban_count"`
	KanbanStandardQty     *int      `gorm:"column:kanban_standard_qty"`
	SafetyStockQty        *float64  `gorm:"column:safety_stock_qty"`
	DailyUsageQty         *float64  `gorm:"column:daily_usage_qty"`
	Status                *string   `gorm:"column:status"`
	StockDays             *int      `gorm:"column:stock_days"`
	BuyNotBuy             string    `gorm:"column:buy_not_buy"`
	StockToCompleteKanban *float64  `gorm:"column:stock_to_complete_kanban"`
	CreatedBy             *string   `gorm:"column:created_by"`
	CreatedAt             time.Time `gorm:"column:created_at"`
	UpdatedBy             *string   `gorm:"column:updated_by"`
	UpdatedAt             time.Time `gorm:"column:updated_at"`
}

type SubconRow struct {
	ID               int64      `gorm:"column:id"`
	UniqCode         string     `gorm:"column:uniq_code"`
	PartNumber       *string    `gorm:"column:part_number"`
	PartName         *string    `gorm:"column:part_name"`
	PONumber         *string    `gorm:"column:po_number"`
	POPeriod         *string    `gorm:"column:po_period"`
	SubconVendorID   *int64     `gorm:"column:subcon_vendor_id"`
	SubconVendorName *string    `gorm:"column:subcon_vendor_name"`
	StockAtVendorQty float64    `gorm:"column:stock_at_vendor_qty"`
	TotalPOQty       *float64   `gorm:"column:total_po_qty"`
	TotalReceivedQty *float64   `gorm:"column:total_received_qty"`
	DeltaPO          *float64   `gorm:"column:delta_po"`
	SafetyStockQty   *float64   `gorm:"column:safety_stock_qty"`
	DateDelivery     *time.Time `gorm:"column:date_delivery"`
	Status           string     `gorm:"column:status"`
	CreatedBy        *string    `gorm:"column:created_by"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedBy        *string    `gorm:"column:updated_by"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
}

type HistoryRow struct {
	ID           int64     `gorm:"column:id"`
	UniqCode     string    `gorm:"column:uniq_code"`
	QtyChange    float64   `gorm:"column:qty_change"`
	WeightChange *float64  `gorm:"column:weight_change"`
	MovementType string    `gorm:"column:movement_type"`
	SourceFlag   *string   `gorm:"column:source_flag"`
	DNNumber     *string   `gorm:"column:dn_number"`
	Notes        *string   `gorm:"column:notes"`
	LoggedBy     *string   `gorm:"column:logged_by"`
	LoggedAt     time.Time `gorm:"column:logged_at"`
	LogStatus    string    `gorm:"column:log_status"` // confirmed | pending | in_progress | rejected
}

type IncomingRow struct {
	ScanID       int64     `gorm:"column:scan_id"`
	ItemUniqCode string    `gorm:"column:item_uniq_code"`
	IncomingQty  float64   `gorm:"column:incoming_qty"`
	Warehouse    *string   `gorm:"column:warehouse"`
	ScanDate     time.Time `gorm:"column:scan_date"`
	SupplierName *string   `gorm:"column:supplier_name"`
	PONumber     *string   `gorm:"column:po_number"`
	DNNumber     string    `gorm:"column:dn_number"`
	QCStatus     string    `gorm:"column:qc_status"`
	UOM          *string   `gorm:"column:uom"`
	DNType       string    `gorm:"column:dn_type"`
}

type RMStats struct {
	TotalItems         int64 `gorm:"column:total_items"`
	BuyRecommendations int64 `gorm:"column:buy_recommendations"`
	LowStockItems      int64 `gorm:"column:low_stock_items"`
}

// ---------------------------------------------------------------------------
// Interface
// ---------------------------------------------------------------------------

type IRepository interface {
	// Raw Material
	ListRawMaterials(ctx context.Context, f ListFilter) ([]RawMaterialRow, int64, error)
	GetRawMaterialStats(ctx context.Context) (*RMStats, error)
	GetRawMaterialByID(ctx context.Context, id int64) (*invModels.RawMaterial, error)
	CreateRawMaterial(ctx context.Context, rm *invModels.RawMaterial) error
	BulkCreateRawMaterials(ctx context.Context, items []invModels.RawMaterial) error
	UpdateRawMaterial(ctx context.Context, id int64, updates map[string]interface{}) (*invModels.RawMaterial, error)
	SoftDeleteRawMaterial(ctx context.Context, id int64, deletedBy string) error
	GetMovementHistory(ctx context.Context, category, uniqCode string, f ListFilter) ([]HistoryRow, int64, error)

	// Indirect Raw Material
	ListIndirectMaterials(ctx context.Context, f ListFilter) ([]IndirectRow, int64, error)
	GetIndirectStats(ctx context.Context) (*RMStats, error)
	GetIndirectByID(ctx context.Context, id int64) (*invModels.IndirectRawMaterial, error)
	CreateIndirectMaterial(ctx context.Context, irm *invModels.IndirectRawMaterial) error
	BulkCreateIndirectMaterials(ctx context.Context, items []invModels.IndirectRawMaterial) error
	UpdateIndirectMaterial(ctx context.Context, id int64, updates map[string]interface{}) (*invModels.IndirectRawMaterial, error)
	SoftDeleteIndirectMaterial(ctx context.Context, id int64, deletedBy string) error

	// Subcon Inventory
	ListSubconInventory(ctx context.Context, f SubconListFilter) ([]SubconRow, int64, error)
	GetSubconByID(ctx context.Context, id int64) (*invModels.SubconInventory, error)
	CreateSubconInventory(ctx context.Context, si *invModels.SubconInventory) error
	UpdateSubconInventory(ctx context.Context, id int64, updates map[string]interface{}) (*invModels.SubconInventory, error)
	SoftDeleteSubconInventory(ctx context.Context, id int64, deletedBy string) error

	// Incoming scans (cross-type tab view)
	ListIncoming(ctx context.Context, f IncomingListFilter) ([]IncomingRow, int64, error)
}

// ---------------------------------------------------------------------------
// Implementation
// ---------------------------------------------------------------------------

type repo struct{ db *gorm.DB }

func New(db *gorm.DB) IRepository { return &repo{db: db} }

// ---------------------------------------------------------------------------
// Raw Material
// ---------------------------------------------------------------------------

func (r *repo) ListRawMaterials(ctx context.Context, f ListFilter) ([]RawMaterialRow, int64, error) {
	q := r.db.WithContext(ctx).Table("raw_materials rm").Where("rm.deleted_at IS NULL")

	if f.Search != "" {
		s := "%" + f.Search + "%"
		q = q.Where("(rm.uniq_code ILIKE ? OR rm.part_name ILIKE ?)", s, s)
	}
	if f.RMType != "" {
		q = q.Where("rm.raw_material_type = ?", f.RMType)
	}
	if f.RMSource != "" {
		q = q.Where("rm.rm_source = ?", f.RMSource)
	}
	if f.Status != "" {
		q = q.Where("rm.status = ?", f.Status)
	}
	if f.BuyNotBuy != "" {
		q = q.Where("rm.buy_not_buy = ?", f.BuyNotBuy)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("ListRawMaterials count: %w", err)
	}

	var rows []RawMaterialRow
	err := q.Select("rm.*").
		Order(safeOrderDir("rm", f.OrderBy, f.OrderDirection, []string{"uniq_code", "stock_qty", "status", "stock_days", "created_at", "updated_at"})).
		Limit(f.Limit).Offset(f.Offset).
		Scan(&rows).Error
	return rows, total, err
}

func (r *repo) GetRawMaterialStats(ctx context.Context) (*RMStats, error) {
	var stats RMStats
	err := r.db.WithContext(ctx).Table("raw_materials").
		Select(`COUNT(*) AS total_items,
			SUM(CASE WHEN buy_not_buy = 'buy' THEN 1 ELSE 0 END) AS buy_recommendations,
			SUM(CASE WHEN status = 'low_on_stock' THEN 1 ELSE 0 END) AS low_stock_items`).
		Where("deleted_at IS NULL").
		Scan(&stats).Error
	return &stats, err
}

func (r *repo) GetRawMaterialByID(ctx context.Context, id int64) (*invModels.RawMaterial, error) {
	var rm invModels.RawMaterial
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&rm).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.New(http.StatusNotFound, apperror.CodeNotFound, "raw material not found")
	}
	return &rm, err
}

func (r *repo) CreateRawMaterial(ctx context.Context, rm *invModels.RawMaterial) error {
	return r.db.WithContext(ctx).Create(rm).Error
}

func (r *repo) BulkCreateRawMaterials(ctx context.Context, items []invModels.RawMaterial) error {
	return r.db.WithContext(ctx).CreateInBatches(items, 100).Error
}

func (r *repo) UpdateRawMaterial(ctx context.Context, id int64, updates map[string]interface{}) (*invModels.RawMaterial, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		}
	}()

	result := tx.Model(&invModels.RawMaterial{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(updates)
	if result.Error != nil {
		tx.Rollback()
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		return nil, apperror.New(http.StatusNotFound, apperror.CodeNotFound, "raw material not found")
	}

	if _, stockChanged := updates["stock_qty"]; stockChanged {
		if err := recalculateRMStatus(tx, id); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return r.GetRawMaterialByID(ctx, id)
}

func (r *repo) SoftDeleteRawMaterial(ctx context.Context, id int64, deletedBy string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Table("raw_materials").
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"deleted_at": &now,
			"updated_by": &deletedBy,
			"updated_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.New(http.StatusNotFound, apperror.CodeNotFound, "raw material not found")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Indirect Raw Material
// ---------------------------------------------------------------------------

func (r *repo) ListIndirectMaterials(ctx context.Context, f ListFilter) ([]IndirectRow, int64, error) {
	q := r.db.WithContext(ctx).Table("indirect_raw_materials irm").Where("irm.deleted_at IS NULL")

	if f.Search != "" {
		s := "%" + f.Search + "%"
		q = q.Where("(irm.uniq_code ILIKE ? OR irm.part_name ILIKE ?)", s, s)
	}
	if f.Status != "" {
		q = q.Where("irm.status = ?", f.Status)
	}
	if f.BuyNotBuy != "" {
		q = q.Where("irm.buy_not_buy = ?", f.BuyNotBuy)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("ListIndirectMaterials count: %w", err)
	}

	var rows []IndirectRow
	err := q.Select("irm.*").
		Order(safeOrderDir("irm", f.OrderBy, f.OrderDirection, []string{"uniq_code", "stock_qty", "status", "stock_days", "created_at", "updated_at"})).
		Limit(f.Limit).Offset(f.Offset).
		Scan(&rows).Error
	return rows, total, err
}

func (r *repo) GetIndirectStats(ctx context.Context) (*RMStats, error) {
	var stats RMStats
	err := r.db.WithContext(ctx).Table("indirect_raw_materials").
		Select(`COUNT(*) AS total_items,
			SUM(CASE WHEN buy_not_buy = 'buy' THEN 1 ELSE 0 END) AS buy_recommendations,
			SUM(CASE WHEN status = 'low_on_stock' THEN 1 ELSE 0 END) AS low_stock_items`).
		Where("deleted_at IS NULL").
		Scan(&stats).Error
	return &stats, err
}

func (r *repo) GetIndirectByID(ctx context.Context, id int64) (*invModels.IndirectRawMaterial, error) {
	var irm invModels.IndirectRawMaterial
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&irm).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.New(http.StatusNotFound, apperror.CodeNotFound, "indirect raw material not found")
	}
	return &irm, err
}

func (r *repo) CreateIndirectMaterial(ctx context.Context, irm *invModels.IndirectRawMaterial) error {
	return r.db.WithContext(ctx).Create(irm).Error
}

func (r *repo) BulkCreateIndirectMaterials(ctx context.Context, items []invModels.IndirectRawMaterial) error {
	return r.db.WithContext(ctx).CreateInBatches(items, 100).Error
}

func (r *repo) UpdateIndirectMaterial(ctx context.Context, id int64, updates map[string]interface{}) (*invModels.IndirectRawMaterial, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		}
	}()

	result := tx.Model(&invModels.IndirectRawMaterial{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(updates)
	if result.Error != nil {
		tx.Rollback()
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		return nil, apperror.New(http.StatusNotFound, apperror.CodeNotFound, "indirect raw material not found")
	}

	if _, stockChanged := updates["stock_qty"]; stockChanged {
		if err := recalculateIndirectStatus(tx, id); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return r.GetIndirectByID(ctx, id)
}

func (r *repo) SoftDeleteIndirectMaterial(ctx context.Context, id int64, deletedBy string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Table("indirect_raw_materials").
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"deleted_at": &now,
			"updated_by": &deletedBy,
			"updated_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.New(http.StatusNotFound, apperror.CodeNotFound, "indirect raw material not found")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Subcon Inventory
// ---------------------------------------------------------------------------

func (r *repo) ListSubconInventory(ctx context.Context, f SubconListFilter) ([]SubconRow, int64, error) {
	q := r.db.WithContext(ctx).Table("subcon_inventories si").Where("si.deleted_at IS NULL")

	if f.Search != "" {
		s := "%" + f.Search + "%"
		q = q.Where("(si.uniq_code ILIKE ? OR si.part_name ILIKE ? OR si.subcon_vendor_name ILIKE ?)", s, s, s)
	}
	if f.PONumber != "" {
		q = q.Where("si.po_number = ?", f.PONumber)
	}
	if f.SupplierID > 0 {
		q = q.Where("si.subcon_vendor_id = ?", f.SupplierID)
	}
	if f.Period != "" {
		q = q.Where("si.po_period = ?", f.Period)
	}
	if f.Status != "" {
		q = q.Where("si.status = ?", f.Status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("ListSubconInventory count: %w", err)
	}

	var rows []SubconRow
	err := q.Select("si.*").
		Order(safeOrderDir("si", f.OrderBy, f.OrderDirection, []string{"uniq_code", "po_number", "status", "created_at", "updated_at"})).
		Limit(f.Limit).Offset(f.Offset).
		Scan(&rows).Error
	return rows, total, err
}

func (r *repo) GetSubconByID(ctx context.Context, id int64) (*invModels.SubconInventory, error) {
	var si invModels.SubconInventory
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&si).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.New(http.StatusNotFound, apperror.CodeNotFound, "subcon inventory not found")
	}
	return &si, err
}

func (r *repo) CreateSubconInventory(ctx context.Context, si *invModels.SubconInventory) error {
	return r.db.WithContext(ctx).Create(si).Error
}

func (r *repo) UpdateSubconInventory(ctx context.Context, id int64, updates map[string]interface{}) (*invModels.SubconInventory, error) {
	result := r.db.WithContext(ctx).Model(&invModels.SubconInventory{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, apperror.New(http.StatusNotFound, apperror.CodeNotFound, "subcon inventory not found")
	}
	return r.GetSubconByID(ctx, id)
}

func (r *repo) SoftDeleteSubconInventory(ctx context.Context, id int64, deletedBy string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Table("subcon_inventories").
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"deleted_at": &now,
			"updated_by": &deletedBy,
			"updated_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperror.New(http.StatusNotFound, apperror.CodeNotFound, "subcon inventory not found")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Movement History
// ---------------------------------------------------------------------------

func (r *repo) GetMovementHistory(ctx context.Context, category, uniqCode string, f ListFilter) ([]HistoryRow, int64, error) {
	q := r.db.WithContext(ctx).Table("inventory_movement_logs").
		Where("movement_category = ? AND uniq_code = ?", category, uniqCode)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("GetMovementHistory count: %w", err)
	}

	var rows []HistoryRow
	err := q.Select(`
		id, uniq_code, qty_change, weight_change, movement_type, source_flag, dn_number, notes, logged_by, logged_at,
		CASE WHEN source_flag = 'incoming_scan' THEN 'incoming' ELSE 'confirmed' END AS log_status
	`).
		Order("logged_at DESC").
		Limit(f.Limit).Offset(f.Offset).
		Scan(&rows).Error
	return rows, total, err
}

// ---------------------------------------------------------------------------
// Incoming Scans (tab view)
// ---------------------------------------------------------------------------

func (r *repo) ListIncoming(ctx context.Context, f IncomingListFilter) ([]IncomingRow, int64, error) {
	q := r.db.WithContext(ctx).
		Table("incoming_receiving_scans irs").
		Joins("JOIN delivery_note_items dni ON dni.id = irs.incoming_dn_item_id").
		Joins("JOIN delivery_notes dn ON dn.id = dni.dn_id").
		Joins("LEFT JOIN suppliers s ON s.id = dn.supplier_id").
		Joins(`LEFT JOIN LATERAL (
			SELECT id, status FROM qc_tasks
			WHERE task_type = 'incoming_qc' AND incoming_dn_item_id = dni.id
			ORDER BY id DESC LIMIT 1
		) qt ON TRUE`).
		Joins("LEFT JOIN raw_materials rm ON rm.uniq_code = dni.item_uniq_code AND rm.deleted_at IS NULL")

	if f.DNType != "" {
		q = q.Where("UPPER(TRIM(dn.type)) IN ?", dnTypeVariants(f.DNType))
	}
	if f.PONumber != "" {
		q = q.Where("dn.po_number = ?", f.PONumber)
	}
	if f.SupplierID > 0 {
		q = q.Where("dn.supplier_id = ?", f.SupplierID)
	}
	// Incoming tab always excludes approved scans — those are already posted to inventory
	q = q.Where("COALESCE(qt.status, 'pending') != 'approved'")
	if f.Status != "" {
		q = q.Where("COALESCE(qt.status, 'pending') = ?", f.Status)
	}
	if f.Search != "" {
		s := "%" + f.Search + "%"
		q = q.Where("(dni.item_uniq_code ILIKE ? OR dn.po_number ILIKE ? OR dn.dn_number ILIKE ?)", s, s, s)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("ListIncoming count: %w", err)
	}

	var rows []IncomingRow
	err := q.Select(`
		irs.id                                    AS scan_id,
		dni.item_uniq_code,
		irs.qty                                   AS incoming_qty,
		rm.warehouse_location                     AS warehouse,
		irs.scanned_at                            AS scan_date,
		s.supplier_name,
		dn.po_number,
		dn.dn_number,
		COALESCE(qt.status, 'pending')            AS qc_status,
		dni.uom,
		dn.type                                   AS dn_type
	`).
		Order("irs.scanned_at DESC").
		Limit(f.Limit).Offset(f.Offset).
		Scan(&rows).Error
	return rows, total, err
}

// ---------------------------------------------------------------------------
// Recalculate helpers (used on update)
// ---------------------------------------------------------------------------

func recalculateRMStatus(tx *gorm.DB, id int64) error {
	var rm invModels.RawMaterial
	if err := tx.First(&rm, id).Error; err != nil {
		return err
	}

	safetyStock := 10.0
	if rm.SafetyStockQty != nil && *rm.SafetyStockQty > 0 {
		safetyStock = *rm.SafetyStockQty
	}
	dailyUsage := 1.0
	if rm.DailyUsageQty != nil && *rm.DailyUsageQty > 0 {
		dailyUsage = *rm.DailyUsageQty
	}

	stockDays := int(rm.StockQty / dailyUsage)
	status := computeStatus(rm.StockQty, safetyStock)
	buyNotBuy := "not_buy"
	if strings.EqualFold(rm.RawMaterialType, "ssp") {
		buyNotBuy = "n/a"
	} else if rm.StockQty < safetyStock {
		buyNotBuy = "buy"
	}

	return tx.Model(&rm).Updates(map[string]interface{}{
		"status":      status,
		"stock_days":  stockDays,
		"buy_not_buy": buyNotBuy,
		"updated_at":  time.Now(),
	}).Error
}

func recalculateIndirectStatus(tx *gorm.DB, id int64) error {
	var irm invModels.IndirectRawMaterial
	if err := tx.First(&irm, id).Error; err != nil {
		return err
	}

	safetyStock := 10.0
	if irm.SafetyStockQty != nil && *irm.SafetyStockQty > 0 {
		safetyStock = *irm.SafetyStockQty
	}
	dailyUsage := 1.0
	if irm.DailyUsageQty != nil && *irm.DailyUsageQty > 0 {
		dailyUsage = *irm.DailyUsageQty
	}

	stockDays := int(irm.StockQty / dailyUsage)
	status := computeStatus(irm.StockQty, safetyStock)
	buyNotBuy := "not_buy"
	if irm.StockQty < safetyStock {
		buyNotBuy = "buy"
	}
	statusPtr := status

	return tx.Model(&irm).Updates(map[string]interface{}{
		"status":      statusPtr,
		"stock_days":  stockDays,
		"buy_not_buy": buyNotBuy,
		"updated_at":  time.Now(),
	}).Error
}

func computeStatus(stock, safetyStock float64) string {
	if stock < safetyStock {
		return "low_on_stock"
	}
	if stock > safetyStock*2 {
		return "overstock"
	}
	return "normal"
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func safeOrderDir(tableAlias, col, dir string, allowed []string) string {
	allowedMap := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		allowedMap[a] = true
	}
	safeCol := tableAlias + ".created_at"
	if allowedMap[col] {
		safeCol = tableAlias + "." + col
	}
	safeDir := "DESC"
	if strings.EqualFold(dir, "asc") {
		safeDir = "ASC"
	}
	return safeCol + " " + safeDir
}

func dnTypeVariants(t string) []string {
	switch strings.ToUpper(strings.TrimSpace(t)) {
	case "RM", "RAW MATERIAL":
		return []string{"RM", "RAW MATERIAL"}
	case "IB", "INDIRECT", "INDIRECT RAW MATERIAL":
		return []string{"IB", "INDIRECT", "INDIRECT RAW MATERIAL"}
	case "SC", "SUBCON", "SUBCON MATERIAL", "SUB CON", "SUB-CON":
		return []string{"SC", "SUBCON", "SUBCON MATERIAL", "SUB CON", "SUB-CON", "SUBCON RAW MATERIAL"}
	default:
		return []string{t}
	}
}

func strPtr(s string) *string { return &s }
