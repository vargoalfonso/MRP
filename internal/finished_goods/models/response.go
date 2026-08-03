package models

import "time"

// FGPagination is the standard pagination envelope for FG list endpoints.
type FGPagination struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

// ---------------------------------------------------------------------------
// Summary (4 cards on dashboard)
// ---------------------------------------------------------------------------

// FGSummary is returned by GET /api/v1/finished-goods/summary.
type FGSummary struct {
	TotalFGItems  int64   `json:"total_fg_items"`  // active uniq count
	LowStockItems int64   `json:"low_stock_items"` // stock_qty < min_threshold
	TotalStock    float64 `json:"total_stock"`     // sum of stock_qty
	ActiveAlerts  int64   `json:"active_alerts"`   // low_on_stock + overstock count
}

// ---------------------------------------------------------------------------
// Status Monitoring (Status Monitoring tab)
// ---------------------------------------------------------------------------

// FGStatusMonitoringSummary is the grouped header for the Status Monitoring tab.
type FGStatusMonitoringSummary struct {
	LowStockCount  int64 `json:"low_stock_count"`
	OverstockCount int64 `json:"overstock_count"`
	NormalCount    int64 `json:"normal_count"`
	TotalAlerts    int64 `json:"total_alerts"` // low + overstock
}

// FGAlertItem is a single row in the Status Monitoring alert table.
type FGAlertItem struct {
	ID             int64     `json:"id"`
	UniqCode       string    `json:"uniq_code"`
	PartName       *string   `json:"part_name"`
	Model          *string   `json:"model"`
	AlertType      string    `json:"alert_type"` // low_on_stock | overstock
	CurrentStock   float64   `json:"current_stock"`
	Threshold      float64   `json:"threshold"`      // min_threshold if low; max_threshold if over
	Recommendation string    `json:"recommendation"` // "Schedule production immediately" | "Consider delivery acceleration"
	Priority       string    `json:"priority"`       // "High" | "Medium"
	UpdatedAt      time.Time `json:"updated_at"`
}

// FGStatusMonitoringResponse is the full response for GET /api/v1/finished-goods/status-monitoring.
type FGStatusMonitoringResponse struct {
	Summary    FGStatusMonitoringSummary `json:"summary"`
	Items      []FGAlertItem             `json:"items"`
	Pagination FGPagination              `json:"pagination"`
}

// ---------------------------------------------------------------------------
// Finished Goods item (list + detail)
// ---------------------------------------------------------------------------

