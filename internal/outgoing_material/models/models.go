package models

import (
	"time"

	"github.com/google/uuid"
)

// OutgoingRawMaterial is the business transaction record for each outgoing RM event.
// Source of truth for the outgoing list UI; inventory_movement_logs is the centralized audit trail.
type OutgoingRawMaterial struct {
	ID   int64     `gorm:"primaryKey;autoIncrement"`
	UUID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`

	TransactionID   string    `gorm:"uniqueIndex;not null;size:64"`
	TransactionDate time.Time `gorm:"type:date;not null"`

	RawMaterialID *int64  `gorm:"index"`
	PackingListRM *string `gorm:"size:128"`
	Uniq          string  `gorm:"not null;index;size:64"`
	RMName        *string `gorm:"size:255"`

	Unit        *string `gorm:"size:32"`
	QuantityOut float64 `gorm:"type:numeric(15,4);not null"`
	StockBefore float64 `gorm:"type:numeric(15,4);not null"`
	StockAfter  float64 `gorm:"type:numeric(15,4);not null"`

	Reason              string  `gorm:"not null;size:64"`
	Purpose             *string `gorm:"type:text"`
	WorkOrderNo         *string `gorm:"size:128"`
	DestinationLocation *string `gorm:"size:255"`
	RequestedBy         *string `gorm:"size:255"`
	Remarks             *string `gorm:"type:text"`

	// StockRestoredAt is set when a user manually returns this transaction's
	// quantity back into raw_materials stock via the "restore stock" action.
	// A non-nil value prevents a second (double) restore.
	StockRestoredAt *time.Time `gorm:"index"`

	CreatedBy *string    `gorm:"size:255"`
	CreatedAt time.Time  `gorm:"not null;default:now()"`
	UpdatedBy *string    `gorm:"size:255"`
	UpdatedAt time.Time  `gorm:"not null;default:now()"`
	DeletedAt *time.Time `gorm:"index"`
}

func (OutgoingRawMaterial) TableName() string { return "outgoing_raw_material" }
