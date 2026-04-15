// Package models defines domain structs for the Raw Material Inventory module.
package models

import (
	"time"

	"github.com/google/uuid"
)

// RawMaterial tracks on-hand inventory for raw materials from suppliers.
// Includes status calculation (Low/Normal/Overstock) and buy/not buy recommendation.
type RawMaterial struct {
	ID       int64     `gorm:"primaryKey;autoIncrement"`
	UUID     uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	UniqCode string    `gorm:"uniqueIndex;not null;size:64"`

	// Item details (auto-filled from scan)
	PartNumber *string `gorm:"size:128"`
	PartName   *string `gorm:"size:255"`
	ItemID     *int64  `gorm:"index"` // reference to items table

	// Type & source
	RawMaterialType string `gorm:"size:32"` // sheet_plate | wire | ssp | others
	RMSource        string `gorm:"size:32"` // process | supplier

	// Warehouse & storage
	WarehouseLocation *string `gorm:"size:255"`
	UOM               *string `gorm:"size:32"`

	// Stock tracking (updated by QC approval or manual decrease)
	StockQty      float64  `gorm:"type:numeric(15,4);default:0"`
	StockWeightKg *float64 `gorm:"type:numeric(15,4)"`
	KanbanCount   *int

	// Reference parameters (from parameter menu)
	KanbanStandardQty *int
	SafetyStockQty    *float64 `gorm:"type:numeric(15,4)"`
	DailyUsageQty     *float64 `gorm:"type:numeric(15,4)"`

	// Derived fields (calculated & stored for quick access)
	Status                string   `gorm:"size:32"` // low_on_stock | normal | overstock
	StockDays             *int     // computed: stock_qty / max(1, daily_usage_qty)
	BuyNotBuy             string   `gorm:"size:10"` // buy | not_buy | n/a
	StockToCompleteKanban *float64 `gorm:"type:numeric(15,4)"`

	// Audit
	CreatedBy *string    `gorm:"size:255"`
	CreatedAt time.Time  `gorm:"not null;default:now()"`
	UpdatedBy *string    `gorm:"size:255"`
	UpdatedAt time.Time  `gorm:"not null;default:now()"`
	DeletedAt *time.Time `gorm:"index"`
}

func (RawMaterial) TableName() string { return "raw_materials" }

// IndirectRawMaterial tracks inventory for indirect raw materials (MRO, consumables).
// Similar structure to RawMaterial but for non-direct materials.
type IndirectRawMaterial struct {
	ID       int64     `gorm:"primaryKey;autoIncrement"`
	UUID     uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	UniqCode string    `gorm:"uniqueIndex;not null;size:64"`

	// Item details
	PartNumber *string `gorm:"size:128"`
	PartName   *string `gorm:"size:255"`
	ItemID     *int64  `gorm:"index"`

	// Warehouse & storage
	WarehouseLocation *string `gorm:"size:255"`
	UOM               *string `gorm:"size:32"`

	// Stock tracking
	StockQty      float64  `gorm:"type:numeric(15,4);default:0"`
	StockWeightKg *float64 `gorm:"type:numeric(15,4)"`
	KanbanCount   *int

	// Reference parameters
	KanbanStandardQty *int
	SafetyStockQty    *float64 `gorm:"type:numeric(15,4)"`
	DailyUsageQty     *float64 `gorm:"type:numeric(15,4)"`

	// Derived fields
	Status                *string `gorm:"size:32"`
	StockDays             *int
	BuyNotBuy             string   `gorm:"size:10"`
	StockToCompleteKanban *float64 `gorm:"type:numeric(15,4)"`

	// Audit
	CreatedBy *string    `gorm:"size:255"`
	CreatedAt time.Time  `gorm:"not null;default:now()"`
	UpdatedBy *string    `gorm:"size:255"`
	UpdatedAt time.Time  `gorm:"not null;default:now()"`
	DeletedAt *time.Time `gorm:"index"`
}

func (IndirectRawMaterial) TableName() string { return "indirect_raw_materials" }

