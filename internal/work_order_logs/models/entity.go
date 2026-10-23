package models

import "time"

// WorkOrderAutomationLog menyimpan baris yang gagal masuk / gagal terkirim dari
// integrasi Raigine (robot automation) untuk Work Order.
type WorkOrderAutomationLog struct {
	ID                  int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID                string    `gorm:"uniqueIndex;not null" json:"uuid"`
	WoNumber            string    `gorm:"column:wo_number;size:64;index" json:"wo_number"`
	AutomationJobID     string    `gorm:"column:automation_job_id;size:128;index" json:"automation_job_id"`
	RobotName           string    `gorm:"column:robot_name;size:255" json:"robot_name"`
	RowNo               int       `gorm:"not null;default:0" json:"row_no"`
	ItemUniqCode        string    `gorm:"size:100" json:"item_uniq_code"`
	PartName            string    `gorm:"size:255" json:"part_name"`
	Description         string    `gorm:"type:text" json:"description"`
	Qty                 float64   `gorm:"type:numeric(15,4);default:0" json:"qty"`
	FailureReason       string    `gorm:"type:text" json:"failure_reason"`
	SpecialInstructions string    `gorm:"type:text" json:"special_instructions"`
	Source              string    `gorm:"size:64;default:automation" json:"source"`
	Status              string    `gorm:"size:32;default:failed" json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (WorkOrderAutomationLog) TableName() string {
	return "work_order_automation_logs"
}

type ListLogFilters struct {
	Search          string
	WoNumber        string
	AutomationJobID string
	Limit           int
	Offset          int
}
