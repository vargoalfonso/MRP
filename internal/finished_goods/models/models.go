// Package models defines domain structs for the Finished Goods module.
package models

import (
	"time"

	"github.com/google/uuid"
)

// FGStatus enumerates finished goods stock status values.
const (
	FGStatusLowStock  = "low_on_stock"
	FGStatusNormal    = "normal"
	FGStatusOverstock = "overstock"
)

// ---------------------------------------------------------------------------
// FinishedGoods — on-hand balance record per Uniq.
// ---------------------------------------------------------------------------

// FinishedGoods tracks the current FG balance for a given Uniq code.
// Kanban threshold values are denormalized snapshots from kanban_parameters
// at create/update time so the list endpoint never needs a JOIN.
type FinishedGoods struct {
	ID   int64     `gorm:"primaryKey;autoIncrement"`
	UUID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`

	// Item identity (auto-resolved from bom_item via uniq_code)
	UniqCode   string  `gorm:"not null;size:64;uniqueIndex"`
	ItemID     *int64  `gorm:"index"` // optional FK to bom items table
	PartNumber *string `gorm:"size:128"`
	PartName   *string `gorm:"size:255"`
	Model      *string `gorm:"size:128"`

	// Traceability
	WONumber          *string `gorm:"size:128"` // last WO that produced this FG
	WarehouseLocation *string `gorm:"size:255"` // destination warehouse

	// Stock
	StockQty float64 `gorm:"type:numeric(15,4);not null;default:0"`
	UOM      *string `gorm:"size:32"`

	// Kanban snapshot (from kanban_parameters)
	KanbanCount           *int     // computed: floor(stock_qty / kanban_standard_qty)
	KanbanStandardQty     *int     // mirror of kanban_parameters.kanban_qty
	MinThreshold          *float64 `gorm:"type:numeric(15,4)"` // kanban_parameters.min_stock
	MaxThreshold          *float64 `gorm:"type:numeric(15,4)"` // kanban_parameters.max_stock
	SafetyStockQty        *float64 `gorm:"type:numeric(15,4)"` // target qty shown as "Target" in UI
	StockToCompleteKanban *float64 `gorm:"type:numeric(15,4)"` // max(0, safety_stock_qty - stock_qty)

	// Status: recomputed on every stock change
	// stock_qty < min_threshold → low_on_stock
	// stock_qty > max_threshold → overstock
	// else                      → normal
	Status string `gorm:"not null;size:32;default:'normal';index"`

	// Audit
	CreatedBy *string    `gorm:"size:255"`
	CreatedAt time.Time  `gorm:"not null;default:now()"`
	UpdatedBy *string    `gorm:"size:255"`
	UpdatedAt time.Time  `gorm:"not null;default:now()"`
	DeletedAt *time.Time `gorm:"index"`
}

func (FinishedGoods) TableName() string { return "finished_goods" }

// ---------------------------------------------------------------------------
// FGMovementLog — append-only ledger for every stock change.
// ---------------------------------------------------------------------------

// FGMovementLog records every individual stock change for audit purposes.
// Never updated or deleted.
type FGMovementLog struct {
	ID int64 `gorm:"primaryKey;autoIncrement"`

	FgID     int64  `gorm:"not null;index"`
	UniqCode string `gorm:"not null;size:64;index"`

	// movement_type: incoming_production | delivery_scan | manual_add |
	//                manual_deduct | stock_opname | wo_complete | delete
	MovementType string  `gorm:"not null;size:64;index"`
	QtyChange    float64 `gorm:"type:numeric(15,4);not null"` // positive = in, negative = out
	QtyBefore    float64 `gorm:"type:numeric(15,4);not null"`
	QtyAfter     float64 `gorm:"type:numeric(15,4);not null"`

	// source_flag: action_ui | manual | stock_opname | delivery
	SourceFlag  *string `gorm:"size:64"`
	WONumber    *string `gorm:"size:128"`
	DNNumber    *string `gorm:"size:128"`
	ReferenceID *string `gorm:"size:255"`
	Notes       *string `gorm:"type:text"`

	LoggedBy *string   `gorm:"size:255"`
	LoggedAt time.Time `gorm:"not null;default:now();index"`
}

func (FGMovementLog) TableName() string { return "fg_movement_logs" }
