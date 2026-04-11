package models

import (
	"time"
)

type POSplitSetting struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	ItemUniqCode  string    `json:"item_uniq_code"`
	BudgetType    string    `json:"budget_type"`
	MinOrderQty   int       `json:"min_order_qty"`
	MaxSplitLines int       `json:"max_split_lines"`
	SplitRule     string    `json:"split_rule"`
	Type          string    `json:"type"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
