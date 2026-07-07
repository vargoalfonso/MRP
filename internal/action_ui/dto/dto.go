package dto

import "time"

type ScanContextResponse struct {
	WOID           int64   `json:"wo_id"`
	WONumber       string  `json:"wo_number"`
	Uniq           string  `json:"uniq"`
	KanbanNumber   string  `json:"kanban_number"`
	PartName       string  `json:"part_name"`
	PartNumber     string  `json:"part_number"`
	Model          string  `json:"model"`
	UOM            string  `json:"uom"`
	MachineID      string  `json:"machine_id"`
	ProductionLine string  `json:"production_line"`
	ProcessName    string  `json:"process_name"`
	NextProcess    string  `json:"next_process"`
	CurrentStep    int     `json:"current_step"`
	TotalStep      int     `json:"total_step"`
	CurrentQCStep  int64   `json:"current_qc_step"`
	TotalQCStep    int     `json:"total_qc_step"`
	DefaultQty     float64 `json:"default_qty"`
	Status         string  `json:"status"`
	// RawMaterials   []ScanContextRawMaterial `json:"raw_materials"`
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
	RMUUID        string  `json:"rm_uuid"`
	UniqCode      string  `json:"uniq_code"`       // opsional: hasil scan RM
	PackingListRM string  `json:"packing_list_rm"` // opsional
	Qty           float64 `json:"qty"`
}

type ScanInRequest struct {
	WOID                 int64   `json:"wo_id" binding:"required"`
	WOItemID 			 int64   `json:"wo_item_id"`
	Uniq                 string  `json:"uniq" binding:"required"`
	MachineID            string  `json:"machine_id"`      // optional
	ProductionLine       string  `json:"production_line"` // optional
	Qty                  float64 `json:"qty" binding:"required"`
	Shift                string  `json:"shift" binding:"required"`
	DandoriTime          string  `json:"dandori_time"`  // optional
	SetupQCTime          string  `json:"setup_qc_time"` // optional
	ScannedBy            string  `json:"scanned_by"`    // dari user login / optional override
	ProductIssue         bool    `json:"product_issue"`
	ProductIssueType     string  `json:"product_issue_type"`
	ProductIssueDuration int64   `json:"product_issue_duration"`
	IsWIP                bool    `json:"is_wip"`

	RawMaterials []RawMaterialInput `json:"raw_materials"`
}

type ScanOutRequest struct {
	WOID           int64   `json:"wo_id" binding:"required"`
	WOItemID       int64   `json:"wo_item_id" binding:"required"`
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

	TotalProduction float64              `json:"total_production"`
	RawMaterials    []ScanOutRawMaterial `json:"raw_materials"`
}

