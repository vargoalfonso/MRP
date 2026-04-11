package models

import "time"

// ---------------------------------------------------------------------------
// KPI Summary
// ---------------------------------------------------------------------------

// SummaryResponse is returned by GET /procurement/purchase-orders:summary.
type SummaryResponse struct {
	TotalPos        int64   `json:"total_pos"`
	ActiveSuppliers int64   `json:"active_suppliers"`
	TotalPoValue    float64 `json:"total_po_value"`
	LateDeliveries  int64   `json:"late_deliveries"`
}

// ---------------------------------------------------------------------------
// PO Board (list)
// ---------------------------------------------------------------------------

// POBoardItem is one row in the paginated PO board table.
type POBoardItem struct {
	PoID          int64   `json:"po_id"`
	PoType        string  `json:"po_type"`
	PoStage       *int    `json:"po_stage,omitempty"`
	Period        string  `json:"period"`
	PoNumber      string  `json:"po_number"`
	TotalBudgetPo float64 `json:"total_budget_po"`
	QtyDelivered  float64 `json:"qty_delivered"`
	UniqCode      *string `json:"uniq_code,omitempty"`
	SupplierID    *int64  `json:"supplier_id,omitempty"`
	SupplierName  *string `json:"supplier_name,omitempty"`
	Status        string  `json:"status"`
	IsLate        bool    `json:"is_late"`
}

