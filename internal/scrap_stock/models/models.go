// Package models defines domain structs for the Scrap Stock module.
package models

import (
	"time"

	"github.com/google/uuid"
)

// ScrapType enumerates allowed scrap classifications (BRD).
const (
	ScrapTypeSettingMachine = "setting_machine_scrap"
	ScrapTypeProcess        = "process_scrap"
	ScrapTypeProductReturn  = "product_return_scrap"
)

// ValidScrapTypes is the set of accepted scrap_type values.
var ValidScrapTypes = map[string]struct{}{
	ScrapTypeSettingMachine: {},
	ScrapTypeProcess:        {},
	ScrapTypeProductReturn:  {},
}

// ReleaseType enumerates allowed scrap release methods (BRD).
const (
	ReleaseTypeSell = "Sell"
	ReleaseTypeDump = "Dump"
)

// ApprovalStatus enumerates release approval states.
// "Completed" = approved and stock has been deducted (matches UI label).
const (
	ApprovalStatusPending   = "Pending"
	ApprovalStatusCompleted = "Completed"
	ApprovalStatusRejected  = "Rejected"
)

// ScrapStockStatus enumerates scrap stock record states.
const (
	StockStatusActive   = "Active"
	StockStatusInactive = "Inactive"
)

// ---------------------------------------------------------------------------
// ScrapStock — balance record per scrap entry.
// ---------------------------------------------------------------------------

// ScrapStock tracks the current scrap balance for a given item entry.
type ScrapStock struct {
	ID   int64     `gorm:"primaryKey;autoIncrement"`
	UUID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`

	// Item identity
	UniqCode      string  `gorm:"not null;size:64;index"`
	PartNumber    *string `gorm:"size:128"`
	PartName      *string `gorm:"size:255"`
	Model         *string `gorm:"size:128"`
	PackingNumber *string `gorm:"size:128;index"`

	// Traceability
	WONumber *string `gorm:"size:128;index"` // reference to work order

	// Scrap classification (setting_machine_scrap | process_scrap | product_return_scrap)
	ScrapType      string  `gorm:"not null;size:64;index"`
	DisposalReason *string `gorm:"size:128"`

	// Stock balance
	Quantity float64  `gorm:"type:numeric(15,4);not null;default:0"`
	UOM      *string  `gorm:"size:32"`
	WeightKg *float64 `gorm:"type:numeric(15,4)"`

	// Meta
	DateReceived *time.Time
	Validator    *string `gorm:"size:255"` // person who validated the entry
	Remarks      *string `gorm:"type:text"`

	// Record status: Active | Inactive
	Status string `gorm:"not null;size:32;default:'Active';index"`

	// Audit
	CreatedBy *string    `gorm:"size:255"`
	CreatedAt time.Time  `gorm:"not null;default:now()"`
	UpdatedBy *string    `gorm:"size:255"`
	UpdatedAt time.Time  `gorm:"not null;default:now()"`
	DeletedAt *time.Time `gorm:"index"`
}

func (ScrapStock) TableName() string { return "scrap_stocks" }

// ---------------------------------------------------------------------------
// ScrapRelease — release event (Sell/Dump) with approval gate.
// ---------------------------------------------------------------------------

// ScrapRelease records a single release transaction from scrap stock.
// Stock is deducted atomically when ApprovalStatus transitions to "Completed".
type ScrapRelease struct {
	ID   int64     `gorm:"primaryKey;autoIncrement"`
	UUID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`

	// Human-readable release number: SR-YYYY-NNN
	ReleaseNumber string `gorm:"not null;size:64;uniqueIndex"`

	// Link back to the balance record
	ScrapStockID int64 `gorm:"not null;index"`

	// Release details
	ReleaseDate    *time.Time
	ReleaseType    string   `gorm:"not null;size:32"` // Sell | Dump
	ReleaseQty     float64  `gorm:"type:numeric(15,4);not null"`
	WeightReleased *float64 `gorm:"type:numeric(15,4)"`

	// Sell-specific
	CustomerName *string  `gorm:"size:255"`
	PricePerUnit *float64 `gorm:"type:numeric(15,4)"`
	TotalValue   *float64 `gorm:"type:numeric(15,4)"`

	// Dump-specific
	DisposalReason *string `gorm:"type:text"`

	// Approval gate: Pending | Completed | Rejected
	ApprovalStatus string     `gorm:"not null;size:32;default:'Pending';index"`
	Validator      *string    `gorm:"size:255"` // person who submitted the release
	Approver       *string    `gorm:"size:255"` // intended approver role/name
	ApprovedBy     *string    `gorm:"size:255"` // actual approver (JWT at approval time)
	ApprovedAt     *time.Time

	Remarks *string `gorm:"type:text"`

	// Audit
	CreatedBy *string    `gorm:"size:255"`
	CreatedAt time.Time  `gorm:"not null;default:now()"`
	UpdatedBy *string    `gorm:"size:255"`
	UpdatedAt time.Time  `gorm:"not null;default:now()"`
	DeletedAt *time.Time `gorm:"index"`
}

func (ScrapRelease) TableName() string { return "scrap_releases" }

// ---------------------------------------------------------------------------
// ScrapMovementRow — query result for history log (from inventory_movement_logs).
// ---------------------------------------------------------------------------

type ScrapMovementRow struct {
	ID          int64     `gorm:"column:id"`
	UniqCode    string    `gorm:"column:uniq_code"`
	PackingList *string   `gorm:"column:packing_list"`
	Delta       float64   `gorm:"column:qty_change"`
	SourceFlag  *string   `gorm:"column:source_flag"`
	ReferenceID *string   `gorm:"column:reference_id"`
	Notes       *string   `gorm:"column:notes"`
	LoggedBy    *string   `gorm:"column:logged_by"`
	LoggedAt    time.Time `gorm:"column:logged_at"`
}
