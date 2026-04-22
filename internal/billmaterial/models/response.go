package models

import "github.com/ganasa18/go-template/pkg/pagination"

// ---------------------------------------------------------------------------
// Asset info — tells the frontend what kind of file and whether CAD view works
// ---------------------------------------------------------------------------

// AssetInfo is embedded in every BOM row so the frontend knows
// whether to show the CAD viewer button or just an image preview.
//
//	asset_type : "photo" | "drawing" | "3d-model" | "other"
//	cad_viewable: true only for "3d-model" assets
//	label       : "3D Available" | "2D Available" | "-"
type AssetInfo struct {
	ID          *int64  `json:"id"`           // nil if no asset; use this as asset_id when replacing via upload
	URL         *string `json:"url"`          // nil if no asset
	AssetType   string  `json:"asset_type"`   // raw type from DB
	Label       string  `json:"label"`        // display label for the Drawing column
	CadViewable bool    `json:"cad_viewable"` // frontend uses this to show/hide CAD viewer button
}

// ---------------------------------------------------------------------------
// List BOM — tree row (matches the expandable table in the UI)
// ---------------------------------------------------------------------------

type BomTreeRow struct {
	ID         int64        `json:"id"`
	BomID      *int64       `json:"bom_id,omitempty"` // bom_item.id — hanya ada di parent row; gunakan untuk GET /products/bom/:bom_id
	LineID     *int64       `json:"line_id,omitempty"`
	UniqCode   string       `json:"uniq_code"`
	PartName   string       `json:"part_name"`
	PartNumber *string      `json:"part_number"`
	Model      *string      `json:"model"`
	Asset      AssetInfo    `json:"asset"`
	Level      interface{}  `json:"level"` // "Parent" | 1 | 2 | 3 | 4
	QPU        *float64     `json:"qpu"`   // nil for parent
	Version    *string      `json:"version"`
	Status     string       `json:"status"`
	Children   []BomTreeRow `json:"children"`
}

type ListBomResponse struct {
	Items      []BomTreeRow    `json:"items"`
	Pagination pagination.Meta `json:"pagination"` // page, limit, total, total_pages
}

// ---------------------------------------------------------------------------
// Detail BOM — full data for the CAD viewer sidebar + tree
// ---------------------------------------------------------------------------

type ProcessRouteDetail struct {
	RouteID       int64           `json:"route_id"`
	OpSeq         int             `json:"op_seq"`
	ProcessID     int64           `json:"process_id,omitempty"`
	ProcessName   string          `json:"process_name"`
	MachineID     *int64          `json:"machine_id,omitempty"`
	MachineName   *string         `json:"machine_name"`
	CycleTimeSec  *float64        `json:"cycle_time_sec"`
	SetupTimeMin  *float64        `json:"setup_time_min"`
	MachineStroke *string         `json:"machine_stroke"`
	ToolingRef    *string         `json:"tooling_ref"`
	Toolings      []ToolingDetail `json:"toolings"`
}

type ToolingDetail struct {
	ToolingType string  `json:"tooling_type"`
	ToolingCode *string `json:"tooling_code"`
	ToolingName string  `json:"tooling_name"`
}

type MaterialSpecDetail struct {
	MaterialGrade *string  `json:"material_grade"`
	Form          *string  `json:"form"`
	WidthMm       *float64 `json:"width_mm"`
	DiameterMm    *float64 `json:"diameter_mm"`
	ThicknessMm   *float64 `json:"thickness_mm"`
	LengthMm      *float64 `json:"length_mm"`
	WeightKg      *float64 `json:"weight_kg"`
	SupplierName  *string  `json:"supplier_name"`
	CycleTimeSec  *float64 `json:"cycle_time_sec"`
	SetupTimeMin  *float64 `json:"setup_time_min"`
}

type BomDetailChild struct {
	ID            int64                `json:"id"`
	LineID        int64                `json:"line_id"`
	UniqCode      string               `json:"uniq_code"`
	PartName      string               `json:"part_name"`
	PartNumber    *string              `json:"part_number"`
	Model         *string              `json:"model"`
	Level         int16                `json:"level"`
	QPU           float64              `json:"qty_per_uniq"`
	Version       *string              `json:"version"`
	Asset         AssetInfo            `json:"asset"`
	Status        string               `json:"status"`
	ProcessRoutes []ProcessRouteDetail `json:"process_routes"`
	MaterialSpec  *MaterialSpecDetail  `json:"material_spec"`
	Children      []BomDetailChild     `json:"children"`
}

type BomDetailResponse struct {
	BomID         int64                `json:"bom_id"`
	BomVersion    int                  `json:"bom_version"`
	BomStatus     string               `json:"bom_status"`
	IsCurrent     bool                 `json:"is_current"`
	ReadOnly      bool                 `json:"read_only"`
	ChangeNote    *string              `json:"change_note"`
	ID            int64                `json:"id"`
	UniqCode      string               `json:"uniq_code"`
	PartName      string               `json:"part_name"`
	PartNumber    *string              `json:"part_number"`
	Model         *string              `json:"model"`
	Version       *string              `json:"version"`
	Asset         AssetInfo            `json:"asset"`
	Status        string               `json:"status"`
	Description   *string              `json:"description"`
	ProcessRoutes []ProcessRouteDetail `json:"process_routes"`
	MaterialSpec  *MaterialSpecDetail  `json:"material_spec"`
	Children      []BomDetailChild     `json:"children"`
}

type BomVersionOption struct {
	BomID      int64   `json:"bom_id"`
	BomVersion int     `json:"bom_version"`
	Label      string  `json:"label"`
	BomStatus  string  `json:"bom_status"`
	IsCurrent  bool    `json:"is_current"`
	ReadOnly   bool    `json:"read_only"`
	ChangeNote *string `json:"change_note"`
	CreatedAt  string  `json:"created_at,omitempty"`
	CreatedBy  *string `json:"created_by,omitempty"`
}

type BomVersionsResponse struct {
	RootItemID     int64              `json:"root_item_id"`
	RootItemCode   string             `json:"root_item_code"`
	RootItemName   string             `json:"root_item_name"`
	CurrentBomID   *int64             `json:"current_bom_id"`
	CurrentVersion *int               `json:"current_version"`
	Versions       []BomVersionOption `json:"versions"`
}

type CreateBomRevisionResponse struct {
	SourceBomID   int64   `json:"source_bom_id"`
	SourceVersion int     `json:"source_version"`
	BomID         int64   `json:"bom_id"`
	BomVersion    int     `json:"bom_version"`
	BomStatus     string  `json:"bom_status"`
	IsCurrent     bool    `json:"is_current"`
	ReadOnly      bool    `json:"read_only"`
	ChangeNote    *string `json:"change_note"`
	Message       string  `json:"message"`
}

type ProcessRouteMutationResponse struct {
	RouteID       int64    `json:"route_id"`
	BomID         int64    `json:"bom_id"`
	BomVersion    int      `json:"bom_version"`
	LineID        *int64   `json:"line_id,omitempty"`
	OpSeq         int      `json:"op_seq"`
	ProcessID     int64    `json:"process_id"`
	ProcessName   string   `json:"process_name"`
	MachineID     *int64   `json:"machine_id"`
	MachineName   *string  `json:"machine_name"`
	CycleTimeSec  *float64 `json:"cycle_time_sec"`
	SetupTimeMin  *float64 `json:"setup_time_min"`
	MachineStroke *string  `json:"machine_stroke"`
	ToolingRef    *string  `json:"tooling_ref"`
}
