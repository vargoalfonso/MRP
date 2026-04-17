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
	UOM                   *string   `json:"uom"`
	KanbanCount           *int      `json:"kanban_count"`
	KanbanStandardQty     *int      `json:"kanban_standard_qty"`
	SafetyStockQty        *float64  `json:"safety_stock_qty"`
	MinThreshold          *float64  `json:"min_threshold"`
	MaxThreshold          *float64  `json:"max_threshold"`
	StockToCompleteKanban *float64  `json:"stock_to_complete_kanban"`
	KanbanProgress        int       `json:"kanban_progress"` // floor(stock/safety*100), computed
	Status                string    `json:"status"`
	CreatedBy             *string   `json:"created_by,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// FinishedGoodsListResponse is the list envelope for GET /finished-goods.
type FinishedGoodsListResponse struct {
	Items      []FinishedGoodsItem `json:"items"`
	Pagination FGPagination        `json:"pagination"`
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
