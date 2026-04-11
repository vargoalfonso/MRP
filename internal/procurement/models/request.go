package models

// ---------------------------------------------------------------------------
// Generate PO request
// ---------------------------------------------------------------------------

// GeneratePORequest is the body for POST /procurement/purchase-orders:generate.
//
// Multi-supplier behaviour:
//   - If supplier_id is set → generate PO for that specific supplier only.
//   - If supplier_id is 0 / omitted AND generate_mode = "bulk_all_suppliers"
//     → backend groups po_budget_entries by supplier and creates one PO per supplier
//     (per stage when generate_mode includes both stages).
//
// Type-safety:
//   - All po_budget_entries resolved from po_budget_entry_ids MUST have the same
//     budget_type that maps to po_type.  Any mismatch returns 422.
//   - You cannot mix RM/INDIRECT/SUBCON in a single generate call.
type GeneratePORequest struct {
	// PoType must be one of: raw_material | indirect | subcon.
	PoType string `json:"po_type" binding:"required,oneof=raw_material indirect subcon"`

	// Period in YYYY-MM format, e.g. "2025-10".
	Period string `json:"period" binding:"required"`

	// PoBudgetEntryIDs selects which po_budget_entries rows to use.
	// Supplier is resolved automatically from the budget entries.
	PoBudgetEntryIDs []int64 `json:"po_budget_entry_ids"`

	// ExternalSystem + ExternalPoNumber: referensi PO di sistem eksternal (misal Zahir). Opsional.
	ExternalSystem   string `json:"external_system"`
	ExternalPoNumber string `json:"external_po_number"`

	// GenerateMode controls which stages are created:
	//   "both_stages"       → PO1 + PO2 (default)
	//   "stage_only"        → only the stage specified in Stage field
	//   "bulk_all_suppliers"→ PO1+PO2 for every supplier in the budget entries
	GenerateMode string `json:"generate_mode"` // both_stages | stage_only | bulk_all_suppliers

	// Stage is only used when GenerateMode = "stage_only".
	Stage int `json:"stage"` // 1 or 2
}

// ---------------------------------------------------------------------------
// DN list filter (for GET /procurement/incoming-dns)
// ---------------------------------------------------------------------------

// DNListFilter holds query-param filters for the DN list.
type DNListFilter struct {
	PoNumber string
	Period   string
	Status   string
	Page     int
	Limit    int
}
