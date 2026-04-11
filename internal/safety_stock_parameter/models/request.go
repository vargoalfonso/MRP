package models

type CreateSafetyStockRequest struct {
	InventoryType   string  `json:"inventory_type" binding:"required"`
	ItemUniqCode    string  `json:"item_uniq_code" binding:"required"`
	CalculationType string  `json:"calculation_type" binding:"required"`
	Constanta       float64 `json:"constanta"`
}

type BulkCreateSafetyStockRequest struct {
	Items []CreateSafetyStockRequest `json:"items"`
}

type UpdateSafetyStockRequest struct {
	CalculationType string  `json:"calculation_type"`
	Constanta       float64 `json:"constanta"`
}
