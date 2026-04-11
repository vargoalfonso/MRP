// Package models defines domain structs for the Procurement module.
package models

import "time"

// ---------------------------------------------------------------------------
// DB-mapped structs (GORM)
// ---------------------------------------------------------------------------

// PurchaseOrder maps to the legacy `purchase_orders` table.
// supplier_id stays BIGINT (legacy `supplier` table).
// po_budget_id stays BIGINT (legacy `po_budget` table).
// po_budget_entry_id (new) links to `po_budget_entries` (v2 budget system).
type PurchaseOrder struct {
	PoID             int64      `gorm:"column:po_id;primaryKey;autoIncrement"`
	PoType           string     `gorm:"column:po_type;size:32;not null"` // RM | INDIRECT | SUBCON
	Period           string     `gorm:"column:period;size:32;not null"`
	PoNumber         string     `gorm:"column:po_number;size:128;uniqueIndex"`
	PoStage          *int       `gorm:"column:po_stage"`           // 1=PO1, 2=PO2; nil=no split
	PoBudgetID       *int64     `gorm:"column:po_budget_id"`       // legacy po_budget.pob_id
	PoBudgetEntryID  *int64     `gorm:"column:po_budget_entry_id"` // po_budget_entries.id (v2)
	SupplierID       *int64     `gorm:"column:supplier_id"`        // legacy supplier.supplier_id
	Status           string     `gorm:"column:status;size:32;default:pending"`
	PoDate           *time.Time `gorm:"column:po_date;type:date"`
	ExpectedDelivery *time.Time `gorm:"column:expected_delivery_date;type:date"`
	Currency         *string    `gorm:"column:currency;size:8"`
	TotalWeight      *float64   `gorm:"column:total_weight;type:numeric(15,4)"`
	TotalAmount      *float64   `gorm:"column:total_amount;type:numeric(18,2)"`
	ExternalSystem   *string    `gorm:"column:external_system;size:64"`
	ExternalPoNumber *string    `gorm:"column:external_po_number;size:128"`
	CreatedBy        *string    `gorm:"column:created_by;size:255"`
	UpdatedBy        *string    `gorm:"column:updated_by;size:255"`
	CreatedAt        time.Time  `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;not null;default:now()"`
}

func (PurchaseOrder) TableName() string { return "purchase_orders" }

// PurchaseOrderItem maps to `purchase_order_items`.
type PurchaseOrderItem struct {
	ID           int64    `gorm:"column:id;primaryKey;autoIncrement"`
	PoID         int64    `gorm:"column:po_id;not null"`
	LineNo       int      `gorm:"column:line_no;default:1"`
	ItemUniqCode string   `gorm:"column:item_uniq_code;size:64;not null"`
	ProductModel *string  `gorm:"column:product_model;size:255"`
	MaterialType *string  `gorm:"column:material_type;size:64"`
	PartName     *string  `gorm:"column:part_name;size:255"`
	PartNumber   *string  `gorm:"column:part_number;size:128"`
	Uom          *string  `gorm:"column:uom;size:32"`
	WeightKg     *float64 `gorm:"column:weight_kg;type:numeric(15,4)"`
	Description  *string  `gorm:"column:description;type:text"`
	OrderedQty   float64  `gorm:"column:ordered_qty;type:numeric(15,4);not null"`
	UnitPrice    *float64 `gorm:"column:unit_price;type:numeric(18,6)"`
	// Amount is GENERATED ALWAYS AS (unit_price * ordered_qty) STORED — read-only.
	Amount          *float64 `gorm:"column:amount;type:numeric(18,2);->"`
	PcsPerKanban    *int     `gorm:"column:pcs_per_kanban"`
	PoBudgetEntryID *int64   `gorm:"column:po_budget_entry_id"` // trace line → budget entry
	Status          string   `gorm:"column:status;size:32;default:open"`
	// SalesPlan is transient (not persisted) — carried from budget entry for response building.
	SalesPlan float64   `gorm:"-" json:"-"`
	CreatedAt time.Time `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now()"`
}

func (PurchaseOrderItem) TableName() string { return "purchase_order_items" }

// PurchaseOrderLog is the append-only audit log per PO (purchase_order_logs).
type PurchaseOrderLog struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	PoID       int64     `gorm:"column:po_id;not null"`
	Action     string    `gorm:"column:action;size:64;not null"`
	Notes      *string   `gorm:"column:notes;type:text"`
	Username   *string   `gorm:"column:username;size:255"`
	OccurredAt time.Time `gorm:"column:occurred_at;not null;default:now()"`
}

func (PurchaseOrderLog) TableName() string { return "purchase_order_logs" }

// LegacySupplier maps to the legacy `supplier` table (bigint PK).
type LegacySupplier struct {
	SupplierID   int64  `gorm:"column:supplier_id;primaryKey"`
	SupplierName string `gorm:"column:supplier_name;size:128"`
}

func (LegacySupplier) TableName() string { return "suppliers" }

