package models

import "time"

// ---------------------------------------------------------------------------
// Shared
// ---------------------------------------------------------------------------

// KanbanSummary is returned by GET /api/v1/inventory/kanban-summary?uniq_code=X
// Frontend calls this per-row asynchronously to show kanban progress and stock status per item.
type KanbanSummary struct {
	UniqCode string `json:"uniq_code"`

	// Stock expressed in kanban units
	StockQty    float64 `json:"stock_qty"`    // raw stock qty in UoM
	TotalKanban int64   `json:"total_kanban"` // floor(stock_qty / kanban_pkg_qty)

	// How many more kanbans must be ordered to reach safety stock
	KanbansNeeded   int64   `json:"kanbans_needed"`    // ceil(deficit / kanban_pkg_qty), 0 if no deficit
	StockToComplete float64 `json:"stock_to_complete"` // kanbans_needed × kanban_pkg_qty (pcs)
	KanbanPkgQty    int     `json:"kanban_pkg_qty"`    // pcs per kanban/package

	// Safety stock threshold (computed from parameter or stored value)
	SafetyStockQty float64 `json:"safety_stock_qty"`

	// Status & buy recommendation
	Status    string `json:"status"`      // low_on_stock | normal | overstock
	BuyNotBuy string `json:"buy_not_buy"` // buy | not_buy | n/a
	StockDays *int   `json:"stock_days"`  // floor(stock_qty / effective_daily_usage)

	// Calculation context — shows exactly which values were used so behaviour is transparent.
	CalcContext KanbanCalcContext `json:"calc_context"`
}

// KanbanCalcContext carries the intermediate values used to produce the KanbanSummary.
// Useful for debugging parameter changes without having to query multiple tables manually.
type KanbanCalcContext struct {
	ForecastPeriod       string   `json:"forecast_period"`        // period used for PRL + working_days lookup
	WorkingDays          int      `json:"working_days"`           // working_days active for that period
	PRLTotal             float64  `json:"prl_total"`              // sum of approved PRL qty for the period
	PRLFromCache         bool     `json:"prl_from_cache"`         // true = read from prl_item_period_summaries
	BaseDailyUsage       float64  `json:"base_daily_usage"`       // prl_total / working_days
	StockdaysCalcType    string   `json:"stockdays_calc_type"`    // normalised calculation_type from stockdays_parameters
	StockdaysConstanta   float64  `json:"stockdays_constanta"`    // constanta from stockdays_parameters (0 = not set)
	EffectiveDailyUsage  float64  `json:"effective_daily_usage"`  // base_daily_usage * stockdays_constanta
	SafetyStockCalcType  string   `json:"safety_stock_calc_type"` // normalised calculation_type from safety_stock_parameters
	SafetyStockConstanta float64  `json:"safety_stock_constanta"` // constanta from safety_stock_parameters (0 = not set)
	Warnings             []string `json:"warnings,omitempty"`     // non-empty when a calc_type is not yet implemented
}

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
