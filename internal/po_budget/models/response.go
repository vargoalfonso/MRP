package models

import "time"

// ---------------------------------------------------------------------------
// Single entry response
// ---------------------------------------------------------------------------

type EntryResponse struct {
	ID int64 `json:"id"`
	// PoBudgetRef is a server-generated human-friendly identifier.
	// Format: POB-{YYYY}-{TYPE}-{id}, e.g. POB-2026-RM-000123
	PoBudgetRef  string  `json:"po_budget_ref"`
	BudgetType   string  `json:"budget_type"`
	CustomerID   *int64  `json:"customer_id"`
	CustomerName *string `json:"customer_name"`
	// Uniq is an alias for uniq_code (frontend column naming).
	Uniq            string    `json:"uniq"`
	UniqCode        string    `json:"uniq_code"`
	ProductModel    *string   `json:"product_model"`
	MaterialType    *string   `json:"material_type"`
	PartName        *string   `json:"part_name"`
	PartNumber      *string   `json:"part_number"`
	Quantity        float64   `json:"quantity"`
	Uom             *string   `json:"uom"`
	WeightKg        *float64  `json:"weight_kg"`
	Description     *string   `json:"description"`
	SupplierID      *string   `json:"supplier_id"`
	SupplierName    *string   `json:"supplier_name"`
	Period          string    `json:"period"`
	PeriodDate      time.Time `json:"period_date"`
	SalesPlan       float64   `json:"sales_plan"`
	PurchaseRequest float64   `json:"purchase_request"`
	Po1Pct          float64   `json:"po1_pct"`
	Po2Pct          float64   `json:"po2_pct"`
	Po1Qty          float64   `json:"po1_qty"`  // purchase_request * po1_pct/100
	Po2Qty          float64   `json:"po2_qty"`  // purchase_request * po2_pct/100
	TotalPO         float64   `json:"total_po"` // po1_qty + po2_qty

	// UI-friendly aliases for "Calculation Results" cards.
	// These are quantities/amounts in the same unit as purchase_request & prl.
	Po1Amount       float64    `json:"po1_amount"`
	Po2Amount       float64    `json:"po2_amount"`
	TotalPOAmount   float64    `json:"total_po_amount"`
	ApoPrlAmount    float64    `json:"apo_prl_amount"`
	ApoPrlState     string     `json:"apo_prl_state"` // over|under|match
	Prl             float64    `json:"prl"`
	DeltaApoPrl     float64    `json:"delta_apo_prl"` // total_po - prl
	Status          string     `json:"status"`
	BudgetSubtype   *string    `json:"budget_subtype,omitempty"` // adhoc | regular | null
	PrlRef          *string    `json:"prl_ref,omitempty"`        // prls.prl_id
	PrlRowID        *int64     `json:"prl_row_id,omitempty"`     // prls.id
	ApprovedBy      *string    `json:"approved_by"`
	ApprovedAt      *time.Time `json:"approved_at"`
	ApprovedByName  *string    `json:"approved_by_name,omitempty"`
	SubmittedBy     *string    `json:"submitted_by,omitempty"`
	SubmittedAt     time.Time  `json:"submitted_at"`
	SubmittedByName *string    `json:"submitted_by_name,omitempty"`
	CreatedBy       *string    `json:"created_by"`
	CreatedByName   *string    `json:"created_by_name,omitempty"`
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
	// Uniq is an alias for uniq_code (frontend column naming).
	Uniq           string  `json:"uniq"`
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

	// UI helper for the APO-PRL card.
	ApoPrlAmount     float64 `json:"apo_prl_amount"` // same as delta_apo_prl
	ApoPrlAbs        float64 `json:"apo_prl_abs"`    // abs(delta)
	ApoPrlState      string  `json:"apo_prl_state"`  // over|under|match
	PendingApprovals int64   `json:"pending_approvals"`

	// Breakdown by budget_subtype (regular/adhoc). Null is treated as regular.
	TotalEntriesRegular     int64   `json:"total_entries_regular"`
	TotalEntriesAdhoc       int64   `json:"total_entries_adhoc"`
	TotalPurchaseReqRegular float64 `json:"total_purchase_request_regular"`
	TotalPurchaseReqAdhoc   float64 `json:"total_purchase_request_adhoc"`
	TotalPORegular          float64 `json:"total_po_regular"`
	TotalPOAdhoc            float64 `json:"total_po_adhoc"`
	TotalPRLRegular         float64 `json:"total_prl_regular"`
	TotalPRLAdhoc           float64 `json:"total_prl_adhoc"`
	PendingApprovalsRegular int64   `json:"pending_approvals_regular"`
	PendingApprovalsAdhoc   int64   `json:"pending_approvals_adhoc"`
}