type QCSubmitRequest struct {
	QCTaskID int64 `json:"qc_task_id" binding:"required"`

	WOID     int64 `json:"wo_id" binding:"required"`
	WOItemID int64 `json:"wo_item_id" binding:"required"`

	QCRound int `json:"qc_round" binding:"required"`

	QtyChecked float64 `json:"qty_checked" binding:"required"`
	QtyPass    float64 `json:"qty_pass"`
	QtyDefect  float64 `json:"qty_defect"`
	QtyScrap   float64 `json:"qty_scrap"`

	Status string `json:"status"`
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

type ListQCTaskRequest struct {
	Page     int    `form:"page"`
	Limit    int    `form:"limit"`
	Status   string `form:"status"`    // pending / done
	TaskType string `form:"task_type"` // production_qc
	Search   string `form:"search"`    // kanban / uniq / process
}

type QCTaskListItem struct {
	ID       int64  `json:"id"`
	TaskType string `json:"task_type"`
	Status   string `json:"status"`
	Round    int    `json:"round"`

	WOID     *int64 `json:"wo_id"`
	WOItemID *int64 `json:"wo_item_id"`
	WONumber string `json:"wo_number"`

	Uniq         string  `json:"uniq"`
	KanbanNumber string  `json:"kanban_number"`
	ProcessName  string  `json:"process_name"`
	Qty          float64 `json:"qty"`

	CreatedAt time.Time `json:"created_at"`
}

type WOListItem struct {
	WOID           int64   `json:"wo_id"`
	WONumber       string  `json:"wo_number"`
	Status         string  `json:"status"`
	Model          string  `json:"model"`
	PartName       string  `json:"part_name"`
	ProductionLine string  `json:"production_line"`
	TotalQty       float64 `json:"total_qty"`
	UniqCount      int64   `json:"uniq_count"`
}

type WODetailResponse struct {
	WOID           int64          `json:"wo_id"`
	WONumber       string         `json:"wo_number"`
	Status         string         `json:"status"`
	Model          string         `json:"model"`
	PartName       string         `json:"part_name"`
	ProductionLine string         `json:"production_line"`
	TotalQty       float64        `json:"total_qty"`
	UniqCount      int            `json:"uniq_count"`
	Uniqs          []WODetailUniq `json:"uniqs"`
}

type WODetailUniq struct {
	WOItemID       int64   `json:"wo_item_id"`
	Uniq           string  `json:"uniq"`
	PartName       string  `json:"part_name"`
	PartNumber     string  `json:"part_number"`
	KanbanNumber   string  `json:"kanban_number"`
	UOM            string  `json:"uom"`
	Qty            float64 `json:"qty"`
	Status         string  `json:"status"`
	MachineID      string  `json:"machine_id"`
	MachineNumber  string  `json:"machine_number"`
	ProductionLine string  `json:"production_line"`
	ProcessName string     `json:"process_name"`
	NextProcess string     `json:"next_process"`
	CurrentStep int        `json:"current_step"`
	TotalStep   int        `json:"total_step"`
	ScanInCount    int       `json:"scan_in_count"`
	ScanOutCount   int       `json:"scan_out_count"`
	MachineScanned bool      `json:"machine_scanned"`
	ProcessFlow []WODetailProcessStep `json:"process_flow"`

	SavedQty     float64                     `json:"saved_qty"`
	DandoriTime  string                      `json:"dandori_time"`
	SetupQCTime  string                      `json:"setup_qc_time"`
	RawMaterials []ScanOutContextRawMaterial `json:"raw_materials"`
}

type RawMaterialLookupResponse struct {
	RMID              int64   `json:"rm_id"`
	RMUUID            string  `json:"rm_uuid"`
	UniqCode          string  `json:"uniq_code"`
	PartNumber        string  `json:"part_number"`
	PartName          string  `json:"part_name"`
	RawMaterialType   string  `json:"raw_material_type"` 
	TypeLabel         string  `json:"type_label"`        
	UOM               string  `json:"uom"`
	AvailableStock    float64 `json:"available_stock"`
	StockWeightKg     float64 `json:"stock_weight_kg"`
	WarehouseLocation string  `json:"warehouse_location"`
}

type ScanOutRawMaterial struct {
	RMUUID        string  `json:"rm_uuid"`
	UniqCode      string  `json:"uniq_code"`
	PackingListRM string  `json:"packing_list_rm"`
	QtyUsed       float64 `json:"qty_used"`
}

// Langkah proses lengkap untuk Poka Yoke (Step 1)
type WODetailProcessStep struct {
	OpSeq       int    `json:"op_seq"`
	ProcessName string `json:"process_name"`
	MachineName string `json:"machine_name"`
	Status      string `json:"status"`
}

// Request/response validasi scan mesin
type ScanMachineRequest struct {
	WOID     int64  `json:"wo_id"`
	WOItemID int64  `json:"wo_item_id"`
	Machine  string `json:"machine"`
}

type ScanMachineResponse struct {
	Valid           bool                  `json:"valid"`
	Message         string                `json:"message"`
	MachineID       string                `json:"machine_id"`
	MachineNumber   string                `json:"machine_number"`
	MachineName     string                `json:"machine_name"`
	ProductionLine  string                `json:"production_line"`
	ScannedProcess  string                `json:"scanned_process"`
	ExpectedProcess string                `json:"expected_process"`
	CurrentStep     int                   `json:"current_step"`
	TotalStep       int                   `json:"total_step"`
	NextProcess     string                `json:"next_process"`
	ProcessFlow     []WODetailProcessStep `json:"process_flow"`
}

type ScanOutContextRawMaterial struct {
	RMUUID         string  `json:"rm_uuid"`
	PackingListRM  string  `json:"packing_list_rm"`
	TypeLabel      string  `json:"type_label"`
	IsWIP          bool    `json:"is_wip"`
	UOM            string  `json:"uom"`
	AvailableStock float64 `json:"available_stock"`
	StockWeightKg  float64 `json:"stock_weight_kg"`
	QtyUsed        float64 `json:"qty_used"`
}

type ScanOutContextItem struct {
	WOItemID       int64                       `json:"wo_item_id"`
	Uniq           string                      `json:"uniq"`
	MachineScanned bool                        `json:"machine_scanned"`
	ScanInCount    int                         `json:"scan_in_count"`
	ScanOutCount   int                         `json:"scan_out_count"`
	RawMaterials   []ScanOutContextRawMaterial `json:"raw_materials"`
}

type ScanOutContextResponse struct {
	WOID  int64                `json:"wo_id"`
	Items []ScanOutContextItem `json:"items"`
}