package models

import "time"

type LiveProductionSummary struct {
	AsOf               *time.Time       `json:"as_of"`
	StaleWindowMinutes int              `json:"stale_window_minutes"`
	ThroughputToday    float64          `json:"throughput_today"`
	ActiveMachines     int64            `json:"active_machines"`
	RunningMachines    int64            `json:"running_machines"`
	IdleMachines       int64            `json:"idle_machines"`
	Items              []LiveProduction `json:"items"`
}

type LiveProduction struct {
	Machine    LiveProductionMachine  `json:"machine"`
	Production LiveProductionCurrent  `json:"production"`
	Progress   LiveProductionProgress `json:"progress"`
	Quality    LiveProductionQuality  `json:"quality"`
}

type LiveProductionMachine struct {
	Number         string `json:"number"`
	ProductionLine string `json:"production_line"`
	RuntimeStatus  string `json:"runtime_status"`
	ScannedBy      string `json:"scanned_by"`
}

type LiveProductionCurrent struct {
	WONumber     string     `json:"wo_number"`
	CurrentUniq  string     `json:"current_uniq"`
	WOStatus     string     `json:"wo_status"`
	ProcessName  string     `json:"process_name"`
	LastScanType string     `json:"last_scan_type"`
	LastScanAt   *time.Time `json:"last_scan_at"`
}

type LiveProductionProgress struct {
	TargetQty       float64 `json:"target_qty"`
	ThroughputToday float64 `json:"throughput_today"`
	OutputQty       float64 `json:"output_qty"`
	ProgressPercent float64 `json:"progress_percent"`
}

type LiveProductionQuality struct {
	CheckedQty  float64 `json:"checked_qty"`
	PassQty     float64 `json:"pass_qty"`
	DefectQty   float64 `json:"defect_qty"`
	ScrapQty    float64 `json:"scrap_qty"`
	RatePercent float64 `json:"rate_percent"`
}

type DeliveryReadinessSummary struct {
	AsOf              time.Time               `json:"as_of"`
	TotalScheduled    int64                   `json:"total_scheduled"`
	ReadyItems        int64                   `json:"ready_items"`
	AtRiskItems       int64                   `json:"at_risk_items"`
	CriticalItems     int64                   `json:"critical_items"`
	TotalRequiredQty  float64                 `json:"total_required_qty"`
	TotalAvailableQty float64                 `json:"total_available_qty"`
	TotalShortfallQty float64                 `json:"total_shortfall_qty"`
	Items             []DeliveryReadinessItem `json:"items"`
}

type DeliveryReadinessItem struct {
	Identity  DeliveryReadinessIdentity  `json:"identity"`
	Delivery  DeliveryReadinessDelivery  `json:"delivery"`
	Inventory DeliveryReadinessInventory `json:"inventory"`
	Readiness DeliveryReadinessAnalysis  `json:"readiness"`
}

type DeliveryReadinessIdentity struct {
	ScheduleNumber string `json:"schedule_number"`
	CustomerName   string `json:"customer_name"`
	ItemUniqCode   string `json:"item_uniq_code"`
	PartNumber     string `json:"part_number"`
	PartName       string `json:"part_name"`
}

type DeliveryReadinessDelivery struct {
	ScheduleDate  time.Time `json:"schedule_date"`
	DueDate       string    `json:"due_date"`
	DueTime       *string   `json:"due_time"`
	HoursUntilDue *int64    `json:"hours_until_due"`
	RequiredQty   float64   `json:"required_qty"`
}

type DeliveryReadinessInventory struct {
	FGQty            float64 `json:"fg_qty"`
	WIPQty           float64 `json:"wip_qty"`
	AvailableQty     float64 `json:"available_qty"`
	FGReadinessState string  `json:"fg_readiness_state"`
}

type DeliveryReadinessAnalysis struct {
	ShortfallQty    float64 `json:"shortfall_qty"`
	CoveragePercent float64 `json:"coverage_percent"`
	ReadinessStatus string  `json:"readiness_status"`
}

type ProductionIssuesSummary struct {
	AsOf            time.Time         `json:"as_of"`
	SourceAvailable bool              `json:"source_available"`
	WindowHours     int               `json:"window_hours"`
	TotalIssues     int64             `json:"total_issues"`
	OpenIssues      int64             `json:"open_issues"`
	CriticalIssues  int64             `json:"critical_issues"`
	HighPriority    int64             `json:"high_priority"`
	Items           []ProductionIssue `json:"items"`
}

type ProductionIssue struct {
	ID         int64      `json:"id"`
	Title      string     `json:"title"`
	IssueType  string     `json:"issue_type"`
	Machine    string     `json:"machine"`
	Status     string     `json:"status"`
	Severity   string     `json:"severity"`
	Priority   string     `json:"priority"`
	ReportedBy string     `json:"reported_by"`
	OccurredAt *time.Time `json:"occurred_at"`
	UpdatedAt  *time.Time `json:"updated_at"`
}

type PaginationMeta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

type ScanEventsSummary struct {
	AsOf         *time.Time     `json:"as_of"`
	WindowHours  int            `json:"window_hours"`
	TotalEvents  int64          `json:"total_events"`
	ScanInCount  int64          `json:"scan_in_count"`
	ScanOutCount int64          `json:"scan_out_count"`
	QCCount      int64          `json:"qc_count"`
	Items        []ScanEvent    `json:"items"`
	Pagination   PaginationMeta `json:"pagination"`
}

type ScanEvent struct {
	ScannedAt      time.Time `json:"scanned_at"`
	ScanType       string    `json:"scan_type"`
	MachineNumber  string    `json:"machine_number"`
	ProductionLine string    `json:"production_line"`
	WONumber       string    `json:"wo_number"`
	CurrentUniq    string    `json:"current_uniq"`
	ProcessName    string    `json:"process_name"`
	Qty            float64   `json:"qty"`
	ScannedBy      string    `json:"scanned_by"`
}
