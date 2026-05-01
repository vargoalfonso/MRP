package models

import (
	"time"
)

type ProcessParameter struct {
	ID          int64     `gorm:"primaryKey"`
	ProcessCode string    `gorm:"column:process_code"`
	ProcessName string    `gorm:"column:process_name"`
	Category    string    `gorm:"column:category"`
	Sequence    int       `gorm:"column:sequence"`
	Status      string    `gorm:"column:status"`
	Subcon      bool      `gorm:"column:sub_con;default:false"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (ProcessParameter) TableName() string {
	return "process_parameters"
}
