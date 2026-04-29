package models

import "time"

type OverviewResponse struct {
	AsOf               time.Time       `json:"as_of"`
	WindowHours        int             `json:"window_hours"`
	Cards              OverviewCards   `json:"cards"`
	BySource           []SourceSummary `json:"by_source"`
	TopIssues          []IssueSummary  `json:"top_issues"`
	ImplementationNote string          `json:"implementation_note"`
}

type OverviewCards struct {
	TotalReports  int64   `json:"total_reports"`
	TotalDefects  float64 `json:"total_defects"`
	TotalScrap    float64 `json:"total_scrap"`
	PendingRework int64   `json:"pending_rework"`
}

type SourceSummary struct {
	DefectSource string  `json:"defect_source"`
	QtyDefect    float64 `json:"qty_defect"`
	QtyScrap     float64 `json:"qty_scrap"`
}

type IssueSummary struct {
	ReasonCode string  `json:"reason_code"`
	ReasonText string  `json:"reason_text"`
	QtyDefect  float64 `json:"qty_defect"`
}

type Pagination struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

type ProductionQCItem struct {
	QCLogID            int64   `json:"qc_log_id"`
	ReportDate         string  `json:"report_date"`
	WONumber           string  `json:"wo_number"`
	UniqCode           string  `json:"uniq_code"`
	KanbanNumber       string  `json:"kanban_number"`
	ItemsChecked       float64 `json:"items_checked"`
	IssueLabel         *string `json:"issue_label"`
	QtyDefect          float64 `json:"qty_defect"`
	QtyScrap           float64 `json:"qty_scrap"`
	QualityRatePercent float64 `json:"quality_rate_percent"`
	Status             string  `json:"status"`
}

type ProductionQCListResponse struct {
	Items      []ProductionQCItem `json:"items"`
	Pagination Pagination         `json:"pagination"`
}

type IncomingQCItem struct {
	QCLogID            int64   `json:"qc_log_id"`
	QCTaskID           int64   `json:"qc_task_id"`
	ReportDate         string  `json:"report_date"`
	DNNumber           string  `json:"dn_number"`
	KanbanPLScan       string  `json:"kanban_pl_scan"`
	PONumber           string  `json:"po_number"`
	PartnerType        string  `json:"partner_type"`
	PartnerName        string  `json:"partner_name"`
	SupplierID         *int64  `json:"supplier_id"`
	SupplierName       string  `json:"supplier_name"`
	UniqCode           string  `json:"uniq_code"`
	ItemsChecked       float64 `json:"items_checked"`
	IssueLabel         *string `json:"issue_label"`
	QtyDefect          float64 `json:"qty_defect"`
	QtyScrap           float64 `json:"qty_scrap"`
	QualityRatePercent float64 `json:"quality_rate_percent"`
	Status             string  `json:"status"`
}

type IncomingQCListResponse struct {
	Items      []IncomingQCItem `json:"items"`
	Pagination Pagination       `json:"pagination"`
}

type ProductReturnQCItem struct {
	QCLogID             int64   `json:"qc_log_id"`
	ProductReturnID     int64   `json:"product_return_id"`
	ReportDate          string  `json:"report_date"`
	ProductReturnNumber string  `json:"product_return_number"`
	DNNumber            string  `json:"dn_number"`
	PartnerType         string  `json:"partner_type"`
	PartnerName         string  `json:"partner_name"`
	ItemsChecked        float64 `json:"items_checked"`
	IssueLabel          *string `json:"issue_label"`
	QtyRework           float64 `json:"qty_rework"`
	QtyDefect           float64 `json:"qty_defect"`
	QtyScrap            float64 `json:"qty_scrap"`
	QualityRatePercent  float64 `json:"quality_rate_percent"`
	Status              string  `json:"status"`
}

type ProductReturnQCListResponse struct {
	Items      []ProductReturnQCItem `json:"items"`
	Pagination Pagination            `json:"pagination"`
}

type DefectItem struct {
	DefectID       int64   `json:"defect_id"`
	QCLogID        int64   `json:"qc_log_id"`
	ReportDate     string  `json:"report_date"`
	DefectSource   string  `json:"defect_source"`
	KanbanPL       string  `json:"kanban_pl"`
	UniqCode       string  `json:"uniq_code"`
	ProductName    string  `json:"product_name"`
	ReasonCode     string  `json:"reason_code"`
	ReasonText     string  `json:"reason_text"`
	QtyDefect      float64 `json:"qty_defect"`
	QtyScrap       float64 `json:"qty_scrap"`
	IsRepairable   bool    `json:"is_repairable"`
	WOReworkStatus string  `json:"wo_rework_status"`
	ReworkQCTaskID *int64  `json:"rework_qc_task_id"`
}

type DefectListResponse struct {
	Items              []DefectItem `json:"items"`
	Pagination         Pagination   `json:"pagination"`
	ImplementationNote string       `json:"implementation_note"`
}

type IssueListItem struct {
	ReasonCode string `json:"reason_code"`
	ReasonText string `json:"reason_text"`
	Category   string `json:"category"`
}

type CreateManualQCReportRequest struct {
	QCType            string  `json:"qc_type"`
	ReportDate        string  `json:"report_date"`
	ReferenceNumber   string  `json:"reference_number"`
	UniqCode          string  `json:"uniq_code"`
	NumberOfItemCheck float64 `json:"number_of_item_check"`
	IssueReasonCode   string  `json:"issue_reason_code"`
	IssueReasonText   string  `json:"issue_reason_text"`
	NumberOfDefect    float64 `json:"number_of_defect"`
	NumberOfScrap     float64 `json:"number_of_scrap"`
	Status            string  `json:"status"`
}

type ManualReferenceOptionItem struct {
	QCType                string  `json:"qc_type"`
	ReferenceNumber       string  `json:"reference_number"`
	SecondaryReference    string  `json:"secondary_reference"`
	UniqCode              string  `json:"uniq_code"`
	ContextID             string  `json:"context_id"`
	KanbanOrPackingNumber string  `json:"kanban_or_packing_number"`
	PartName              string  `json:"part_name"`
	UOM                   string  `json:"uom"`
	ItemQty               float64 `json:"item_qty"`
}

type ManualReferenceOptionsResponse struct {
	Items []ManualReferenceOptionItem `json:"items"`
}
