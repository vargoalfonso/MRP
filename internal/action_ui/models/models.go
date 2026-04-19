package models

import "time"

type WorkOrder struct {
	ID             int64
	UUID           string
	WONumber       string
	WOType         string
	ReferenceWO    string
	Status         string
	ApprovalStatus string
	CreatedDate    time.Time
	TargetDate     time.Time
	ScanStartDate  *time.Time
	CloseDate      *time.Time
	OperatorName   string
	Notes          string
	QRImageBase64  string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CreatedBy      string
	CreatedByName  string
	WOKind         string
	SourceMaterial string
	TargetMaterial string
	Model          string
	GradeSize      string
	InputQty       float64
	InputUOM       string
	OutputQty      float64
	OutputUOM      string
	DateIssued     *time.Time
	DateCompleted  *time.Time
	CycleTimeDays  int
	Remarks        string
}

type WorkOrderItem struct {
	ID                 int64
	UUID               string
	WOID               int64
	ItemUniqCode       string
	PartName           string
	PartNumber         string
	UOM                string
	Quantity           float64
	ProcessName        string
	KanbanNumber       string
	Status             string
	MachineID          int
	ProcessFlowJSON    string
	CurrentStepSeq     int
	LastScannedProcess string
	ScanInCount        int
	ScanOutCount       int
	TotalGoodQty       float64
	TotalNGQty         float64
	TotalScrapQty      float64
	Model              string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type ProductionScanLog struct {
	ID             int64
	UUID           string
	WOID           int64
	WOItemID       int64
	MachineID      int64
	RawMaterialID  *int64
	KanbanNumber   string
	ProcessName    string
	ProductionLine string
	ScanType       string
	QtyInput       float64
	QtyOutput      float64
	QtyRMUsed      float64
	NGMachine      float64
	NGProcess      float64
	QtyScrap       float64
	QtyRework      float64
	Shift          string
	DandoriTime    float64
	SetupQCTime    float64
	ScannedBy      string
	ScannedAt      time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type QCLog struct {
	ID         int64
	UUID       string
	WOID       int64
	WOItemID   int64
	UniqCode   string
	QCRound    int
	QtyChecked float64
	QtyPass    float64
	QtyDefect  float64
	QtyScrap   float64
	Status     string
	CheckedBy  string
	CheckedAt  time.Time
	CreatedAt  time.Time
}

type FinishedGoods struct {
	ID                    int64
	UUID                  string
	UniqCode              string
	ItemID                int64
	PartNumber            string
	PartName              string
	Model                 string
	WONumber              string
	WarehouseLocation     string
	StockQty              float64
	UOM                   string
	KanbanCount           int
	KanbanStandardQty     float64
	MinThreshold          float64
	MaxThreshold          float64
	SafetyStockQty        float64
	StockToCompleteKanban float64
	Status                string
	CreatedBy             string
	CreatedAt             time.Time
	UpdatedBy             string
	UpdatedAt             time.Time
	DeletedAt             *time.Time
}

type ProcessFlow struct {
	OpSeq        int     `json:"op_seq"`
	ProcessName  string  `json:"process_name"`
	MachineName  *string `json:"machine_name"`
	CycleTimeSec int     `json:"cycle_time_sec"`
	SetupTimeMin int     `json:"setup_time_min"`
}

type RawMaterialLog struct {
	ID         int64
	UUID       string
	WOID       int64
	WOItemID   int64
	UniqCode   string
	RMUUID     string
	PartNumber string
	PartName   string
	UOM        string
	QtyUsed    float64
	ScannedBy  string
	ScannedAt  time.Time
	CreatedAt  time.Time
}

type MasterMachine struct {
	ID              int `gorm:"primaryKey;autoIncrement"`
	MachineNumber   string
	MachineName     string
	ProductionLine  string
	ProcessID       *int64
	MachineCapacity *int
	QRImageBase64   *string `gorm:"column:qr_image_base64;type:text"`
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