// IncomingDN maps to delivery_notes table.
type IncomingDN struct {
	ID              int64     `gorm:"column:id;primaryKey"`
	DnNumber        string    `gorm:"column:dn_number;size:128;not null"`
	Period          string    `gorm:"column:period;size:32;not null"`
	PoNumber        string    `gorm:"column:po_number;size:128;not null"`
	DnType          string    `gorm:"column:type;size:64;not null"`
	TotalPoQty      *int      `gorm:"column:total_po_qty"`
	TotalDnCreated  *int      `gorm:"column:total_dn_created"`
	TotalDnIncoming *int      `gorm:"column:total_dn_incoming"`
	SupplierID      *int64    `gorm:"column:supplier_id"`
	Status          string    `gorm:"column:status;size:64;default:Pending Receipt"`
	CreatedAt       time.Time `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt       time.Time `gorm:"column:updated_at;not null;default:now()"`
}

func (IncomingDN) TableName() string { return "delivery_notes" }

// IncomingDNItem maps to delivery_note_items table.
type IncomingDNItem struct {
	ID             int64      `gorm:"column:id;primaryKey"`
	IncomingDNID   int64      `gorm:"column:dn_id;not null"`
	ItemUniqCode   string     `gorm:"column:item_uniq_code;size:64;not null"`
	Quantity       int        `gorm:"column:quantity"`
	OrderQty       int        `gorm:"column:order_qty;not null"`
	Weight         *int       `gorm:"column:weight"`
	DateIncoming   time.Time  `gorm:"column:date_incoming;type:date;not null"`
	QtyStated      int        `gorm:"column:qty_stated;default:0"`
	QtyReceived    int        `gorm:"column:qty_received;default:0"`
	WeightReceived *float64   `gorm:"column:weight_received"`
	QualityStatus  string     `gorm:"column:quality_status;size:32;default:Pending"`
	PackingNumber  *string    `gorm:"-"` // not a DB column; populated in application layer from dn.dn_number
	PcsPerKanban   *int       `gorm:"column:pcs_per_kanban"`
	Uom            *string    `gorm:"column:uom;size:32"`
	ReceivedAt     *time.Time `gorm:"column:received_at"`
	CreatedAt      time.Time  `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;not null;default:now()"`
}

func (IncomingDNItem) TableName() string { return "delivery_note_items" }

// POBudgetEntry is a read-only reference from po_budget_entries (for form_options / generate).
// We only read from it here; writes happen in the po_budget module.
type POBudgetEntry struct {
	ID              int64    `gorm:"column:id;primaryKey"`
	BudgetType      string   `gorm:"column:budget_type"` // raw_material | indirect | subcon
	UniqCode        string   `gorm:"column:uniq_code"`
	ProductModel    *string  `gorm:"column:product_model"`
	MaterialType    *string  `gorm:"column:material_type"`
	PartName        *string  `gorm:"column:part_name"`
	PartNumber      *string  `gorm:"column:part_number"`
	Quantity        float64  `gorm:"column:quantity"`
	Uom             *string  `gorm:"column:uom"`
	WeightKg        *float64 `gorm:"column:weight_kg"`
	SupplierID      *int64   `gorm:"column:supplier_id"` // legacy supplier.supplier_id (BIGINT)
	SupplierName    *string  `gorm:"column:supplier_name"`
	Period          string   `gorm:"column:period"`
	SalesPlan       float64  `gorm:"column:sales_plan"`
	PurchaseRequest float64  `gorm:"column:purchase_request"`
	Po1Pct          float64  `gorm:"column:po1_pct"`
	Po2Pct          float64  `gorm:"column:po2_pct"`
	Po1Qty          float64  `gorm:"column:po1_qty"`
	Po2Qty          float64  `gorm:"column:po2_qty"`
	PcsPerKanban    *int     `gorm:"column:pcs_per_kanban"`
	KanbanNumber    *string  `gorm:"column:kanban_number"`
	Status          string   `gorm:"column:status"`
}

func (POBudgetEntry) TableName() string { return "po_budget_entries" }

// SupplierLegacyMap bridges new UUID supplier_id → legacy BIGINT supplier_id.
type SupplierLegacyMap struct {
	LegacySupplierID int64  `gorm:"column:legacy_supplier_id;primaryKey"`
	SupplierUUID     string `gorm:"column:supplier_uuid"`
}

func (SupplierLegacyMap) TableName() string { return "supplier_legacy_map" }

// ---------------------------------------------------------------------------
// Computed / aggregate-only structs (not DB tables)
// ---------------------------------------------------------------------------

// POSummaryRow is returned by the summary aggregate query.
type POSummaryRow struct {
	TotalPos        int64   `gorm:"column:total_pos"`
	ActiveSuppliers int64   `gorm:"column:active_suppliers"`
	TotalPoValue    float64 `gorm:"column:total_po_value"`
	LateDeliveries  int64   `gorm:"column:late_deliveries"`
}

// POBoardRow is one row in the PO board list (join result).
type POBoardRow struct {
	PoID          int64    `gorm:"column:po_id"`
	PoType        string   `gorm:"column:po_type"`
	PoStage       *int     `gorm:"column:po_stage"`
	Period        string   `gorm:"column:period"`
	PoNumber      string   `gorm:"column:po_number"`
	TotalBudgetPo float64  `gorm:"column:total_budget_po"`
	QtyDelivered  float64  `gorm:"column:qty_delivered"`
	UniqCode      *string  `gorm:"column:uniq_code"`
	SupplierID    *int64   `gorm:"column:supplier_id"`
	SupplierName  *string  `gorm:"column:supplier_name"`
	Status        string   `gorm:"column:status"`
	IsLate        bool     `gorm:"column:is_late"`
	TotalAmount   *float64 `gorm:"column:total_amount"`
}
