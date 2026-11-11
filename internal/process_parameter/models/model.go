package models

import (
	"time"
)

type ProcessParameter struct {
	ID          int64     `gorm:"primaryKey"                       json:"id"`
	ProcessCode string    `gorm:"column:process_code"              json:"process_code"`
	ProcessName string    `gorm:"column:process_name"              json:"process_name"`
	Category    string    `gorm:"column:category"                  json:"category"`
	Sequence    int       `gorm:"column:sequence"                  json:"sequence"`
	Status      string    `gorm:"column:status"                    json:"status"`
	Subcon      bool      `gorm:"column:sub_con;default:false"     json:"sub_con"`
	IsAssembly  bool      `gorm:"column:is_assembly;default:false" json:"is_assembly"`
	CreatedAt   time.Time `gorm:"column:created_at"                json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"                json:"updated_at"`
}

func (ProcessParameter) TableName() string {
	return "process_parameters"
}
