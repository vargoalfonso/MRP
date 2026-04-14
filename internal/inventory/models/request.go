package models

import "time"

// ---------------------------------------------------------------------------
// Raw Material
// ---------------------------------------------------------------------------

type CreateRawMaterialRequest struct {
	UniqCode string `json:"uniq_code" validate:"required"`
	// Optional: when raw_materials.uniq_code differs from items.uniq_code, provide this to auto-fill part fields.
	ItemUniqCode      *string  `json:"item_uniq_code"`
	RawMaterialType   string   `json:"raw_material_type"` // sheet_plate | wire | ssp | others
	RMSource          string   `json:"rm_source"`         // process | supplier
	PartNumber        *string  `json:"part_number"`
	PartName          *string  `json:"part_name"`
	WarehouseLocation *string  `json:"warehouse_location"`
	UOM               *string  `json:"uom"`
	StockQty          float64  `json:"stock_qty"`
	StockWeightKg     *float64 `json:"stock_weight_kg"`
	KanbanCount       *int     `json:"kanban_count"`
	KanbanStandardQty *int     `json:"kanban_standard_qty"`
	SafetyStockQty    *float64 `json:"safety_stock_qty"`
	DailyUsageQty     *float64 `json:"daily_usage_qty"`
}

type BulkCreateRawMaterialRequest struct {
	Items []CreateRawMaterialRequest `json:"items" validate:"required,min=1"`
}

type UpdateRawMaterialRequest struct {
	RawMaterialType   *string  `json:"raw_material_type"`
	RMSource          *string  `json:"rm_source"`
	PartNumber        *string  `json:"part_number"`
	PartName          *string  `json:"part_name"`
	WarehouseLocation *string  `json:"warehouse_location"`
	UOM               *string  `json:"uom"`
	StockQty          *float64 `json:"stock_qty"`
	StockWeightKg     *float64 `json:"stock_weight_kg"`
	KanbanCount       *int     `json:"kanban_count"`
	KanbanStandardQty *int     `json:"kanban_standard_qty"`
	SafetyStockQty    *float64 `json:"safety_stock_qty"`
	DailyUsageQty     *float64 `json:"daily_usage_qty"`
}

// ---------------------------------------------------------------------------
// Indirect Raw Material
// ---------------------------------------------------------------------------

type CreateIndirectMaterialRequest struct {
	UniqCode          string   `json:"uniq_code" validate:"required"`
	ItemUniqCode      *string  `json:"item_uniq_code"`
	PartNumber        *string  `json:"part_number"`
	PartName          *string  `json:"part_name"`
	WarehouseLocation *string  `json:"warehouse_location"`
	UOM               *string  `json:"uom"`
	StockQty          float64  `json:"stock_qty"`
	StockWeightKg     *float64 `json:"stock_weight_kg"`
	KanbanCount       *int     `json:"kanban_count"`
	KanbanStandardQty *int     `json:"kanban_standard_qty"`
	SafetyStockQty    *float64 `json:"safety_stock_qty"`
	DailyUsageQty     *float64 `json:"daily_usage_qty"`
}

type BulkCreateIndirectMaterialRequest struct {
	Items []CreateIndirectMaterialRequest `json:"items" validate:"required,min=1"`
}

type UpdateIndirectMaterialRequest struct {
	PartNumber        *string  `json:"part_number"`
	PartName          *string  `json:"part_name"`
	WarehouseLocation *string  `json:"warehouse_location"`
	UOM               *string  `json:"uom"`
	StockQty          *float64 `json:"stock_qty"`
	StockWeightKg     *float64 `json:"stock_weight_kg"`
	KanbanCount       *int     `json:"kanban_count"`
	KanbanStandardQty *int     `json:"kanban_standard_qty"`
	SafetyStockQty    *float64 `json:"safety_stock_qty"`
	DailyUsageQty     *float64 `json:"daily_usage_qty"`
}

// ---------------------------------------------------------------------------
// Subcon Inventory
// ---------------------------------------------------------------------------

type CreateSubconInventoryRequest struct {
	UniqCode         string     `json:"uniq_code" validate:"required"`
	ItemUniqCode     *string    `json:"item_uniq_code"`
	PartNumber       *string    `json:"part_number"`
	PartName         *string    `json:"part_name"`
	PONumber         *string    `json:"po_number"`
	POPeriod         *string    `json:"po_period"`
	SubconVendorID   *int64     `json:"subcon_vendor_id"`
	SubconVendorName *string    `json:"subcon_vendor_name"`
	StockAtVendorQty float64    `json:"stock_at_vendor_qty"`
	TotalPOQty       *float64   `json:"total_po_qty"`
	SafetyStockQty   *float64   `json:"safety_stock_qty"`
	DateDelivery     *time.Time `json:"date_delivery"`
}

type UpdateSubconInventoryRequest struct {
	PartNumber       *string    `json:"part_number"`
	PartName         *string    `json:"part_name"`
	PONumber         *string    `json:"po_number"`
	POPeriod         *string    `json:"po_period"`
	SubconVendorID   *int64     `json:"subcon_vendor_id"`
	SubconVendorName *string    `json:"subcon_vendor_name"`
	StockAtVendorQty *float64   `json:"stock_at_vendor_qty"`
	TotalPOQty       *float64   `json:"total_po_qty"`
	SafetyStockQty   *float64   `json:"safety_stock_qty"`
	DateDelivery     *time.Time `json:"date_delivery"`
}