// SubconInventory tracks inventory at subcon vendors.
// Links to PO and tracks qty at vendor vs overall PO.
type SubconInventory struct {
	ID       int64     `gorm:"primaryKey;autoIncrement"`
	UUID     uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	UniqCode string    `gorm:"uniqueIndex;not null;size:64"`

	// Item details
	PartNumber *string `gorm:"size:128"`
	PartName   *string `gorm:"size:255"`

	// PO & vendor info
	PONumber         *string `gorm:"size:128;index"`
	POPeriod         *string `gorm:"size:32"`
	SubconVendorID   *int64  `gorm:"index"`
	SubconVendorName *string `gorm:"size:255"`

	// Stock tracking at vendor
	StockAtVendorQty float64  `gorm:"type:numeric(15,4);default:0"`
	TotalPOQty       *float64 `gorm:"type:numeric(15,4)"`
	TotalReceivedQty *float64 `gorm:"type:numeric(15,4)"`
	DeltaPO          *float64 `gorm:"type:numeric(15,4)"` // PO qty - stock (logic)

	// Reference
	SafetyStockQty *float64 `gorm:"type:numeric(15,4)"`
	DateDelivery   *time.Time

	// Status
	Status string `gorm:"size:32"` // low_on_stock | normal | overstock | above_po

	// Audit
	CreatedBy *string    `gorm:"size:255"`
	CreatedAt time.Time  `gorm:"not null;default:now()"`
	UpdatedBy *string    `gorm:"size:255"`
	UpdatedAt time.Time  `gorm:"not null;default:now()"`
	DeletedAt *time.Time `gorm:"index"`
}

func (SubconInventory) TableName() string { return "subcon_inventories" }

// InventoryMovementLog is a unified append-only log for all inventory movements.
// Use flags to distinguish source/category instead of joining multiple log tables.
type InventoryMovementLog struct {
	ID int64 `gorm:"primaryKey;autoIncrement"`

	// MovementCategory: raw_material | indirect_raw_material | subcon
	MovementCategory string `gorm:"size:64;index;not null"`
	// MovementType: incoming | outgoing | stock_opname | adjustment | received_from_vendor
	MovementType string `gorm:"size:64;index;not null"`

	// Item identity
	UniqCode string `gorm:"size:64;index;not null"`

	// EntityID keeps source record id (raw_materials.id, indirect_raw_materials.id, subcon_inventories.id)
	EntityID *int64 `gorm:"index"`

	QtyChange    float64  `gorm:"type:numeric(15,4)"`
	WeightChange *float64 `gorm:"type:numeric(15,4)"`

	// Additional flags/context for quick filtering in one table
	SourceFlag  *string `gorm:"size:64;index"` // manual | incoming_scan | qc_approve | wo_approve | production_reject | stock_opname (plus legacy)
	DNNumber    *string `gorm:"size:128"`
	ReferenceID *string `gorm:"size:255"`
	Notes       *string `gorm:"type:text"`

	LoggedBy *string   `gorm:"size:255"`
	LoggedAt time.Time `gorm:"not null;default:now();index"`
}

func (InventoryMovementLog) TableName() string { return "inventory_movement_logs" }

// RawMaterialLog tracks all increases/decreases to raw material stock (append-only audit log).
type RawMaterialLog struct {
	ID              int64     `gorm:"primaryKey;autoIncrement"`
	RawMaterialID   int64     `gorm:"not null;index"`
	TransactionType string    `gorm:"size:32"` // incoming | outgoing | stock_opname | adjustment
	QtyChange       float64   `gorm:"type:numeric(15,4)"`
	WeightChange    *float64  `gorm:"type:numeric(15,4)"`
	ReferenceID     *string   `gorm:"size:255"` // DN number, WO number, etc
	Notes           *string   `gorm:"type:text"`
	LoggedBy        *string   `gorm:"size:255"`
	LoggedAt        time.Time `gorm:"not null;default:now()"`
}

func (RawMaterialLog) TableName() string { return "raw_material_logs" }

// IndirectRawMaterialLog similar to RawMaterialLog for indirect materials.
type IndirectRawMaterialLog struct {
	ID                    int64     `gorm:"primaryKey;autoIncrement"`
	IndirectRawMaterialID int64     `gorm:"not null;index"`
	TransactionType       string    `gorm:"size:32"`
	QtyChange             float64   `gorm:"type:numeric(15,4)"`
	WeightChange          *float64  `gorm:"type:numeric(15,4)"`
	ReferenceID           *string   `gorm:"size:255"`
	Notes                 *string   `gorm:"type:text"`
	LoggedBy              *string   `gorm:"size:255"`
	LoggedAt              time.Time `gorm:"not null;default:now()"`
}

func (IndirectRawMaterialLog) TableName() string { return "indirect_raw_material_logs" }

// SubconInventoryLog tracks all transactions in subcon inventory.
type SubconInventoryLog struct {
	ID                int64     `gorm:"primaryKey;autoIncrement"`
	SubconInventoryID int64     `gorm:"not null;index"`
	TransactionType   string    `gorm:"size:32"` // incoming | outgoing | received_from_vendor | stock_opname
	QtyChange         float64   `gorm:"type:numeric(15,4)"`
	DNNumber          *string   `gorm:"size:128"`
	ReferenceID       *string   `gorm:"size:255"`
	Notes             *string   `gorm:"type:text"`
	LoggedBy          *string   `gorm:"size:255"`
	LoggedAt          time.Time `gorm:"not null;default:now()"`
}

func (SubconInventoryLog) TableName() string { return "subcon_inventory_logs" }
