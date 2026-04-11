package models

// ---------------------------------------------------------------------------
// List / filter queries
// ---------------------------------------------------------------------------

type ListBudgetQuery struct {
	BudgetType     string // raw_material | subcon | indirect
	BudgetSubtype  string // regular | adhoc (empty = all)
	UniqCode       string
	CustomerID     int64
	Period         string // "October 2025"
	Status         string
	Search         string
	Page           int
	Limit          int
	OrderBy        string
	OrderDirection string
}

// ---------------------------------------------------------------------------
// Create / Update entry
// ---------------------------------------------------------------------------

type CreateEntryRequest struct {
	BudgetType string `json:"budget_type"      validate:"required,oneof=raw_material subcon indirect"`
	// BudgetSubtype distinguishes regular plan vs additional (adhoc) purchase.
	// If omitted, backend defaults to "regular".
	BudgetSubtype   *string  `json:"budget_subtype"   validate:"omitempty,oneof=adhoc regular"`
	CustomerID      *int64   `json:"customer_id"`
	CustomerName    *string  `json:"customer_name"`
	UniqCode        string   `json:"uniq_code"        validate:"required"`
	ProductModel    *string  `json:"product_model"`
	MaterialType    *string  `json:"material_type"`
	PartName        *string  `json:"part_name"`
	PartNumber      *string  `json:"part_number"`
	Quantity        float64  `json:"quantity"         validate:"gte=0"`
	Uom             *string  `json:"uom"`
	WeightKg        *float64 `json:"weight_kg"`
	Description     *string  `json:"description"`
	SupplierID      *int64   `json:"supplier_id"`
	SupplierName    *string  `json:"supplier_name"`
	Period          string   `json:"period"           validate:"required"` // "October 2025"
	SalesPlan       float64  `json:"sales_plan"       validate:"gte=0"`
	PurchaseRequest float64  `json:"purchase_request" validate:"gte=0"`
	// Po1Pct / Po2Pct optional — if omitted they are fetched from po_split_settings
	Po1Pct *float64 `json:"po1_pct"`
	Po2Pct *float64 `json:"po2_pct"`
	Prl    float64  `json:"prl"`
}

type UpdateEntryRequest struct {
	BudgetSubtype   *string  `json:"budget_subtype"   validate:"omitempty,oneof=adhoc regular"`
	CustomerID      *int64   `json:"customer_id"`
	CustomerName    *string  `json:"customer_name"`
	ProductModel    *string  `json:"product_model"`
	MaterialType    *string  `json:"material_type"`
	PartName        *string  `json:"part_name"`
	PartNumber      *string  `json:"part_number"`
	Quantity        *float64 `json:"quantity"`
	Uom             *string  `json:"uom"`
	WeightKg        *float64 `json:"weight_kg"`
	Description     *string  `json:"description"`
	SupplierID      *int64   `json:"supplier_id"`
	SupplierName    *string  `json:"supplier_name"`
	Period          *string  `json:"period"`
	SalesPlan       *float64 `json:"sales_plan"`
	PurchaseRequest *float64 `json:"purchase_request"`
	Po1Pct          *float64 `json:"po1_pct"`
	Po2Pct          *float64 `json:"po2_pct"`
	Prl             *float64 `json:"prl"`
	Status          *string  `json:"status"`
}

// ---------------------------------------------------------------------------
// PO Split Setting update
// ---------------------------------------------------------------------------

type UpdateSplitSettingRequest struct {
	Po1Pct        *float64 `json:"po1_pct"        validate:"omitempty,gte=0,lte=100"`
	Po2Pct        *float64 `json:"po2_pct"        validate:"omitempty,gte=0,lte=100"`
	MinOrderQty   *int64   `json:"min_order_qty"  validate:"omitempty,gte=0"`
	MaxSplitLines *int64   `json:"max_split_lines" validate:"omitempty,gte=0"`
	SplitRule     *string  `json:"split_rule"     validate:"omitempty"`
	Status        *string  `json:"status"         validate:"omitempty,oneof=Active Inactive"`
	Description   *string  `json:"description"    validate:"omitempty"`
}

// ---------------------------------------------------------------------------
// Approval
// ---------------------------------------------------------------------------

