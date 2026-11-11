package models

type CreatePOSplitRequest struct {
	BudgetType    *string  `json:"budget_type"`
	Po1Pct        *float64 `json:"po1_pct"`
	Po2Pct        *float64 `json:"po2_pct"`
	Description   *string  `json:"description"`
	MinOrderQty   *int     `json:"min_order_qty"`
	MaxSplitLines *int     `json:"max_split_lines"`
	SplitRule     *string  `json:"split_rule"`
	Status        *string  `json:"status"`
}

type UpdatePOSplitRequest struct {
	BudgetType    *string  `json:"budget_type"`
	Po1Pct        *float64 `json:"po1_pct"`
	Po2Pct        *float64 `json:"po2_pct"`
	Description   *string  `json:"description"`
	MinOrderQty   *int     `json:"min_order_qty"`
	MaxSplitLines *int     `json:"max_split_lines"`
	SplitRule     *string  `json:"split_rule"`
	Status        *string  `json:"status"`
}
