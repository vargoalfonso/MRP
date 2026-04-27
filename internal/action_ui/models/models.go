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
	DandoriTime    string
	SetupQCTime    string
	ScannedBy      string
	ScannedAt      time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Warehouse      string
}

type QCLog struct {
	ID           int64
	UUID         string
	QCTaskID     *int64
	WOID         *int64
	WOItemID     *int64
	DNItemID     *int64
	UniqCode     string
	QCRound      int
	QtyChecked   float64
	QtyPass      float64
	QtyDefect    float64
	QtyScrap     float64
	Status       string
	ProcessName  string
	DefectSource string
	CheckedBy    string
	CheckedAt    time.Time
	CreatedAt    time.Time
}

func (QCLog) TableName() string { return "qc_logs" }

type QCDefectItem struct {
	ID               int64
	QCLogID          int64
	QCTaskID         *int64
	WOID             *int64
	WOItemID         *int64
	DNItemID         *int64
	UniqCode         string
	DefectSource     string
	DefectReasonCode string
	DefectReasonText string
	QtyDefect        float64
	QtyScrap         float64
	IsRepairable     bool
	MachineID        *string
	ProcessName      string
	ReportedBy       string
	ReportedAt       time.Time
}

func (QCDefectItem) TableName() string { return "qc_defect_items" }

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

type DeliveryNoteItem struct {
	ID             int64      `json:"id" gorm:"primaryKey;column:id"`
	DNID           int64      `json:"dn_id" gorm:"column:dn_id;index"`             //foreign key ke delivery note
	ItemUniqCode   string     `json:"item_uniq_code" gorm:"column:item_uniq_code"` //request, diambil dari item_uniq_code di purchase order items bedasarkan po_id
	Quantity       int64      `json:"quantity" gorm:"column:quantity"`             //request, diambil dari ordered_qty di purchase order items bedasarkan po_id dan item_uniq_code
	UOM            string     `json:"uom" gorm:"column:uom"`                       //request, diambil dari uom di purchase order items bedasarkan po_id dan item_uniq_code
	Weight         int64      `json:"weight" gorm:"column:weight"`                 //request, diambil dari weight di purchase order items bedasarkan po_id dan item_uniq_code
	KanbanID       int64      `json:"kanban_id" gorm:"column:kanban_id"`           //request, diambil dari table kanban_parameter bedasarkan item_uniq_code
	QR             string     `json:"qr" gorm:"column:qr"`                         //hasil generate qr code yang berisi dn_number dan item_uniq_code, format datanya dn_number-item_uniq_code
	CreatedAt      time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt      time.Time  `json:"updated_at" gorm:"column:updated_at"`
	OrderQty       int64      `json:"order_qty" gorm:"column:order_qty"`             //request, diambil dari ordered_qty di purchase order items bedasarkan po_id dan item_uniq_code
	DateIncoming   *time.Time `json:"date_incoming" gorm:"column:date_incoming"`     //request, diisi ketika barang diterima
	QtyStated      int64      `json:"qty_stated" gorm:"column:qty_stated"`           //request, diisi ketika barang diterima, diambil dari quantity di delivery note item
	QtyReceived    int64      `json:"qty_received" gorm:"column:qty_received"`       //request, diisi ketika barang diterima, diambil dari quantity yang diterima di lapangan, bisa lebih kecil atau lebih besar dari qty_stated
	WeightReceived float64    `json:"weight_received" gorm:"column:weight_received"` //request, diisi ketika barang diterima, diambil dari weight yang diterima di lapangan, bisa lebih kecil atau lebih besar dari weight di delivery note item
	QualityStatus  string     `json:"quality_status" gorm:"column:quality_status"`   //request, diisi ketika barang diterima, bisa bernilai "good" atau "damaged"
	PcsPerKanban   int64      `json:"pcs_per_kanban" gorm:"column:pcs_per_kanban"`   //request, diambil dari pcs_per_kanban di purchase order items bedasarkan po_id dan item_uniq_code
	ReceivedAt     *time.Time `json:"received_at" gorm:"column:received_at"`         //request, diisi ketika barang diterima, berisi tanggal dan jam ketika barang diterima
	PackingNumber  string     `json:"packing_number" gorm:"column:packing_number"`   //request, diambil dari kanban number di kanban parameter bedasarkan item_uniq_code
	Check          string     `json:"check" gorm:"check"`                            //field untuk menampung nilai check ketika menerima barang,
	QtySent        int64      `json:"qty_sent" gorm:"qty_sent"`
}

type ProductionIssue struct {
	ID             int64      `db:"id" json:"id"`
	UUID           string     `db:"uuid" json:"uuid"`
	WOID           int64      `db:"wo_id" json:"wo_id"`
	WOItemID       int64      `db:"wo_item_id" json:"wo_item_id"`
	MachineID      int64      `db:"machine_id" json:"machine_id,omitempty"`
	ProcessName    string     `db:"process_name" json:"process_name"`
	ProductionLine string     `db:"production_line" json:"production_line"`
	IssueType      string     `db:"issue_type" json:"issue_type"`
	IssueDuration  int64      `db:"issue_duration" json:"issue_duration"` // menit
	QtyAffected    float64    `db:"qty_affected" json:"qty_affected"`
	ReportedBy     string     `db:"reported_by" json:"reported_by"`
	ReportedAt     time.Time  `db:"reported_at" json:"reported_at"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `db:"updated_at,omitempty" json:"updated_at,omitempty"`
	DeletedAt      *time.Time `db:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}