type ApproveRequest struct {
	Status string `json:"status"      validate:"required,oneof=Approved Rejected"`
	// NOTE: this is always overwritten by backend from JWT uid (claims.uid).
	ApprovedBy string  `json:"approved_by,omitempty"`
	Notes      *string `json:"notes"`
}

// ---------------------------------------------------------------------------
// Clear
// ---------------------------------------------------------------------------

type ClearRequest struct {
	// If IDs provided → delete those specific entries only.
	// If empty → delete all entries matching the budget_type + optional filters.
	IDs        []int64 `json:"ids"`
	BudgetType string  `json:"budget_type" validate:"required,oneof=raw_material subcon indirect"`
	Period     string  `json:"period"` // optional; if empty → clear all periods
}

// ---------------------------------------------------------------------------
// Bulk from PRL (wizard: Step 1 → 3)
// ---------------------------------------------------------------------------

// BulkFromPRLRequest is the single API call that covers all 3 wizard steps.
//
//	Step 1: prl_id + budget_subtype
//	Step 2: items[] each with suppliers[] — total supplier qty MUST NOT exceed item budget_qty
//	Step 3: period + po1_pct / po2_pct
type BulkFromPRLRequest struct {
	PrlID         string          `json:"prl_id"         validate:"required"`
	BudgetSubtype string          `json:"budget_subtype" validate:"required,oneof=adhoc regular"`
	Period        string          `json:"period"         validate:"required"`
	Po1Pct        *float64        `json:"po1_pct"` // optional; falls back to po_split_settings
	Po2Pct        *float64        `json:"po2_pct"`
	Items         []BulkItemInput `json:"items"          validate:"required,min=1,dive"`
}

// BulkItemInput is one UNIQ row from Step 2, with one or more supplier allocations.
// The sum of all Suppliers[].Quantity MUST NOT exceed BudgetQty (from PRL item).
type BulkItemInput struct {
	PrlItemID           int64               `json:"prl_item_id"   validate:"required"`
	UniqCode            string              `json:"uniq_code"     validate:"required"`
	ProductModel        *string             `json:"product_model"`
	MaterialType        *string             `json:"material_type"`
	PartName            *string             `json:"part_name"`
	PartNumber          *string             `json:"part_number"`
	WeightKg            *float64            `json:"weight_kg"`
	BudgetQty           float64             `json:"budget_qty"    validate:"gt=0"` // PRL item qty ceiling
	Uom                 *string             `json:"uom"`
	ExistingRawMaterial *string             `json:"existing_raw_material"`
	SalesPlan           float64             `json:"sales_plan"`
	Prl                 float64             `json:"prl"`
	Suppliers           []BulkSupplierInput `json:"suppliers"     validate:"required,min=1,dive"`
}

// BulkSupplierInput is one supplier allocation for a UNIQ item.
// Quantity is the portion of BudgetQty allocated to this supplier.
type BulkSupplierInput struct {
	SupplierID   *int64  `json:"supplier_id"`
	SupplierName string  `json:"supplier_name" validate:"required"`
	Quantity     float64 `json:"quantity"      validate:"gt=0"`
}

// ---------------------------------------------------------------------------
// Robot split preview
// ---------------------------------------------------------------------------

// RobotSplitRequest is the payload for POST /:type/budget/robot-split.
// po_type = "manual" → return { robot: false }, no pct fields.
// po_type = "robot"  → call external robot service, return po1_pct + po2_pct.
type RobotSplitRequest struct {
	PoType   string `json:"po_type"   validate:"required,oneof=manual robot"`
	UniqCode string `json:"uniq_code" validate:"required"`
}

// ---------------------------------------------------------------------------
// Excel import row
// ---------------------------------------------------------------------------

type ImportRow struct {
	UniqCode        string  `json:"uniq_code"`
	CustomerName    string  `json:"customer_name"`
	ProductModel    string  `json:"product_model"`
	MaterialType    string  `json:"material_type"`
	PartName        string  `json:"part_name"`
	PartNumber      string  `json:"part_number"`
	Quantity        float64 `json:"quantity"`
	Uom             string  `json:"uom"`
	WeightKg        float64 `json:"weight_kg"`
	Description     string  `json:"description"`
	SupplierName    string  `json:"supplier_name"`
	SalesPlan       float64 `json:"sales_plan"`
	PurchaseRequest float64 `json:"purchase_request"`
	Prl             float64 `json:"prl"`
}
