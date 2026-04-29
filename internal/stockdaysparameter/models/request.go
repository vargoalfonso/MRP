package models

type CreateStockdaysRequest struct {
	IventoryType    string  `json:"inventory_type" binding:"required"`
	ItemCode        string  `json:"item_code" binding:"required"`
	CalculationType string  `json:"calculation_type"`
	Constanta       int     `json:"constanta"`
	Status          *string `json:"status"`
}

type UpdateStockdaysRequest struct {
	IventoryType    string  `json:"inventory_type"`
	ItemCode        string  `json:"item_code" binding:"required"`
	CalculationType string  `json:"calculation_type"`
	Constanta       int     `json:"constanta"`
	Status          *string `json:"status"`
}

type BulkCreateStockdaysRequest struct {
	Items []CreateStockdaysRequest `json:"items"`
}