// POBoardListResponse is the paginated response for the PO board.
type POBoardListResponse struct {
	Items      []POBoardItem  `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
}

// PaginationMeta is the standard pagination envelope.
type PaginationMeta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

// ---------------------------------------------------------------------------
// PO Detail
// ---------------------------------------------------------------------------

// PODetailResponse is returned by GET /procurement/purchase-orders/{po_id}.
type PODetailResponse struct {
	PO          POHeaderDetail `json:"po"`
	Items       []POItemDetail `json:"items"`
	HistoryLogs []POLogEntry   `json:"history_logs"`
}

// POHeaderDetail is the header section of PO detail.
type POHeaderDetail struct {
	PoID             int64   `json:"po_id"`
	PoType           string  `json:"po_type"`
	PoStage          *int    `json:"po_stage,omitempty"`
	Period           string  `json:"period"`
	PoNumber         string  `json:"po_number"`
	PoBudgetRef      string  `json:"po_budget_ref"`
	SalesPlan        float64 `json:"sales_plan"`
	TotalBudgetPo    float64 `json:"total_budget_po"`
	SupplierID       *int64  `json:"supplier_id,omitempty"`
	SupplierName     *string `json:"supplier_name,omitempty"`
	TotalQuantity    float64 `json:"total_quantity"`
	TotalUniq        int     `json:"total_uniq"`
	TotalWeight      float64 `json:"total_weight"`
	Status           string  `json:"status"`
	ExternalSystem   *string `json:"external_system,omitempty"`
	ExternalPoNumber *string `json:"external_po_number,omitempty"`
}

// POItemDetail is one line item in the PO detail.
type POItemDetail struct {
	ID           int64    `json:"id"`
	LineNo       int      `json:"line_no"`
	UniqCode     string   `json:"uniq_code"`
	PartNumber   *string  `json:"part_number,omitempty"`
	PartName     *string  `json:"part_name,omitempty"`
	Model        *string  `json:"model,omitempty"`
	Qty          float64  `json:"qty"`
	Uom          *string  `json:"uom,omitempty"`
	PcsPerKanban *int     `json:"pcs_per_kanban,omitempty"`
	WeightKg     *float64 `json:"weight_kg,omitempty"`
	Budget       float64  `json:"budget"`
	UnitPrice    *float64 `json:"unit_price,omitempty"`
	Amount       *float64 `json:"amount,omitempty"`
}

// POLogEntry is one history log entry.
type POLogEntry struct {
	Action     string    `json:"action"`
	Notes      *string   `json:"notes,omitempty"`
	Username   *string   `json:"username,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}

// ---------------------------------------------------------------------------
// DN List / Detail
// ---------------------------------------------------------------------------

// DNListItem is one row in the DN list.
type DNListItem struct {
	DnID      int64     `json:"dn_id"`
	DnNumber  string    `json:"dn_number"`
	Period    string    `json:"period"`
	PoNumber  string    `json:"po_number"`
	DnType    string    `json:"dn_type"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// DNListResponse is the paginated DN list.
type DNListResponse struct {
	Items      []DNListItem   `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
}

// DNDetailResponse is returned by GET /procurement/incoming-dns/{dn_id}.
type DNDetailResponse struct {
	DnID            int64          `json:"dn_id"`
	DnNumber        string         `json:"dn_number"`
	Period          string         `json:"period"`
	PoNumber        string         `json:"po_number"`
	DnType          string         `json:"dn_type"`
	SupplierID      *int64         `json:"supplier_id,omitempty"`
	SupplierName    *string        `json:"supplier_name,omitempty"`
	TotalPoQty      *int           `json:"total_po_qty,omitempty"`
	TotalDnCreated  *int           `json:"total_dn_created,omitempty"`
	TotalDnIncoming *int           `json:"total_dn_incoming,omitempty"`
	Status          string         `json:"status"`
	CreatedAt       time.Time      `json:"created_at"`
	Items           []DNItemDetail `json:"items"`
}

// DNItemDetail is one line item in a DN.
type DNItemDetail struct {
	ID            int64     `json:"id"`
	ItemUniqCode  string    `json:"item_uniq_code"`
	OrderQty      int       `json:"order_qty"`
	QtyStated     int       `json:"qty_stated"`
	QtyReceived   int       `json:"qty_received"`
	QualityStatus string    `json:"quality_status"`
	DateIncoming  time.Time `json:"date_incoming"`
	Uom           *string   `json:"uom,omitempty"`
}

// ---------------------------------------------------------------------------
// Form Options (wizard dropdown data)
// ---------------------------------------------------------------------------

// FormOptionsResponse is returned by GET /procurement/purchase-orders:form_options.
type FormOptionsResponse struct {
	PoStageOptions  []PoStageOption    `json:"po_stage_options"`
	SplitSetting    SplitSettingOption `json:"split_setting"`
	SupplierOptions []SupplierOption   `json:"supplier_options"`
	PoBudgetOptions []PoBudgetOption   `json:"po_budget_options"`
}

// PoStageOption is one entry in the po_stage dropdown.
type PoStageOption struct {
	Stage int    `json:"stage"`
	Label string `json:"label"`
}

// SplitSettingOption shows the effective PO1/PO2 percentages.
type SplitSettingOption struct {
	Po1Pct float64 `json:"po1_pct"`
	Po2Pct float64 `json:"po2_pct"`
}

// SupplierOption is one supplier in the dropdown (legacy BIGINT ID).
type SupplierOption struct {
	SupplierID   int64  `json:"supplier_id"`
	SupplierName string `json:"supplier_name"`
}

// PoBudgetOption represents an available budget batch for the wizard.
// PoBudgetRef is server-generated: "POB-{YYYY}-{TYPE}-{id}".
type PoBudgetOption struct {
	PoBudgetID    int64   `json:"po_budget_id"`
	PoBudgetRef   string  `json:"po_budget_ref"`
	Period        string  `json:"period"`
	TotalBudgetPo float64 `json:"total_budget_po"`
	TotalQuantity float64 `json:"total_quantity"`
	TotalUniq     int     `json:"total_uniq"`
}

// ---------------------------------------------------------------------------
// Generate PO response
// ---------------------------------------------------------------------------

// GeneratePOResponse is returned by POST /procurement/purchase-orders:generate.
type GeneratePOResponse struct {
	Pos []GeneratedPOGroup `json:"pos"`
}

// GeneratedPOGroup groups one stage (PO1 or PO2) with its header and items.
type GeneratedPOGroup struct {
	Stage int            `json:"stage"`
	PO    POHeaderDetail `json:"po"`
	Items []POItemDetail `json:"items"`
}
