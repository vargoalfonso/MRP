package models

// CreateFinishedGoodsRequest is the body for POST /api/v1/finished-goods.
// Only 2 fields are required from the user; everything else is auto-resolved.
type CreateFinishedGoodsRequest struct {
	// Required
	UniqCode          string `json:"uniq_code" validate:"required"`
	WarehouseLocation string `json:"warehouse_location" validate:"required"`
}

// UpdateFinishedGoodsRequest is the body for PUT /api/v1/finished-goods/:id.
// Edit is single-item only (no bulk edit per Designer's Note).
// Only stock_qty is directly editable; kanban params re-synced from parameter table.
type UpdateFinishedGoodsRequest struct {
	WONumber          *string  `json:"wo_number"`
	WarehouseLocation *string  `json:"warehouse_location"`
	StockQty          *float64 `json:"stock_qty" validate:"omitempty,gte=0"`
	Remarks           *string  `json:"remarks"`
}
