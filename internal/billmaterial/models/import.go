package models

// BomImportItemRow represents one row from the Items sheet.
type BomImportItemRow struct {
	SheetRow int
	RawData  []string

	BomGroup       string
	RowType        string
	UniqCode       string
	ParentUniqCode string
	PartName       string
	PartNumber     string
	Model          string
	Uom            string
	Level      int16
	QtyPerUniq float64
	Status     string
	Description    string

	MaterialGrade string
	Form          string
	WidthMM       *float64
	ThicknessMM   *float64
	LengthMM      *float64
	DiameterMM    *float64
	WeightKG      *float64
	SupplierName  string  // input dari Excel
	SupplierID    *string // resolved UUID setelah lookup ke DB
	CustomerCycle string

	// Inline route fields — index 0 = route 1, up to MaxBomRoutes (7) per item.
	ProcessCodes   []string
	MachineNumbers []string
	OpSeqs         []string
	CycleTimeSecs  []string
	SetupTimeMins  []string
	MachineStrokes []string
	ToolingRefs    []string
}

// BomImportRouteRow is the parsed representation of one process route for an item.
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
