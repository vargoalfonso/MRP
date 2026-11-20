package models

import (
	"encoding/json"
	"time"

	"gorm.io/datatypes"
)

// QCTask maps to legacy qc_tasks.
// Note: this repository uses it primarily for incoming QC.
type QCTask struct {
	ID int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`

	TaskType string `gorm:"column:task_type;size:32;not null" json:"task_type"`
	Status   string `gorm:"column:status;size:32;not null" json:"status"`

	// IncomingDNItemID links to delivery_note_items.id (bigint).
	IncomingDNItemID *int64 `gorm:"column:incoming_dn_item_id" json:"incoming_dn_item_id"`
	WOID             *int64 `gorm:"column:wo_id" json:"wo_id"`
	WOItemID         *int64 `gorm:"column:wo_item_id" json:"wo_item_id"`
	SourceScanID     *int64 `gorm:"column:source_scan_id" json:"source_scan_id"`

	GoodQuantity  *int       `gorm:"column:good_quantity" json:"good_quantity"`
	NgQuantity    *int       `gorm:"column:ng_quantity" json:"ng_quantity"`
	ScrapQuantity *int       `gorm:"column:scrap_quantity" json:"scrap_quantity"`
	DateChecked   *time.Time `gorm:"column:date_checked;type:date" json:"date_checked"`

	Round        int            `gorm:"column:round;default:1" json:"round"`
	RoundResults datatypes.JSON `gorm:"column:round_results;type:jsonb" json:"round_results"`

	CreatedAt time.Time `gorm:"column:created_at;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now()" json:"updated_at"`
}

func (QCTask) TableName() string { return "qc_tasks" }

func EmptyJSONArray() datatypes.JSON {
	b, _ := json.Marshal([]interface{}{})
	return datatypes.JSON(b)
}
