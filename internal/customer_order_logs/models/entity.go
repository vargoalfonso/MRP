package models

import "time"

// CustomerOrderAutomationLog stores rows that failed to enter / failed to be
// sent (etc.) from the Raigine automation integration for customer orders.
type CustomerOrderAutomationLog struct {
	ID                  int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID                string    `gorm:"uniqueIndex;not null" json:"uuid"`
	DocumentID          *int64    `gorm:"index" json:"-"`
	DocumentNumber      string    `gorm:"size:128" json:"document_number"`
	RowNo               int       `gorm:"not null;default:0" json:"row_no"`
	ItemUniqCode        string    `gorm:"size:100" json:"item_uniq_code"`
	PartName            string    `gorm:"size:255" json:"part_name"`
	Description         string    `gorm:"type:text" json:"description"`
	QtyActive           float64   `gorm:"type:numeric(15,4);default:0" json:"qty_active"`
	FailureReason       string    `gorm:"type:text" json:"failure_reason"`
	SpecialInstructions string    `gorm:"type:text" json:"special_instructions"`
	Source              string    `gorm:"size:64;default:automation" json:"source"`
	Status              string    `gorm:"size:32;default:failed" json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (CustomerOrderAutomationLog) TableName() string {
	return "customer_order_automation_logs"
}

type ListLogFilters struct {
	Search         string
	DocumentNumber string
	Limit          int
	Offset         int
}
