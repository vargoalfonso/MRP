package models

// ---------------------------------------------------------------------------
// Scrap Stock requests
// ---------------------------------------------------------------------------

// CreateScrapStockRequest is the body for POST /api/v1/scrap-stocks.
type CreateScrapStockRequest struct {
	UniqCode      string   `json:"uniq"          validate:"required"`
	PartNumber    *string  `json:"part_number"`
	PartName      *string  `json:"part_name"`
	Model         *string  `json:"model"`
	PackingNumber *string  `json:"packing_number"`
	WONumber      *string  `json:"wo_number"`
	ScrapType     string   `json:"scrap_type"    validate:"required"`
	Quantity      float64  `json:"quantity"      validate:"required,gt=0"`
	UOM           *string  `json:"uom"`
	WeightKg      *float64 `json:"weight_kg"`
	DateReceived  *string  `json:"date_received"` // YYYY-MM-DD
	Remarks       *string  `json:"remarks"`
}

// ---------------------------------------------------------------------------
// Incoming Scrap (Action UI) request
// ---------------------------------------------------------------------------

// IncomingScrapRequest is the body for POST /api/v1/action-ui/scrap/incoming.
type IncomingScrapRequest struct {
	UniqCode      string   `json:"uniq"             validate:"required"`
	PackingNumber *string  `json:"packing_number"`
	WONumber      *string  `json:"wo_number"`
	ScrapType     string   `json:"scrap_type"       validate:"required"`
	Quantity      float64  `json:"quantity"         validate:"required,gt=0"`
	UOM           *string  `json:"uom"`
	WeightKg      *float64 `json:"weight_kg"`
	DateReceived  *string  `json:"date_received"`   // YYYY-MM-DD
	ClientEventID *string  `json:"client_event_id"` // idempotency key (UUID)
}

// ---------------------------------------------------------------------------
// Scrap Release requests
// ---------------------------------------------------------------------------

// CreateScrapReleaseRequest is the body for POST /api/v1/scrap-releases.
type CreateScrapReleaseRequest struct {
	ScrapStockID   int64    `json:"scrap_stock_id"  validate:"required,gt=0"`
	ReleaseDate    *string  `json:"release_date"`    // YYYY-MM-DD
	ReleaseType    string   `json:"release_type"    validate:"required"` // Sell | Dump
	ReleaseQty     float64  `json:"release_qty"     validate:"required,gt=0"`
	WeightReleased *float64 `json:"weight_released"`
	CustomerName   *string  `json:"customer_name"`
	PricePerUnit   *float64 `json:"price_per_unit"`
	DisposalReason *string  `json:"disposal_reason"`
	Approver       *string  `json:"approver"`
	Remarks        *string  `json:"remarks"`
}

// ApproveScrapReleaseRequest is the body for PUT /api/v1/scrap-releases/:id/approve.
type ApproveScrapReleaseRequest struct {
	// Action: "Completed" (approve) or "Rejected"
	Action  string  `json:"action"  validate:"required"`
	Remarks *string `json:"remarks"`
}
