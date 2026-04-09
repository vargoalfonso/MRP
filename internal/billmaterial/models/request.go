package models

// ---------------------------------------------------------------------------
// Create BOM — single wizard call (Step 1 + Step 2 combined)
// POST /api/v1/products/bom
// ---------------------------------------------------------------------------

type ToolingInput struct {
	ToolingType string  `json:"tooling_type" validate:"required,oneof=Dies JIG CF Other"`
	ToolingCode *string `json:"tooling_code"`
	ToolingName string  `json:"tooling_name" validate:"required,max=255"`
}

type ProcessRouteInput struct {
	OpSeq         int            `json:"op_seq" validate:"required,min=10"`
	ProcessID     int64          `json:"process_id" validate:"required"`
	MachineID     *int64         `json:"machine_id"`
	CycleTimeSec  *float64       `json:"cycle_time_sec"`
	SetupTimeMin  *float64       `json:"setup_time_min"`
	MachineStroke *string        `json:"machine_stroke"`
	ToolingRef    *string        `json:"tooling_ref" validate:"omitempty,max=500"`
	Toolings      []ToolingInput `json:"toolings"`
}

type MaterialSpecInput struct {
	MaterialGrade *string  `json:"material_grade"`
	Form          *string  `json:"form" validate:"omitempty,oneof=Plate Coil Pipe Rod Wire Other"`
	WidthMm       *float64 `json:"width_mm"`
	DiameterMm    *float64 `json:"diameter_mm"`
	ThicknessMm   *float64 `json:"thickness_mm"`
	LengthMm      *float64 `json:"length_mm"`
	WeightKg      *float64 `json:"weight_kg"`
	SupplierID    *string  `json:"supplier_id" validate:"omitempty,uuid"`
	CycleTimeSec  *float64 `json:"cycle_time_sec"`
	SetupTimeMin  *float64 `json:"setup_time_min"`
}

// ChildInput — one child node, recursive up to level 4.
type ChildInput struct {
	// Existing item (reference by id) or new (provide uniq_code + part_name)
	ItemID     *int64  `json:"item_id"`
	UniqCode   *string `json:"uniq_code"`
	PartName   *string `json:"part_name"`
	PartNumber *string `json:"part_number"`
	UomID      *int64  `json:"uom_id"`

	// BOM line
	Level       int16    `json:"level" validate:"required,min=1,max=4"`
	QtyPerUniq  float64  `json:"qty_per_uniq" validate:"required,gt=0"`
	ScrapFactor *float64 `json:"scrap_factor"`
	IsPhantom   *bool    `json:"is_phantom"`

	// Optional revision label, picture, routing, material
	Revision      *string             `json:"revision"`
	PictureURL    *string             `json:"picture_url"`
	ProcessRoutes []ProcessRouteInput `json:"process_routes"`
	MaterialSpec  *MaterialSpecInput  `json:"material_spec"`

	Children []ChildInput `json:"children"`
}

// CreateBomRequest — body for POST /api/v1/products/bom
type CreateBomRequest struct {
	// Parent item fields
	UniqCode    string  `json:"uniq_code" validate:"required,max=64"`
	PartName    string  `json:"part_name" validate:"required,max=255"`
	PartNumber  *string `json:"part_number"`
	UomID       int64   `json:"uom_id" validate:"required"`
	Status      string  `json:"status" validate:"omitempty,oneof=Active Inactive"`
	Description *string `json:"description"`

	// Picture URL after upload
	PictureURL *string `json:"picture_url"`

	// Process routes for the parent
	ProcessRoutes []ProcessRouteInput `json:"process_routes"`

	// Material spec for the parent
	MaterialSpec *MaterialSpecInput `json:"material_spec"`

	// Child parts (levels 1–4, nested)
	Children []ChildInput `json:"children"`
}

// ---------------------------------------------------------------------------
// List / query params
// ---------------------------------------------------------------------------

type ListBomQuery struct {
	UniqCode       string
	Status         string
	Search         string // searches uniq_code + part_name
	Page           int    // default 1
	Limit          int    // default 20, max 200
	OrderBy        string
	OrderDirection string
}
