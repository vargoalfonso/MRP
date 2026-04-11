package models

import "time"

// ---------------------------------------------------------------------------
// Shared
// ---------------------------------------------------------------------------

type InventoryPagination struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

type InventoryStats struct {
	TotalItems         int64 `json:"total_items"`
	BuyRecommendations int64 `json:"buy_recommendations"`
	LowStockItems      int64 `json:"low_stock_items"`
}

// ---------------------------------------------------------------------------
// Raw Material
// ---------------------------------------------------------------------------

type RawMaterialItem struct {
	ID                    int64     `json:"id"`
	UniqCode              string    `json:"uniq_code"`
	PartNumber            *string   `json:"part_number"`
	PartName              *string   `json:"part_name"`
	RawMaterialType       string    `json:"raw_material_type"`
	RMSource              string    `json:"rm_source"`
	WarehouseLocation     *string   `json:"warehouse_location"`
	UOM                   *string   `json:"uom"`
	StockQty              float64   `json:"stock_qty"`
	StockWeightKg         *float64  `json:"stock_weight_kg"`
	KanbanCount           *int      `json:"kanban_count"`
	KanbanStandardQty     *int      `json:"kanban_standard_qty"`
	SafetyStockQty        *float64  `json:"safety_stock_qty"`
	DailyUsageQty         *float64  `json:"daily_usage_qty"`
	Status                string    `json:"status"`
	StockDays             *int      `json:"stock_days"`
	BuyNotBuy             string    `json:"buy_not_buy"`
	StockToCompleteKanban *float64  `json:"stock_to_complete_kanban"`
	CreatedBy             *string   `json:"created_by"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedBy             *string   `json:"updated_by"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type RawMaterialListResponse struct {
	Stats      InventoryStats      `json:"stats"`
	Items      []RawMaterialItem   `json:"items"`
	Pagination InventoryPagination `json:"pagination"`
}

// ---------------------------------------------------------------------------
// Indirect Raw Material
// ---------------------------------------------------------------------------

type IndirectMaterialItem struct {
	ID                    int64     `json:"id"`
	UniqCode              string    `json:"uniq_code"`
	PartNumber            *string   `json:"part_number"`
	PartName              *string   `json:"part_name"`
	WarehouseLocation     *string   `json:"warehouse_location"`
	UOM                   *string   `json:"uom"`
	StockQty              float64   `json:"stock_qty"`
	StockWeightKg         *float64  `json:"stock_weight_kg"`
	KanbanCount           *int      `json:"kanban_count"`
	KanbanStandardQty     *int      `json:"kanban_standard_qty"`
	SafetyStockQty        *float64  `json:"safety_stock_qty"`
	DailyUsageQty         *float64  `json:"daily_usage_qty"`
	Status                *string   `json:"status"`
	StockDays             *int      `json:"stock_days"`
	BuyNotBuy             string    `json:"buy_not_buy"`
	StockToCompleteKanban *float64  `json:"stock_to_complete_kanban"`
	CreatedBy             *string   `json:"created_by"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedBy             *string   `json:"updated_by"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type IndirectMaterialListResponse struct {
	Stats      InventoryStats         `json:"stats"`
	Items      []IndirectMaterialItem `json:"items"`
	Pagination InventoryPagination    `json:"pagination"`
}

// ---------------------------------------------------------------------------
// Subcon Inventory
// ---------------------------------------------------------------------------

type SubconInventoryItem struct {
	ID               int64      `json:"id"`
	UniqCode         string     `json:"uniq_code"`
	PartNumber       *string    `json:"part_number"`
	PartName         *string    `json:"part_name"`
	PONumber         *string    `json:"po_number"`
	POPeriod         *string    `json:"po_period"`
	SubconVendorID   *int64     `json:"subcon_vendor_id"`
	SubconVendorName *string    `json:"subcon_vendor_name"`
	StockAtVendorQty float64    `json:"stock_at_vendor_qty"`
	TotalPOQty       *float64   `json:"total_po_qty"`
	TotalReceivedQty *float64   `json:"total_received_qty"`
	DeltaPO          *float64   `json:"delta_po"`
	SafetyStockQty   *float64   `json:"safety_stock_qty"`
	DateDelivery     *time.Time `json:"date_delivery"`
	Status           string     `json:"status"`
	CreatedBy        *string    `json:"created_by"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedBy        *string    `json:"updated_by"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type SubconInventoryListResponse struct {
	Items      []SubconInventoryItem `json:"items"`
	Pagination InventoryPagination   `json:"pagination"`
}

// ---------------------------------------------------------------------------
// Movement History Log
// ---------------------------------------------------------------------------

type HistoryLogItem struct {
	ID            int64     `json:"id"`
	UniqCode      string    `json:"uniq_code"`
	KanbanPacking *string   `json:"kanban_packing"` // dn_number
	QtyChange     float64   `json:"qty_change"`
	WeightChange  *float64  `json:"weight_change"`
	MovementType  string    `json:"movement_type"`
	Reason        string    `json:"reason"`     // human-readable source_flag
	LogStatus     string    `json:"log_status"` // confirmed | pending | in_progress | rejected
	Notes         *string   `json:"notes"`
	LoggedBy      *string   `json:"logged_by"`
	LoggedAt      time.Time `json:"logged_at"`
}

type HistoryLogResponse struct {
	Items      []HistoryLogItem    `json:"items"`
	Pagination InventoryPagination `json:"pagination"`
}

// ---------------------------------------------------------------------------
// Incoming Scans (tab view)
// ---------------------------------------------------------------------------

type IncomingItem struct {
	ScanID          int64     `json:"scan_id"`
	UniqCode        string    `json:"uniq_code"`
	IncomingQty     float64   `json:"incoming_qty"`
	Warehouse       *string   `json:"warehouse"`
	ScanDate        time.Time `json:"scan_date"`
	SupplierName    *string   `json:"supplier_name"`
	PONumber        *string   `json:"po_number"`
	DNNumber        string    `json:"dn_number"`
	QCStatus        string    `json:"qc_status"`         // pending | in_progress | approved | rejected
	QCStatusDisplay string    `json:"qc_status_display"` // Pending Approval | Received | Rejected
	UOM             *string   `json:"uom"`
	DNType          string    `json:"dn_type"`
}

type IncomingListResponse struct {
	Items      []IncomingItem      `json:"items"`
	Pagination InventoryPagination `json:"pagination"`
}
