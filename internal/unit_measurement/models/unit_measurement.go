package models

import (
	"time"
)

type UnitMeasurement struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
