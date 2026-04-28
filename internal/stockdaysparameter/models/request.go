package models

type CreateStockdaysRequest struct {
	ItemUniqCode string  `json:"item_code" binding:"required"`
	StockDays    int     `json:"stock_days"`
	SafetyStock  int     `json:"safety_stock"`
	Status       *string `json:"status"`
}

type UpdateStockdaysRequest struct {
	StockDays   int     `json:"stock_days"`
	SafetyStock int     `json:"safety_stock"`
	Status      *string `json:"status"`
}

type BulkCreateStockdaysRequest struct {
	Items []CreateStockdaysRequest `json:"items"`
}
