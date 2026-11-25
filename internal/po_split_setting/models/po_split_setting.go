package models

import (
	"time"
)

type POSplitSetting struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	BudgetType    string    `gorm:"type:varchar(50)" json:"budget_type"`
	Po1Pct        float64   `gorm:"type:decimal(5,2)" json:"po1_pct"`
	Po2Pct        float64   `gorm:"type:decimal(5,2)" json:"po2_pct"`
	Description   string    `gorm:"type:text" json:"description"`
	MinOrderQty   int       `json:"min_order_qty"`
	MaxSplitLines int       `json:"max_split_lines"`
	SplitRule     string    `gorm:"type:varchar(20)" json:"split_rule"`
	Status        string    `gorm:"type:varchar(20);default:active" json:"status"`
	PoSplitSUM    float64   `gorm:"type:decimal(5,2)" json:"po_split_sum"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
