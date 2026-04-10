package models

type CreatePOSplitRequest struct {
	BudgetType    string `json:"budget_type" binding:"required"`
	MinOrderQty   int    `json:"min_order_qty" binding:"required"`
	MaxSplitLines int    `json:"max_split_lines" binding:"required"`
	SplitRule     string `json:"split_rule" binding:"required"`
}

type UpdatePOSplitRequest struct {
	MinOrderQty   int    `json:"min_order_qty"`
	MaxSplitLines int    `json:"max_split_lines"`
	SplitRule     string `json:"split_rule"`
}