// FinishedGoodsListItem is the lightweight row shape for the FG inventory list.
// Dynamic stock/kanban/status metrics are fetched via parameterized-summary per row.
type FinishedGoodsListItem struct {
	ID                int64     `json:"id"`
	UUID              string    `json:"uuid"`
	UniqCode          string    `json:"uniq_code"`
	PartNumber        *string   `json:"part_number"`
	PartName          *string   `json:"part_name"`
	Model             *string   `json:"model"`
	WONumber          *string   `json:"wo_number"`
	WarehouseLocation *string   `json:"warehouse_location"`
	UOM               *string   `json:"uom"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// FinishedGoodsItem is the per-row shape for list and detail responses.
// kanban_progress is computed on-the-fly and NOT stored in DB.
type FinishedGoodsItem struct {
	ID                    int64     `json:"id"`
	UUID                  string    `json:"uuid"`
	UniqCode              string    `json:"uniq_code"`
	PartNumber            *string   `json:"part_number"`
	PartName              *string   `json:"part_name"`
	Model                 *string   `json:"model"`
	WONumber              *string   `json:"wo_number"`
	WarehouseLocation     *string   `json:"warehouse_location"`
	StockQty              float64   `json:"stock_qty"`
	TargetStockQty        *float64  `json:"target_stock_qty"`
	UOM                   *string   `json:"uom"`
	KanbanCount           *int      `json:"kanban_count"`
	CurrentKanban         *int      `json:"current_kanban"`
	KanbanStandardQty     *int      `json:"kanban_standard_qty"`
	SafetyStockQty        *float64  `json:"safety_stock_qty"`
	MinThreshold          *float64  `json:"min_threshold"`
	MaxThreshold          *float64  `json:"max_threshold"`
	StockToCompleteKanban *float64  `json:"stock_to_complete_kanban"`
	StockGapToTarget      *float64  `json:"stock_gap_to_target"`
	KanbanNeed            *int      `json:"kanban_need"`
	StockToKanbanPCS      *float64  `json:"stock_to_kanban_pcs"`
	StockAfterReplenish   *float64  `json:"stock_after_replenish"`
	KanbanProgress        int       `json:"kanban_progress"` // floor(stock/safety*100), computed
	Status                string    `json:"status"`
	CreatedBy             *string   `json:"created_by,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// FinishedGoodsListResponse is the list envelope for GET /finished-goods.
type FinishedGoodsListResponse struct {
	Items      []FinishedGoodsListItem `json:"items"`
	Pagination FGPagination            `json:"pagination"`
}

// FGCreateUniqOptionItem is one option row for create-form uniq autocomplete.
type FGCreateUniqOptionItem struct {
	UniqCode     string   `json:"uniq_code"`
	PartNumber   *string  `json:"part_number"`
	PartName     *string  `json:"part_name"`
	Model        *string  `json:"model"`
	LastWONumber *string  `json:"last_wo_number"`
	KanbanQty    *int     `json:"kanban_qty"`    // max pcs per 1 kanban (from kanban_parameters)
	MinThreshold *float64 `json:"min_threshold"` // from kanban_parameters.min_stock
	MaxThreshold *float64 `json:"max_threshold"` // from kanban_parameters.max_stock
}

// FGCreateUniqOptionsResponse is response for create-form uniq autocomplete endpoint.
type FGCreateUniqOptionsResponse struct {
	Items []FGCreateUniqOptionItem `json:"items"`
}

// FGBulkCreateResult is the result for one item in a bulk create operation.
type FGBulkCreateResult struct {
	Index    int     `json:"index"`
	UniqCode string  `json:"uniq_code"`
	Status   string  `json:"status"` // "created" | "failed"
	ID       *int64  `json:"id,omitempty"`
	UUID     *string `json:"uuid,omitempty"`
	Error    *string `json:"error,omitempty"`
}

// FGBulkCreateResponse is returned by POST /api/v1/finished-goods/bulk.
type FGBulkCreateResponse struct {
	Created int                  `json:"created"`
	Failed  int                  `json:"failed"`
	Results []FGBulkCreateResult `json:"results"`
}

// FGParameterizedSummary is a dynamic per-item summary for row-level UI refresh.
// Values are computed fresh from finished_goods + kanban_parameters on each request.
type FGParameterizedSummary struct {
	UniqCode            string   `json:"uniq_code"`
	PartNumber          *string  `json:"part_number"`
	PartName            *string  `json:"part_name"`
	Model               *string  `json:"model"`
	WONumber            *string  `json:"wo_number"`
	WarehouseLocation   *string  `json:"warehouse_location"`
	StockQty            float64  `json:"stock_qty"`
	UOM                 *string  `json:"uom"`
	KanbanStandardQty   *int     `json:"kanban_standard_qty"`
	MinThreshold        *float64 `json:"min_threshold"`
	MaxThreshold        *float64 `json:"max_threshold"`
	TargetStockQty      *float64 `json:"target_stock_qty"`
	CurrentKanban       *int     `json:"current_kanban"`
	StockGapToTarget    *float64 `json:"stock_gap_to_target"`
	KanbanNeed          *int     `json:"kanban_need"`
	StockToKanbanPCS    *float64 `json:"stock_to_kanban_pcs"`
	StockAfterReplenish *float64 `json:"stock_after_replenish"`
	Status              string   `json:"status"`
	ParameterSource     string   `json:"parameter_source"`
}

// ---------------------------------------------------------------------------
// History Log (In-Out Activity Log on the FG detail view)
// ---------------------------------------------------------------------------

// FGMovementLogItem is one row of the In-Out Activity Log, derived from
// fg_movement_logs. qty_change is the signed stock delta (positive = in,
// negative = out); qty_after is the resulting balance.
type FGMovementLogItem struct {
	ID           int64     `json:"id"`
	UniqCode     string    `json:"uniq_code"`
	MovementType string    `json:"movement_type"`
	Reason       string    `json:"reason"`
	QtyChange    float64   `json:"qty_change"`
	QtyBefore    float64   `json:"qty_before"`
	QtyAfter     float64   `json:"qty_after"`
	WONumber     *string   `json:"wo_number"`
	DNNumber     *string   `json:"dn_number"`
	ReferenceID  *string   `json:"reference_id"` // kanban / packing list reference
	Notes        *string   `json:"notes"`
	LoggedBy     *string   `json:"logged_by"`
	LoggedAt     time.Time `json:"logged_at"`
}

// FGHistoryResponse is returned by GET /api/v1/finished-goods/history?uniq_code=...
type FGHistoryResponse struct {
	UniqCode   string              `json:"uniq_code"`
	Items      []FGMovementLogItem `json:"items"`
	Pagination FGPagination        `json:"pagination"`
}

// ---------------------------------------------------------------------------
// Packing List (FG detail modal)
// ---------------------------------------------------------------------------

// FGPackingListItem is one packing/kanban row for a finished good. Rows are
// resolved from the work order barcode (work_order_items.kanban_number) and
// enriched with the delivery note that consumes the same packing number.
type FGPackingListItem struct {
	DNNumber      *string `json:"dn_number"`
	PackingNumber string  `json:"packing_number"`
	Quantity      float64 `json:"quantity"`    // qty planned on the packing
	QtyCurrent    float64 `json:"qty_current"` // qty saat ini (scan / opname result)
	QtyMax        float64 `json:"qty_max"`     // qty maksimal (kanban standard)
	Progress      int     `json:"progress"`    // 0-100, qty_current / qty_max
	Status        *string `json:"status"`
	WONumber      *string `json:"wo_number"`
	Source        string  `json:"source"` // work_order | delivery_note
}

// FGPackingListResponse is returned by GET /api/v1/finished-goods/packing-list.
type FGPackingListResponse struct {
	UniqCode        string              `json:"uniq_code"`
	Items           []FGPackingListItem `json:"items"`
	TotalPacking    int                 `json:"total_packing"`
	TotalQtyCurrent float64             `json:"total_qty_current"`
	TotalQtyMax     float64             `json:"total_qty_max"`
}