type EntryDetailResponse struct {
	Entry   EntryResponse    `json:"entry"`
	Summary SummaryResponse  `json:"summary"`
	History []HistoryLogItem `json:"history"`
}

// EntryDetailGroupedResponse is a UI-friendly, de-duplicated response for the
// PO Budget detail screen.
//
// Use query param: ?format=grouped
type EntryDetailGroupedResponse struct {
	BasicInformation      EntryBasicInformation      `json:"basic_information"`
	BudgetCalculations    EntryBudgetCalculations    `json:"budget_calculations"`
	CalculationResults    EntryCalculationResults    `json:"calculation_results"`
	AdditionalInformation EntryAdditionalInformation `json:"additional_information"`
	History               []HistoryLogItem           `json:"history"`
}

type EntryBasicInformation struct {
	ID           int64   `json:"id"`
	PoBudgetRef  string  `json:"po_budget_ref"`
	CustomerName *string `json:"customer_name"`
	Uniq         string  `json:"uniq"`
	ProductModel *string `json:"product_model"`
	PartName     *string `json:"part_name"`
	PartNumber   *string `json:"part_number"`
	SupplierName *string `json:"supplier_name"`
	BudgetType   string  `json:"budget_type"`
	TypeLabel    *string `json:"type_label,omitempty"` // budget_subtype: regular|adhoc
	Period       string  `json:"period"`
}

type EntryBudgetCalculations struct {
	SalesPlan       float64 `json:"sales_plan"`
	PurchaseRequest float64 `json:"purchase_request"`
	PrlAmount       float64 `json:"prl_amount"`
	Po1Pct          float64 `json:"po1_pct"`
	Po2Pct          float64 `json:"po2_pct"`
}

type EntryCalculationResults struct {
	Po1Amount   float64 `json:"po1_amount"`
	Po2Amount   float64 `json:"po2_amount"`
	TotalPO     float64 `json:"total_po"`
	ApoPrlAbs   float64 `json:"apo_prl_abs"`
	ApoPrlState string  `json:"apo_prl_state"` // over|under|match
}

type EntryAdditionalInformation struct {
	SubmittedBy     *string    `json:"submitted_by,omitempty"`
	SubmittedByName *string    `json:"submitted_by_name,omitempty"`
	SubmittedAt     time.Time  `json:"submitted_at"`
	ApprovedBy      *string    `json:"approved_by,omitempty"`
	ApprovedByName  *string    `json:"approved_by_name,omitempty"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
}

type HistoryLogItem struct {
	DateTime string `json:"date_time"` // RFC3339
	Action   string `json:"action"`
	// User is the display name (resolved from users table when possible).
	User *string `json:"user"`
	// UserID is the actor uid (JWT uid / users.uuid) for audit.
	UserID *string `json:"user_id,omitempty"`
	Notes  *string `json:"notes"`
}

// ---------------------------------------------------------------------------
// PO Split Setting response
// ---------------------------------------------------------------------------

type SplitSettingResponse struct {
	ID            int64   `json:"id"`
	BudgetType    string  `json:"budget_type"`
	Po1Pct        float64 `json:"po1_pct"`
	Po2Pct        float64 `json:"po2_pct"`
	MinOrderQty   *int64  `json:"min_order_qty,omitempty"`
	MaxSplitLines *int64  `json:"max_split_lines,omitempty"`
	SplitRule     *string `json:"split_rule,omitempty"`
	Status        string  `json:"status"`
	Description   string  `json:"description"`
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
