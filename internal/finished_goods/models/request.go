package models

// CreateFinishedGoodsRequest is the body for POST /api/v1/finished-goods.
// Only 2 fields are required from the user; everything else is auto-resolved.
type CreateFinishedGoodsRequest struct {
	// Required
	UniqCode          string `json:"uniq_code" validate:"required"`
	WarehouseLocation string `json:"warehouse_location" validate:"required"`
	// Optional override — if set, skips auto-resolve from work_orders
	WONumberOverride *string `json:"wo_number"`
}

// BulkCreateFGItem is a single entry in the bulk create payload.
// Used by both Manual Input (multiple entries, no stock_qty) and Bulk Upload (Excel parsed by FE, stock_qty editable).
// wo_number: optional override; if omitted, auto-resolved from last work_order for that uniq_code.
// stock_qty: optional, default 0 (manual input); editable per row in bulk upload review step.
type BulkCreateFGItem struct {
	UniqCode          string   `json:"uniq_code" validate:"required"`
	WarehouseLocation string   `json:"warehouse_location" validate:"required"`
	WONumber          *string  `json:"wo_number"`
	StockQty          *float64 `json:"stock_qty" validate:"omitempty,gte=0"`
}

// BulkCreateFinishedGoodsRequest is the body for POST /api/v1/finished-goods/bulk.
type BulkCreateFinishedGoodsRequest struct {
	Items []BulkCreateFGItem `json:"items" validate:"required,min=1,dive"`
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
