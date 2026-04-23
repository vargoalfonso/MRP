package models

import "time"

// ScrapPagination is the standard pagination envelope for scrap list endpoints.
type ScrapPagination struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

// ---------------------------------------------------------------------------
// Stats (4 cards on dashboard)
// ---------------------------------------------------------------------------

// ScrapStockStats is returned by GET /api/v1/scrap-stocks/stats.
type ScrapStockStats struct {
	TotalItems    int64   `json:"total_items"`    // count of active scrap stock records
	TotalQty      float64 `json:"total_qty"`      // sum of quantity
	TotalWeightKg float64 `json:"total_weight_kg"` // sum of weight_kg
	ScrapTypes    int64   `json:"scrap_types"`    // count distinct scrap_type
}

// ---------------------------------------------------------------------------
// Scrap Stock responses
// ---------------------------------------------------------------------------

// ScrapStockItem is the per-row shape for list and detail responses.
type ScrapStockItem struct {
	ID            int64      `json:"id"`
	UUID          string     `json:"uuid"`
	UniqCode      string     `json:"uniq"`
	PartNumber    *string    `json:"part_number"`
	PartName      *string    `json:"part_name"`
	Model         *string    `json:"model"`
	PackingNumber *string    `json:"packing_number"`
	WONumber      *string    `json:"wo_number"`
	ScrapType      string  `json:"scrap_type"`
	DisposalReason *string `json:"disposal_reason"`
	Quantity       float64 `json:"quantity"`
	UOM           *string    `json:"uom"`
	WeightKg      *float64   `json:"weight_kg"`
	DateReceived  *time.Time `json:"date_received"`
	Validator     *string    `json:"validator"`
	Remarks       *string    `json:"remarks"`
	Status        string     `json:"status"`
	CreatedBy     *string    `json:"created_by"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ScrapStockListResponse is the list envelope for GET /scrap-stocks.
type ScrapStockListResponse struct {
	Items      []ScrapStockItem `json:"items"`
	Pagination ScrapPagination  `json:"pagination"`
}

// ---------------------------------------------------------------------------
// Scrap Release responses
// ---------------------------------------------------------------------------

// ScrapReleaseItem is the per-row shape for list and detail responses.
type ScrapReleaseItem struct {
	ID             int64      `json:"id"`
	UUID           string     `json:"uuid"`
	ReleaseNumber  string     `json:"release_number"`
	ScrapStockID   int64      `json:"scrap_stock_id"`
	ReleaseDate    *time.Time `json:"release_date"`
	ReleaseType    string     `json:"release_type"`
	ReleaseQty     float64    `json:"release_qty"`
	WeightReleased *float64   `json:"weight_released"`
	CustomerName   *string    `json:"customer_name"`
	PricePerUnit   *float64   `json:"price_per_unit"`
	TotalValue     *float64   `json:"total_value"`
	DisposalReason *string    `json:"disposal_reason"`
	ApprovalStatus string     `json:"approval_status"`
	Validator      *string    `json:"validator"`
	Approver       *string    `json:"approver"`
	ApprovedBy     *string    `json:"approved_by"`
	ApprovedAt     *time.Time `json:"approved_at"`
	Remarks        *string    `json:"remarks"`
	CreatedBy      *string    `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ScrapReleaseListResponse is the list envelope for GET /scrap-releases.
type ScrapReleaseListResponse struct {
	Items      []ScrapReleaseItem `json:"items"`
	Pagination ScrapPagination    `json:"pagination"`
}

// ---------------------------------------------------------------------------
// Scrap Movement History (History Log tab)
// ---------------------------------------------------------------------------

// ScrapMovementItem is one row in the History Log tab.
type ScrapMovementItem struct {
	ID           int64     `json:"id"`
	UniqCode     string    `json:"uniq"`
	PackingList  *string   `json:"packing_list"`
	QtyChange    float64   `json:"qty_change"`    // signed: +6 masuk, -10 keluar
	CurrentStock float64   `json:"current_stock"` // live balance from scrap_stocks.quantity
	Reason       string    `json:"reason"`        // human-readable label
	ReferenceID  *string   `json:"reference_id"`
	LoggedBy     *string   `json:"logged_by"`
	LoggedAt     time.Time `json:"logged_at"`
}

// ScrapMovementListResponse is the list envelope for GET /scrap-stocks/:id/history-logs.
type ScrapMovementListResponse struct {
	Items      []ScrapMovementItem `json:"items"`
	Pagination ScrapPagination     `json:"pagination"`
}
