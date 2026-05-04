package models

// BomImportItemRow represents one row from sheet "Items".
type BomImportItemRow struct {
	SheetRow int
	RawData  []string

	BomGroup       string
	RowType        string
	UniqCode       string
	ParentUniqCode string
	PartName       string
	PartNumber     string
	Uom            string
	Level          int16
	QtyPerUniq     float64
	ScrapFactor    float64
	IsPhantom      bool
	Status         string
	Description    string

	MaterialGrade string
	Form          string
	WidthMM       *float64
	ThicknessMM   *float64
	LengthMM      *float64
	DiameterMM    *float64
	WeightKG      *float64
}

// BomImportRouteRow represents one row from sheet "Routes".
type BomImportRouteRow struct {
	SheetRow int
	RawData  []string

	UniqCode      string
	OpSeq         int
	ProcessID     int64
	MachineID     *int64
	CycleTimeSec  *float64
	SetupTimeMin  *float64
	MachineStroke *string
	ToolingRef    *string
}
