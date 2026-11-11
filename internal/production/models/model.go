package models

import (
	"time"

	"github.com/google/uuid"
)

type ScanType string

const (
	ScanTypeIn  ScanType = "IN"
	ScanTypeOut ScanType = "OUT"
	ScanTypeQC  ScanType = "QC"
)

type ProductionScanLog struct {
	ID              int64      `gorm:"primaryKey"`
	UUID            uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();not null"`
	WOID            int64      `gorm:"column:wo_id;not null;index"`
	WOItemID        *int64     `gorm:"column:wo_item_id"`
	MachineNumber   string     `gorm:"column:machine_number;"`
	RawMaterialUUID *uuid.UUID `gorm:"column:raw_material_uuid;type:uuid"`
	KanbanNumber    string     `gorm:"column:kanban_number;size:64;index"`
	ProcessName     string     `gorm:"column:process_name;size:64;not null"`
	ProductionLine  string     `gorm:"column:production_line;size:64"`
	ScanType        ScanType   `gorm:"column:scan_type;size:10;not null;index"`
	QtyInput        float64    `gorm:"column:qty_input;type:numeric(15,4);default:0"`
	QtyOutput       float64    `gorm:"column:qty_output;type:numeric(15,4);default:0"`
	QtyRMUsed       float64    `gorm:"column:qty_rm_used;type:numeric(15,4);default:0"`
	NGMachine       float64    `gorm:"column:ng_machine;type:numeric(15,4);default:0"`
	NGProcess       float64    `gorm:"column:ng_process;type:numeric(15,4);default:0"`
	QtyScrap        float64    `gorm:"column:qty_scrap;type:numeric(15,4);default:0"`
	QtyRework       float64    `gorm:"column:qty_rework;type:numeric(15,4);default:0"`
	ScannedBy       string     `gorm:"column:scanned_by;size:255"`
	Shift           string     `gorm:"column:shift;size:10"`
	ScannedAt       time.Time  `gorm:"column:scanned_at;autoCreateTime"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (ProductionScanLog) TableName() string {
	return "production_scan_logs"
}

type ProductionScanResponse struct {
	WOID           int64   `json:"wo_id"`
	WONumber       string  `json:"wo_number"`
	ProcessName    string  `json:"process_name"`
	ProductionLine string  `json:"production_line"`
	MachineNumber  string  `json:"machine_number"`
	PackingNumber  string  `json:"packing_number"`
	KanbanNumber   string  `json:"kanban_number"`
	ProductName    string  `json:"product_name"`
	QtyPlan        float64 `json:"qty_plan"`
	Unit           string  `json:"unit"`
}

type WOItemDetail struct {
	WOID           int64
	WONumber       string
	ProcessName    string
	ProductionLine string
	MachineNumber  string

	PackingNumber string
	KanbanNumber  string

	ProductName string
	QtyPlan     float64
	Unit        string
}
