package models

import (
	"time"
)

type StockdaysParameter struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	InventoryType   string    `json:"inventory_type"`
	ItemUniqCode    string    `json:"item_uniq_code"`
	CalculationType string    `json:"calculation_type"`
	Constanta       float64   `json:"constanta"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
