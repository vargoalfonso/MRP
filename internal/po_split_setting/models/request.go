package models

type CreatePOSplitRequest struct {
	ItemUniqCode  string `json:"item_uniq_code" binding:"required"`
	BudgetType    string `json:"budget_type" binding:"required"`
	MinOrderQty   int    `json:"min_order_qty" binding:"required"`
	MaxSplitLines int    `json:"max_split_lines" binding:"required"`
	SplitRule     string `json:"split_rule" binding:"required"`
	Type          string `json:"type" binding:"required"`
	Status        string `json:"status" binding:"required"`
}

type UpdatePOSplitRequest struct {
	ItemUniqCode  string `json:"item_uniq_code"`
	BudgetType    string `json:"budget_type"`
	MinOrderQty   int    `json:"min_order_qty"`
	MaxSplitLines int    `json:"max_split_lines"`
	SplitRule     string `json:"split_rule"`
	Type          string `json:"type"`
	Status        string `json:"status"`
}
