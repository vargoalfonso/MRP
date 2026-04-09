package models

import "time"

// ---------------------------------------------------------------------------
// Single entry response
// ---------------------------------------------------------------------------

type EntryResponse struct {
	ID              int64      `json:"id"`
	BudgetType      string     `json:"budget_type"`
	CustomerID      *int64     `json:"customer_id"`
	CustomerName    *string    `json:"customer_name"`
	UniqCode        string     `json:"uniq_code"`
	ProductModel    *string    `json:"product_model"`
	MaterialType    *string    `json:"material_type"`
	PartName        *string    `json:"part_name"`
	PartNumber      *string    `json:"part_number"`
	Quantity        float64    `json:"quantity"`
	Uom             *string    `json:"uom"`
	WeightKg        *float64   `json:"weight_kg"`
	Description     *string    `json:"description"`
	SupplierID      *string    `json:"supplier_id"`
	SupplierName    *string    `json:"supplier_name"`
	Period          string     `json:"period"`
	PeriodDate      time.Time  `json:"period_date"`
	SalesPlan       float64    `json:"sales_plan"`
	PurchaseRequest float64    `json:"purchase_request"`
	Po1Pct          float64    `json:"po1_pct"`
	Po2Pct          float64    `json:"po2_pct"`
	Po1Qty          float64    `json:"po1_qty"`  // purchase_request * po1_pct/100
	Po2Qty          float64    `json:"po2_qty"`  // purchase_request * po2_pct/100
	TotalPO         float64    `json:"total_po"` // po1_qty + po2_qty
	Prl             float64    `json:"prl"`
	DeltaApoPrl     float64    `json:"delta_apo_prl"` // total_po - prl
	Status          string     `json:"status"`
	ApprovedBy      *string    `json:"approved_by"`
	ApprovedAt      *time.Time `json:"approved_at"`
	CreatedBy       *string    `json:"created_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ---------------------------------------------------------------------------
// List response
// ---------------------------------------------------------------------------

type ListEntryResponse struct {
	Items []EntryResponse `json:"items"`
	Meta  ListMeta        `json:"pagination"`
}

type ListMeta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

// ---------------------------------------------------------------------------
// Aggregated view — sum by uniq_code + period
// Used as the default "current period" dashboard view.
// ---------------------------------------------------------------------------

type AggregatedRow struct {
	UniqCode       string  `json:"uniq_code"`
	CustomerName   string  `json:"customer_name"`
	Period         string  `json:"period"`
	ProductModel   string  `json:"product_model"`
	TotalSalesPlan float64 `json:"total_sales_plan"`
	TotalPR        float64 `json:"total_purchase_request"`
	TotalPO1       float64 `json:"total_po1"` // sum(purchase_request * po1_pct/100)
	TotalPO2       float64 `json:"total_po2"` // sum(purchase_request * po2_pct/100)
	TotalPO        float64 `json:"total_po"`  // total_po1 + total_po2
	TotalPrl       float64 `json:"total_prl"`
	DeltaApoPrl    float64 `json:"delta_apo_prl"` // total_po - total_prl
	RowCount       int     `json:"row_count"`
}

type AggregatedResponse struct {
	Items []AggregatedRow `json:"items"`
	Meta  ListMeta        `json:"pagination"`
}

// ---------------------------------------------------------------------------
// Summary (dashboard cards)
// ---------------------------------------------------------------------------

type SummaryResponse struct {
	TotalEntries     int64   `json:"total_entries"`
	TotalSalesPlan   float64 `json:"total_sales_plan"`
	TotalPurchaseReq float64 `json:"total_purchase_request"`
	TotalPO          float64 `json:"total_po"`
	TotalPRL         float64 `json:"total_prl"`
	DeltaApoPrl      float64 `json:"delta_apo_prl"`
	PendingApprovals int64   `json:"pending_approvals"`
}

// ---------------------------------------------------------------------------
// PO Split Setting response
// ---------------------------------------------------------------------------

type SplitSettingResponse struct {
	ID          int64   `json:"id"`
	BudgetType  string  `json:"budget_type"`
	Po1Pct      float64 `json:"po1_pct"`
	Po2Pct      float64 `json:"po2_pct"`
	Description string  `json:"description"`
}

// ---------------------------------------------------------------------------
// Import result
// ---------------------------------------------------------------------------

type ImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

// ---------------------------------------------------------------------------
// Bulk-from-PRL result
// ---------------------------------------------------------------------------

type BulkFromPRLResult struct {
	Created int      `json:"created"`
	Errors  []string `json:"errors,omitempty"`
}

// ---------------------------------------------------------------------------
// PRL responses (for Step 1 dropdown + Step 2 item list)
// ---------------------------------------------------------------------------

type PrlForecastResponse struct {
	ID           string                    `json:"id"`
	PrlNumber    string                    `json:"prl_number"`
	CustomerID   *int64                    `json:"customer_id"`
	CustomerName *string                   `json:"customer_name"`
	Period       string                    `json:"period"`
	Status       string                    `json:"status"`
	Items        []PrlForecastItemResponse `json:"items,omitempty"`
}

// PrlForecastItemResponse includes current allocation so UI can show
// "Budget: X | Allocated: Y" in the Add Supplier modal.
type PrlForecastItemResponse struct {
	ID                  int64    `json:"id"`
	UniqCode            string   `json:"uniq_code"`
	ProductModel        *string  `json:"product_model,omitempty"`
	PartName            *string  `json:"part_name"`
	PartNumber          *string  `json:"part_number"`
	WeightKg            *float64 `json:"weight_kg"`
	Quantity            float64  `json:"quantity"`      // budget ceiling
	AllocatedQty        float64  `json:"allocated_qty"` // sum already in po_budget_entries
	RemainingQty        float64  `json:"remaining_qty"` // quantity - allocated_qty
	ExistingRawMaterial *string  `json:"existing_raw_material"`
	Uom                 *string  `json:"uom"`
}

type ListPrlResponse struct {
	Items []PrlForecastResponse `json:"items"`
	Meta  ListMeta              `json:"pagination"`
}
