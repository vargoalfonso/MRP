package dto

type ScanContextResponse struct {
	WOID           int64                    `json:"wo_id"`
	WONumber       string                   `json:"wo_number"`
	Uniq           string                   `json:"uniq"`
	KanbanNumber   string                   `json:"kanban_number"`
	PartName       string                   `json:"part_name"`
	PartNumber     string                   `json:"part_number"`
	Model          string                   `json:"model"`
	UOM            string                   `json:"uom"`
	MachineID      string                   `json:"machine_id"`
	ProductionLine string                   `json:"production_line"`
	ProcessName    string                   `json:"process_name"`
	NextProcess    string                   `json:"next_process"`
	CurrentStep    int                      `json:"current_step"`
	TotalStep      int                      `json:"total_step"`
	DefaultQty     float64                  `json:"default_qty"`
	Status         string                   `json:"status"`
	RawMaterials   []ScanContextRawMaterial `json:"raw_materials"`
}

type ScanContextRawMaterial struct {
	Uniq           string  `json:"uniq"`
	PartName       string  `json:"part_name"`
	PartNumber     string  `json:"part_number"`
	UOM            string  `json:"uom"`
	StandardQty    float64 `json:"standard_qty"`
	AvailableStock float64 `json:"available_stock"`
	Qty            float64 `json:"qty"`
	ProcessName    string  `json:"process_name"`
}

type RawMaterialInput struct {
	RMUUID string  `json:"rm_uuid"`
	Qty    float64 `json:"qty"`
}

type ScanInRequest struct {
	WOID                 int64   `json:"wo_id" binding:"required"`
	Uniq                 string  `json:"uniq" binding:"required"`
	MachineID            string  `json:"machine_id"`      // optional
	ProductionLine       string  `json:"production_line"` // optional
	Qty                  float64 `json:"qty" binding:"required"`
	Shift                string  `json:"shift" binding:"required"`
	DandoriTime          float64 `json:"dandori_time"`  // optional
	SetupQCTime          float64 `json:"setup_qc_time"` // optional
	ScannedBy            string  `json:"scanned_by"`    // dari user login / optional override
	ProductIssue         bool    `json:"product_issue"`
	ProductIssueType     string  `json:"product_issue_type"`
	ProductIssueDuration int64   `json:"product_issue_duration"`
}

type ScanOutRequest struct {
	WOID           int64   `json:"wo_id" binding:"required"`
	Uniq           string  `json:"uniq"`
	MachineID      string  `json:"machine_id"`      // optional
	ProductionLine string  `json:"production_line"` // optional
	QtyOutput      float64 `json:"qty_output" binding:"required"`
	QtyRMUsed      float64 `json:"qty_rm_used"` // optional
	NGMachine      float64 `json:"ng_machine"`
	NGProcess      float64 `json:"ng_process"`
	QtyScrap       float64 `json:"qty_scrap"`
	QtyRework      float64 `json:"qty_rework"`
	Shift          string  `json:"shift"`
	ScannedBy      string  `json:"scanned_by"`
	Warehouse      string  `json:"warehouse"`
}

type QCSubmitRequest struct {
	UUID         string          `json:"uuid"`
	Uniq         string          `json:"uniq"`
	QCRound      int             `json:"qc_round"`
	QtyChecked   float64         `json:"qty_checked"`
	QtyPass      float64         `json:"qty_pass"`
	QtyDefect    float64         `json:"qty_defect"`
	QtyScrap     float64         `json:"qty_scrap"`
	Status       string          `json:"status"`
	DefectSource string          `json:"defect_source"`
	Defects      []QCDefectInput `json:"defects"`
}

type QCDefectInput struct {
	ReasonCode   string  `json:"reason_code"`
	ReasonText   string  `json:"reason_text"`
	QtyDefect    float64 `json:"qty_defect"`
	QtyScrap     float64 `json:"qty_scrap"`
	IsRepairable bool    `json:"is_repairable"`
}

type FinishedGoodsResponse struct {
	UniqCode   string  `json:"uniq_code"`
	PartNumber string  `json:"part_number"`
	PartName   string  `json:"part_name"`
	Model      string  `json:"model"`
	WONumber   string  `json:"wo_number"`
	StockQty   float64 `json:"stock_qty"`
	UOM        string  `json:"uom"`
	Status     string  `json:"status"`
}

type IncomingScanRequest struct {
	DNItemID       int64
	ScanRef        string
	Qty            float64
	WeightKg       float64
	Warehouse      string
	ScannedBy      string
	IdempotencyKey string
}
