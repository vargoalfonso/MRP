package dto

import (
	"encoding/json"
	"time"
)

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
	RMUUID        string                         `json:"rm_uuid"`
	UniqCode      string                         `json:"uniq_code"`       // opsional: hasil scan RM
	PackingListRM string                         `json:"packing_list_rm"` // opsional
	Qty           float64                        `json:"qty"`
	Packings      []RawMaterialPackingAllocation `json:"packings"`
}

type RawMaterialPackingAllocation struct {
	PackingNumber string  `json:"packing_number"`
	Qty           float64 `json:"qty"`
}

type ScanInRequest struct {
	WOID             int64   `json:"wo_id" binding:"required"`
	WOItemID         int64   `json:"wo_item_id"`
	Uniq             string  `json:"uniq" binding:"required"`
	MachineID        string  `json:"machine_id"`      // optional
	ProductionLine   string  `json:"production_line"` // optional
	Qty              float64 `json:"qty" binding:"required"`
	Shift            string  `json:"shift" binding:"required"`
	DandoriTime      string  `json:"dandori_time"`  // optional
	SetupQCTime      string  `json:"setup_qc_time"` // optional
	ScannedBy        string  `json:"scanned_by"`    // dari user login / optional override
	ProductIssue     bool    `json:"product_issue"`
	ProductIssueType string  `json:"product_issue_type"`
	// [rm-source] detail bebas untuk jenis issue "Lainnya"
	ProductIssueNote     string `json:"product_issue_note"`
	ProductIssueDuration int64  `json:"product_issue_duration"`
	IsWIP                bool   `json:"is_wip"`

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

type CompleteWORequest struct {
	WOID int64 `json:"wo_id" binding:"required"`
}

// [qc-issue-table] satu baris issue (round 1/2) dengan qty
type QCIssueInput struct {
	Issue  string  `json:"issue"`
	Detail string  `json:"detail"`
	Qty    float64 `json:"qty"`
}

// [qc-issue-table] satu baris reason/info (round 3) dengan qty
type QCReasonInput struct {
	Info string  `json:"info"`
	Qty  float64 `json:"qty"`
}

type QCSubmitRequest struct {
	QCTaskID int64 `json:"qc_task_id" binding:"required"`

	WOID     int64 `json:"wo_id" binding:"required"`
	WOItemID int64 `json:"wo_item_id" binding:"required"`

	QCRound int `json:"qc_round" binding:"required"`

	// [qc-reason] binding:"required" dilepas supaya qty check 0 diterima
	QtyChecked float64 `json:"qty_checked"`
	QtyPass    float64 `json:"qty_pass"`
	QtyDefect  float64 `json:"qty_defect"`
	QtyScrap   float64 `json:"qty_scrap"`

	// [qc-reason] jenis issue + keterangan bebas dari round 1 / 2
	IssueType    string `json:"issue_type"`
	IssueNote    string `json:"issue_note"`
	DefectReason string `json:"defect_reason"`
	ScrapReason  string `json:"scrap_reason"`

	// [qc-issue-table] daftar issue per round (bisa lebih dari 1)
	Issues []QCIssueInput `json:"issues"`

	Status string `json:"status"`
}

type QCFinishRequest struct {
	QCTaskID int64 `json:"qc_task_id" binding:"required"`

	WOID     int64 `json:"wo_id" binding:"required"`
	WOItemID int64 `json:"wo_item_id" binding:"required"`

	TotalProductionQty float64 `json:"total_production_qty"`
	TotalScrapInBox    float64 `json:"total_scrap_in_box"`
	NGDefectQty        float64 `json:"ng_defect_qty"`

	// [qc-reason] keterangan defect (NG) / scrap dari round 3
	DefectReason string `json:"defect_reason"`
	ScrapReason  string `json:"scrap_reason"`

	// [qc-issue-table] daftar reason/info NG & scrap (bisa lebih dari 1)
	NGReasons    []QCReasonInput `json:"ng_reasons"`
	ScrapReasons []QCReasonInput `json:"scrap_reasons"`

	// [overflow-topup] true bila user pilih Batal di modal top-up: jangan isi
	// slot kanban existing, langsung buat kanban baru penuh.
	SkipOverflowTopUp bool `json:"skip_overflow_topup"`
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

// [qc-round-db] response GET /qc/rounds: ronde QC (Round 1 & 2) yang sudah
// tersubmit, direkonstruksi dari qc_logs + qc_defect_items agar bisa dibaca
// ulang lintas-gadget (bukan dari localStorage frontend).
type QCRoundIssue struct {
	Issue  string  `json:"issue"`
	Detail string  `json:"detail"`
	Qty    float64 `json:"qty"`
}

type QCRoundItem struct {
	Round        int            `json:"round"`
	QtyChecked   float64        `json:"qty_checked"`
	QtyPass      float64        `json:"qty_pass"`
	QtyDefect    float64        `json:"qty_defect"`
	QtyScrap     float64        `json:"qty_scrap"`
	DefectReason string         `json:"defect_reason"`
	ScrapReason  string         `json:"scrap_reason"`
	Issues       []QCRoundIssue `json:"issues"`
}

type WOListItem struct {
	WOID           int64   `json:"wo_id"`
	WONumber       string  `json:"wo_number"`
	Status         string  `json:"status"`
	WOType         string  `json:"wo_type"` // New | Assembly | Rework | Addendum
	Model          string  `json:"model"`
	PartName       string  `json:"part_name"`
	ProductionLine string  `json:"production_line"`
	TotalQty       float64 `json:"total_qty"`
	UniqCount      int64   `json:"uniq_count"`
}

type WODetailResponse struct {
	WOID           int64          `json:"wo_id"`
	WONumber       string         `json:"wo_number"`
	WOType         string         `json:"wo_type"`
	Status         string         `json:"status"`
	Model          string         `json:"model"`
	PartName       string         `json:"part_name"`
	ProductionLine string         `json:"production_line"`
	TotalQty       float64        `json:"total_qty"`
	UniqCount      int            `json:"uniq_count"`
	Uniqs          []WODetailUniq `json:"uniqs"`
}

type WODetailUniq struct {
	WOItemID       int64                 `json:"wo_item_id"`
	Uniq           string                `json:"uniq"`
	PartName       string                `json:"part_name"`
	PartNumber     string                `json:"part_number"`
	KanbanNumber   string                `json:"kanban_number"`
	UOM            string                `json:"uom"`
	Qty            float64               `json:"qty"`
	Status         string                `json:"status"`
	MachineID      string                `json:"machine_id"`
	MachineNumber  string                `json:"machine_number"`
	ProductionLine string                `json:"production_line"`
	ProcessName    string                `json:"process_name"`
	NextProcess    string                `json:"next_process"`
	CurrentStep    int                   `json:"current_step"`
	TotalStep      int                   `json:"total_step"`
	ScanInCount    int                   `json:"scan_in_count"`
	ScanOutCount   int                   `json:"scan_out_count"`
	MachineScanned bool                  `json:"machine_scanned"`
	ProcessFlow    []WODetailProcessStep `json:"process_flow"`

	SavedQty     float64                     `json:"saved_qty"`
	DandoriTime  string                      `json:"dandori_time"`
	SetupQCTime  string                      `json:"setup_qc_time"`
	RawMaterials []ScanOutContextRawMaterial `json:"raw_materials"`

	BomMaterials []BomMaterial `json:"bom_materials"`

	// [wip-source] material WIP dari proses sebelumnya (hanya untuk process >= 2).
	WIPMaterial *WIPMaterialDTO `json:"wip_material,omitempty"`
}

// WIPMaterialDTO = UNIQ semi-jadi hasil proses sebelumnya yang menjadi material
// input untuk proses saat ini (Poka Yoke multi-process).
type WIPMaterialDTO struct {
	Uniq         string  `json:"uniq"`
	ProcessName  string  `json:"process_name"`
	PrevProcess  string  `json:"prev_process"`
	OpSeq        int     `json:"op_seq"`
	QtyAvailable float64 `json:"qty_available"`
	UOM          string  `json:"uom"`
	PartName     string  `json:"part_name"`
}

type BomMaterial struct {
	Uniq       string  `json:"uniq"`
	PartName   string  `json:"part_name"`
	PartNumber string  `json:"part_number"`
	Level      int     `json:"level"` // 0 = parent (WO uniq), 1 = child, 2 = grandchild, dst.
	QtyPerUniq float64 `json:"qty_per_uniq"`
	UOM        string  `json:"uom"`

	MaterialGrade string   `json:"material_grade"`
	Grade         string   `json:"grade"`
	TypeMaterial  string   `json:"type_material"`
	Form          string   `json:"form"`
	WidthMm       *float64 `json:"width_mm"`
	DiameterMm    *float64 `json:"diameter_mm"`
	ThicknessMm   *float64 `json:"thickness_mm"`
	LengthMm      *float64 `json:"length_mm"`
	WeightKg      *float64 `json:"weight_kg"`
	SupplierName  string   `json:"supplier_name"`

	RMUUID string `json:"rm_uuid"`
	// [rm-source] uniq_code master Raw Material yang benar-benar dipakai
	// untuk stok (mis. BR50), berbeda dari Uniq item BOM (mis. M19).
	RMUniqCode      string  `json:"rm_uniq_code"`
	RawMaterialType string  `json:"raw_material_type"`
	TypeLabel       string  `json:"type_label"`
	InInventory     bool    `json:"in_inventory"`
	AvailableStock  float64 `json:"available_stock"`
	StockWeightKg   float64 `json:"stock_weight_kg"`
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
	// [wip-source] material berasal dari WIP UNIQ proses sebelumnya; stok RM tidak dipotong.
	IsWIP bool `json:"is_wip"`
	// [repacking] alokasi pengurangan qty per packing/kanban + hasil repack.
	Packings []ScanOutPacking `json:"packings"`
}

// ScanOutPacking = satu baris packing/kanban yang diatur lewat fitur Repacking.
//
//	DeductQty = jumlah yang diambil dari packing ini untuk scan out.
//	FinalQty  = qty_current packing setelah repack (nilai absolut, sudah final).
type ScanOutPacking struct {
	PackingNumber string  `json:"packing_number"`
	DNNumber      string  `json:"dn_number"`
	DeductQty     float64 `json:"deduct_qty"`
	FinalQty      float64 `json:"final_qty"`
	// [packing-deduct] true = qty packing dikurangi sebesar DeductQty
	// (alokasi packing dari Step 1, tanpa Repacking). false = FinalQty
	// dipakai sebagai nilai absolut hasil Repacking (perilaku lama).
	DeductOnly bool `json:"deduct_only"`
}

// RMPackingItem = satu packing/kanban milik satu Raw Material (uniq_code).
type RMPackingItem struct {
	DNNumber      *string `json:"dn_number"`
	PackingNumber string  `json:"packing_number"`
	Quantity      float64 `json:"quantity"`
	QtyCurrent    float64 `json:"qty_current"`
	QtyMax        float64 `json:"qty_max"`
	Progress      int     `json:"progress"`
	Status        *string `json:"status"`
	WONumber      *string `json:"wo_number"`
	Source        string  `json:"source"`
}

// RMPackingListResponse = balikan GET /production/rm-packing-list.
type RMPackingListResponse struct {
	UniqCode        string          `json:"uniq_code"`
	Items           []RMPackingItem `json:"items"`
	TotalPacking    int             `json:"total_packing"`
	TotalQtyCurrent float64         `json:"total_qty_current"`
	TotalQtyMax     float64         `json:"total_qty_max"`
}

// [repack-sisa] RMRepackMove = satu packing tujuan pemindahan sisa material.
type RMRepackMove struct {
	PackingNumber string  `json:"packing_number"`
	DNNumber      string  `json:"dn_number"`
	Qty           float64 `json:"qty"`
}

// RMRepackRequest = body POST /production/rm-repack.
// Sisa qty di SourcePackingNumber dipindahkan ke packing tujuan yang masih
// punya slot. Sisa yang tidak dipindah tetap tinggal di packing asal.
type RMRepackRequest struct {
	RMUUID              string         `json:"rm_uuid"`
	Code                string         `json:"code"`
	SourcePackingNumber string         `json:"source_packing_number"`
	Moves               []RMRepackMove `json:"moves"`
	ScannedBy           string         `json:"scanned_by"`
}

// RMRepackResponse = hasil repacking + daftar packing terbaru.
type RMRepackResponse struct {
	UniqCode string          `json:"uniq_code"`
	Moved    float64         `json:"moved"`
	Items    []RMPackingItem `json:"items"`
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
	RMUUID         string                         `json:"rm_uuid"`
	PackingListRM  string                         `json:"packing_list_rm"`
	MaterialCode   string                         `json:"material_code"`
	Form           string                         `json:"form"`
	QtyPerUniq     float64                        `json:"qty_per_uniq"`
	SpecWeightKg   float64                        `json:"spec_weight_kg"`
	TypeLabel      string                         `json:"type_label"`
	IsWIP          bool                           `json:"is_wip"`
	UOM            string                         `json:"uom"`
	AvailableStock float64                        `json:"available_stock"`
	StockWeightKg  float64                        `json:"stock_weight_kg"`
	QtyUsed        float64                        `json:"qty_used"`
	Packings       []RawMaterialPackingAllocation `json:"packings"`
}

type ScanOutContextItem struct {
	WOItemID       int64                       `json:"wo_item_id"`
	Uniq           string                      `json:"uniq"`
	MachineScanned bool                        `json:"machine_scanned"`
	Status         string                      `json:"status"`
	CurrentStep    int                         `json:"current_step"`
	TotalStep      int                         `json:"total_step"`
	ScanInCount    int                         `json:"scan_in_count"`
	ScanOutCount   int                         `json:"scan_out_count"`
	TotalOutput    float64                     `json:"total_output"`
	RawMaterials   []ScanOutContextRawMaterial `json:"raw_materials"`
}

type ScanOutContextResponse struct {
	WOID  int64                `json:"wo_id"`
	Items []ScanOutContextItem `json:"items"`
}

// ================================
// [scanin-draft-db] Draft Scan-In (seed) bersama lintas gadget
// ================================

// ScanInDraftItem = satu draft kanban (payload berisi ProductionScanOutSeed FE).
type ScanInDraftItem struct {
	WOItemID    int64           `json:"wo_item_id"`
	CurrentStep int             `json:"current_step"`
	Payload     json.RawMessage `json:"payload"`
	UpdatedBy   *string         `json:"updated_by,omitempty"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type ListScanInDraftsResponse struct {
	WOID  int64             `json:"wo_id"`
	Items []ScanInDraftItem `json:"items"`
}

type UpsertScanInDraftRequest struct {
	WOID        int64           `json:"wo_id" binding:"required"`
	WOItemID    int64           `json:"wo_item_id" binding:"required"`
	CurrentStep int             `json:"current_step"`
	Payload     json.RawMessage `json:"payload" binding:"required"`
}

type DeleteScanInDraftRequest struct {
	WOID        int64 `json:"wo_id" binding:"required"`
	WOItemID    int64 `json:"wo_item_id" binding:"required"`
	CurrentStep int   `json:"current_step"`
}


// [overflow-topup] QCOverflowTopUp satu rencana pengisian slot kanban existing.
type QCOverflowTopUp struct {
	WOItemID     int64   `json:"wo_item_id"`
	KanbanNumber string  `json:"kanban_number"`
	WONumber     string  `json:"wo_number"`
	FreeBefore   float64 `json:"free_before"`
	Fill         float64 `json:"fill"`
}

// [overflow-topup] QCOverflowPreview hasil PreviewQCOverflow (utk modal FE).
type QCOverflowPreview struct {
	MaxKanban    float64           `json:"max_kanban"`
	MainGood     float64           `json:"main_good"`
	Excess       float64           `json:"excess"`
	HasTopUp     bool              `json:"has_top_up"`
	TopUps       []QCOverflowTopUp `json:"top_ups"`
	NewKanbanQty float64           `json:"new_kanban_qty"`
}
