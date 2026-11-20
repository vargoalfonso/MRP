// Package models defines response structs for the Production Dashboard module.
//
// The Production Dashboard aggregates data from production_scan_logs (scan IN /
// OUT events recorded on the shop floor) joined with work_orders and
// work_order_items. Each endpoint returns the same envelope shape expected by
// the frontend: { summary_cards, table_data }.
package models

import "time"

// SummaryCards holds the 4 KPI numbers shown on top of every dashboard view.
type SummaryCards struct {
	FGOutput    float64 `gorm:"column:fg_output" json:"fg_output"`
	WIPOutput   float64 `gorm:"column:wip_output" json:"wip_output"`
	TotalNG     float64 `gorm:"column:total_ng" json:"total_ng"`
	TotalRework float64 `gorm:"column:total_rework" json:"total_rework"`
}

// DashboardPayload is the standard data envelope for a dashboard view.
type DashboardPayload struct {
	SummaryCards SummaryCards `json:"summary_cards"`
	TableData    interface{}  `json:"table_data"`
}

// FinishedGoodsRow is one aggregated row for the Finished Goods view.
type FinishedGoodsRow struct {
	ID          string     `gorm:"column:id" json:"id"`
	ReportDate  *time.Time `gorm:"column:report_date" json:"report_date"`
	Uniq        string     `gorm:"column:uniq" json:"uniq"`
	ProductName string     `gorm:"column:product_name" json:"product_name"`
	WONumber    string     `gorm:"column:wo_number" json:"wo_number"`
	Shift       string     `gorm:"column:shift" json:"shift"`
	FGOutput    float64    `gorm:"column:fg_output" json:"fg_output"`
	NGSetting   float64    `gorm:"column:ng_setting" json:"ng_setting"`
	NGProcess   float64    `gorm:"column:ng_process" json:"ng_process"`
	Rework      float64    `gorm:"column:rework" json:"rework"`
	Scrap       float64    `gorm:"column:scrap" json:"scrap"`
}

// WipRow is one aggregated row for the WIP view.
type WipRow struct {
	ID          string     `gorm:"column:id" json:"id"`
	ReportDate  *time.Time `gorm:"column:report_date" json:"report_date"`
	Uniq        string     `gorm:"column:uniq" json:"uniq"`
	ProductName string     `gorm:"column:product_name" json:"product_name"`
	ProcessName string     `gorm:"column:process_name" json:"process_name"`
	WONumber    string     `gorm:"column:wo_number" json:"wo_number"`
	Shift       string     `gorm:"column:shift" json:"shift"`
	WipOutput   float64    `gorm:"column:wip_output" json:"wip_output"`
	NGSetting   float64    `gorm:"column:ng_setting" json:"ng_setting"`
	NGProcess   float64    `gorm:"column:ng_process" json:"ng_process"`
	Rework      float64    `gorm:"column:rework" json:"rework"`
	Scrap       float64    `gorm:"column:scrap" json:"scrap"`
}

// OutputMachineRow is one aggregated row for the Output Per Machine view.
type OutputMachineRow struct {
	ID            string     `gorm:"column:id" json:"id"`
	ReportDate    *time.Time `gorm:"column:report_date" json:"report_date"`
	LineProcess   string     `gorm:"column:line_process" json:"line_process"`
	MachineName   string     `gorm:"column:machine_name" json:"machine_name"`
	Uniq          string     `gorm:"column:uniq" json:"uniq"`
	Shift         string     `gorm:"column:shift" json:"shift"`
	WONumber      string     `gorm:"column:wo_number" json:"wo_number"`
	ProductOutput float64    `gorm:"column:product_output" json:"product_output"`
	NGSetting     float64    `gorm:"column:ng_setting" json:"ng_setting"`
	NGProcess     float64    `gorm:"column:ng_process" json:"ng_process"`
	Rework        float64    `gorm:"column:rework" json:"rework"`
	Scrap         float64    `gorm:"column:scrap" json:"scrap"`
}

// SummaryStrokeRow is one aggregated row (per production line / day) for the
// Summary Stroke view.
type SummaryStrokeRow struct {
	ID               string     `gorm:"column:id" json:"id"`
	ReportDate       *time.Time `gorm:"column:report_date" json:"report_date"`
	ProductionLine   string     `gorm:"column:production_line" json:"production_line"`
	Stroke           float64    `gorm:"column:stroke" json:"stroke"`
	ProductionOutput float64    `gorm:"column:production_output" json:"production_output"`
	MachineTimeMin   float64    `gorm:"column:machine_time_min" json:"machine_time_min"`
	DandoriTimeMin   float64    `gorm:"column:dandori_time_min" json:"dandori_time_min"`
	SetQCTimeMin     float64    `gorm:"column:set_qc_time_min" json:"set_qc_time_min"`
}

// RuntimeRow is one aggregated row (per WO / machine / day) for the Runtime view.
type RuntimeRow struct {
	ID                  string     `gorm:"column:id" json:"id"`
	ReportDate          *time.Time `gorm:"column:report_date" json:"report_date"`
	WONumber            string     `gorm:"column:wo_number" json:"wo_number"`
	ProductionLine      string     `gorm:"column:production_line" json:"production_line"`
	MachineNumber       string     `gorm:"column:machine_number" json:"machine_number"`
	TotalMachineTimeMin float64    `gorm:"column:total_machine_time_min" json:"total_machine_time_min"`
	DandoriTimeMin      float64    `gorm:"column:dandori_time_min" json:"dandori_time_min"`
	SetQCTimeMin        float64    `gorm:"column:set_qc_time_min" json:"set_qc_time_min"`
}
