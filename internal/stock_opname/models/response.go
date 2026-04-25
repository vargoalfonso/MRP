package models

import (
	"time"

	awmodels "github.com/ganasa18/go-template/internal/approval_workflow/models"
)

type Pagination struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

type StockOpnameSessionItem struct {
	ID                int64      `json:"id"`
	UUID              string     `json:"uuid"`
	SessionNumber     string     `json:"session_number"`
	InventoryType     string     `json:"inventory_type"`
	Method            string     `json:"method"`
	PeriodMonth       int        `json:"period_month"`
	PeriodYear        int        `json:"period_year"`
	PeriodLabel       string     `json:"period_label"`
	WarehouseLocation *string    `json:"warehouse_location"`
	ScheduleDate      *time.Time `json:"schedule_date"`
	CountedDate       *time.Time `json:"counted_date"`
	Remarks           *string    `json:"remarks"`
	TotalEntries      int        `json:"total_entries"`
	TotalVarianceQty  float64    `json:"total_variance_qty"`
	SystemQtyTotal    float64    `json:"system_qty_total"`
	PhysicalQtyTotal  float64    `json:"physical_qty_total"`
	VarianceQtyTotal  float64    `json:"variance_qty_total"`
	VariancePctTotal  *float64   `json:"variance_pct_total"`
	CostImpact        float64    `json:"cost_impact"`
	Status            string     `json:"status"`
	StatusLabel       string     `json:"status_label"`
	ImpactLabel       string     `json:"impact_label"`
	SubmittedBy       *string    `json:"submitted_by"`
	SubmittedAt       *time.Time `json:"submitted_at"`
	Approver          *string    `json:"approver"`
	ApprovedBy        *string    `json:"approved_by"`
	ApprovedAt        *time.Time `json:"approved_at"`
	ApprovalRemarks   *string    `json:"approval_remarks"`
	CreatedBy         *string    `json:"created_by"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type StockOpnameEntryItem struct {
	ID                int64      `json:"id"`
	UUID              string     `json:"uuid"`
	SessionID         int64      `json:"session_id"`
	UniqCode          string     `json:"uniq_code"`
	EntityID          *int64     `json:"entity_id"`
	PartNumber        *string    `json:"part_number"`
	PartName          *string    `json:"part_name"`
	UOM               *string    `json:"uom"`
	SystemQtySnapshot float64    `json:"system_qty_snapshot"`
	CountedQty        float64    `json:"counted_qty"`
	VarianceQty       float64    `json:"variance_qty"`
	VariancePct       *float64   `json:"variance_pct"`
	WeightKg          *float64   `json:"weight_kg"`
	CyclePengiriman   *string    `json:"cycle_pengiriman"`
	UserCounter       *string    `json:"user_counter"`
	Remarks           *string    `json:"remarks"`
	Status            string     `json:"status"`
	ApprovedBy        *string    `json:"approved_by"`
	ApprovedAt        *time.Time `json:"approved_at"`
	RejectReason      *string    `json:"reject_reason"`
	CreatedBy         *string    `json:"created_by"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type StockOpnameSessionDetail struct {
	Session  StockOpnameSessionItem  `json:"session"`
	Entries  []StockOpnameEntryItem  `json:"entries"`
	Approval *StockOpnameApprovalDTO `json:"approval,omitempty"`
}

type StockOpnameApprovalDTO struct {
	InstanceID       int64                     `json:"instance_id"`
	WorkflowID       int64                     `json:"workflow_id"`
	WorkflowAction   string                    `json:"workflow_action"`
	CurrentLevel     int                       `json:"current_level"`
	MaxLevel         int                       `json:"max_level"`
	Status           string                    `json:"status"`
	SubmittedBy      string                    `json:"submitted_by"`
	ApprovalProgress awmodels.ApprovalProgress `json:"approval_progress"`
	Level1Role       string                    `json:"level_1_role,omitempty"`
	Level2Role       string                    `json:"level_2_role,omitempty"`
	Level3Role       string                    `json:"level_3_role,omitempty"`
	Level4Role       string                    `json:"level_4_role,omitempty"`
}

type StockOpnameSessionListResponse struct {
	Items      []StockOpnameSessionItem `json:"items"`
	Pagination Pagination               `json:"pagination"`
}

type StockOpnameStats struct {
	TotalRecords int64   `json:"total_records"`
	Completed    int64   `json:"completed"`
	WithVariance int64   `json:"with_variance"`
	CostImpact   float64 `json:"cost_impact"`
}

type UniqOption struct {
	UniqCode   string   `json:"uniq_code"`
	PartNumber *string  `json:"part_number"`
	PartName   *string  `json:"part_name"`
	UOM        *string  `json:"uom"`
	SystemQty  float64  `json:"system_qty"`
	WeightKg   *float64 `json:"weight_kg"`
}

type BulkCreateEntryError struct {
	Row      int    `json:"row"`
	UniqCode string `json:"uniq_code"`
	Message  string `json:"message"`
}

type BulkCreateEntryResponse struct {
	Created int                    `json:"created"`
	Errors  []BulkCreateEntryError `json:"errors"`
}

type HistoryLogItem struct {
	UniqCode   string    `json:"uniq_code"`
	Packing    *string   `json:"packing"`
	QtyChange  float64   `json:"qty_change"`
	Reason     *string   `json:"reason"`
	Qty        *float64  `json:"qty"`
	LastUpdate time.Time `json:"last_update"`
}

type HistoryLogListResponse struct {
	Items      []HistoryLogItem `json:"items"`
	Pagination Pagination       `json:"pagination"`
}

type AuditLogItem struct {
	ID            int64     `json:"id"`
	UUID          string    `json:"uuid"`
	SessionID     int64     `json:"session_id"`
	EntryID       *int64    `json:"entry_id"`
	InventoryType string    `json:"inventory_type"`
	Action        string    `json:"action"`
	EntityType    string    `json:"entity_type"`
	Actor         string    `json:"actor"`
	Remarks       *string   `json:"remarks"`
	Metadata      any       `json:"metadata"`
	CreatedAt     time.Time `json:"created_at"`
}

type AuditLogListResponse struct {
	Items      []AuditLogItem `json:"items"`
	Pagination Pagination     `json:"pagination"`
}
