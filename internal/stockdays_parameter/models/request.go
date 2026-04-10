package models

type CreateStockdaysRequest struct {
	InventoryType   string  `json:"inventory_type" binding:"required"`
	ItemUniqCode    string  `json:"item_uniq_code" binding:"required"`
	CalculationType string  `json:"calculation_type" binding:"required"`
	Constanta       float64 `json:"constanta"`
}

type UpdateStockdaysRequest struct {
	CalculationType string  `json:"calculation_type"`
	Constanta       float64 `json:"constanta"`
}

type BulkCreateStockdaysRequest struct {
	Items []CreateStockdaysRequest `json:"items"`
}
