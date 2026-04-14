package models

import (
	"time"

	"github.com/google/uuid"
)

// WorkOrder is the header record for work orders.
// Table: work_orders (legacy/existing).
type WorkOrder struct {
	ID int64 `gorm:"primaryKey;autoIncrement"`

	UUID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`

	WoNumber       string  `gorm:"column:wo_number;uniqueIndex;not null;size:64"`
	WoType         string  `gorm:"column:wo_type;not null;size:32"`
	ReferenceWO    *string `gorm:"column:reference_wo;size:64"`
	Status         string  `gorm:"column:status;not null;size:32"`
	ApprovalStatus string  `gorm:"column:approval_status;not null;size:32"`

	CreatedDate   time.Time  `gorm:"column:created_date;type:date;not null"`
	TargetDate    *time.Time `gorm:"column:target_date;type:date"`
	ScanStartDate *time.Time `gorm:"column:scan_start_date;type:date"`
	CloseDate     *time.Time `gorm:"column:close_date;type:date"`

	OperatorName *string `gorm:"column:operator_name;size:255"`
	// Creator audit (different from OperatorName).
	CreatedBy     *uuid.UUID `gorm:"column:created_by;type:uuid"`
	CreatedByName *string    `gorm:"column:created_by_name;size:255"`
	Notes         *string    `gorm:"column:notes"`
	QRImageBase64 *string    `gorm:"column:qr_image_base64"`

	CreatedAt time.Time `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now()"`
}

func (WorkOrder) TableName() string { return "work_orders" }

// WorkOrderItem is a single kanban/batch line under a work order.
// Table: work_order_items (legacy/existing).
type WorkOrderItem struct {
	ID int64 `gorm:"primaryKey;autoIncrement"`

	UUID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`

	WoID int64 `gorm:"column:wo_id;index;not null"`

	ItemUniqCode string  `gorm:"column:item_uniq_code;not null;size:64"`
	PartName     *string `gorm:"column:part_name;size:255"`
	PartNumber   *string `gorm:"column:part_number;size:128"`
	Model        *string `gorm:"column:model;size:128"`
	UOM          *string `gorm:"column:uom;size:32"`

	Quantity     float64 `gorm:"column:quantity;type:numeric(15,4);not null;default:0"`
	ProcessName  *string `gorm:"column:process_name;size:64"`
	KanbanNumber string  `gorm:"column:kanban_number;uniqueIndex;not null;size:150"`
	// Traceability back to kanban_parameters.kanban_number (e.g. KBN-2026-0003)
	KanbanParamNumber *string `gorm:"column:kanban_param_number;size:50"`
	KanbanSeq         *int    `gorm:"column:kanban_seq"`
	Status            string  `gorm:"column:status;not null;size:32"`
	QRImageBase64     *string `gorm:"column:qr_image_base64"`

	CreatedAt time.Time `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now()"`
}

func (WorkOrderItem) TableName() string { return "work_order_items" }
