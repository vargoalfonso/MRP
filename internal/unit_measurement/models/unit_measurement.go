package models

import (
	"time"
)

type UomParameter struct {
	ID        int64 `gorm:"primaryKey;autoIncrement"`
	Code      string
	Name      string
	Category  string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (UomParameter) TableName() string { return "uom_parameters" }
