package models

import (
	"time"
)

type KanbanParameter struct {
	ID           int64     `json:"id" gorm:"primaryKey"`
	ItemUniqCode string    `json:"item_uniq_code"`
	KanbanQty    int       `json:"kanban_qty"`
	MinStock     int       `json:"min_stock"`
	MaxStock     int       `json:"max_stock"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
