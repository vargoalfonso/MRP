package models

import "github.com/ganasa18/go-template/pkg/pagination"

// CreateWorkOrderResponse is returned by create endpoint.
type CreateWorkOrderResponse struct {
	ID             string                     `json:"id"`    // WO UUID
	WoID           string                     `json:"wo_id"` // alias of id, for FE clarity
	WoNumber       string                     `json:"wo_number"`
	ApprovalStatus string                     `json:"approval_status"`
	QRDataURL      *string                    `json:"qr_data_url"`
	Items          []CreateWorkOrderItemBrief `json:"items"`
}

type CreateWorkOrderItemBrief struct {
	ID           string  `json:"id"`         // WO item UUID
	WoItemID     string  `json:"wo_item_id"` // alias of id
	KanbanNumber string  `json:"kanban_number"`
	ItemUniqCode string  `json:"item_uniq_code"`
	Quantity     float64 `json:"quantity"`
	ProcessName  *string `json:"process_name"`
	QRDataURL    *string `json:"qr_data_url"`
}

type ProcessOptionItem struct {
	ProcessCode string `json:"process_code"`
	ProcessName string `json:"process_name"`
}

type ProcessOptionsResponse struct {
	Items []ProcessOptionItem `json:"items"`
}

// WorkOrderListItemDetail is a slim item row embedded in the list response.
type WorkOrderListItemDetail struct {
	ID           string  `json:"id"` // WO item UUID
	ItemUniqCode string  `json:"item_uniq_code"`
	PartName     *string `json:"part_name"`
	PartNumber   *string `json:"part_number"`
	Model        *string `json:"model"`
	Quantity     float64 `json:"quantity"`
	UOM          *string `json:"uom"`
	Status       string  `json:"status"`
}

// WorkOrderListItem is a board row.
type WorkOrderListItem struct {
	ID             string  `json:"id"` // WO UUID
	WoNumber       string  `json:"wo_number"`
	WoType         string  `json:"wo_type"`
	ReferenceWO    *string `json:"reference_wo"`
	Status         string  `json:"status"`
	ApprovalStatus string  `json:"approval_status"`
	CreatedDate    string  `json:"created_date"` // YYYY-MM-DD
	TargetDate     *string `json:"target_date"`  // YYYY-MM-DD
	CreatedByName  *string `json:"created_by_name"`
	// Board summary fields
	UniqCount   int                       `json:"uniq_count"`   // distinct uniq_codes in items
	ItemCount   int                       `json:"item_count"`   // total kanban lines
	ClosedCount int                       `json:"closed_count"` // kanban lines with status=Closed
	ProgressPct float64                   `json:"progress_pct"` // closed/total * 100
	AgingDays   int                       `json:"aging_days"`   // days since created_date
	Items       []WorkOrderListItemDetail `json:"items"`
}

type WorkOrderListResponse struct {
	Items      []WorkOrderListItem `json:"items"`
	Pagination pagination.Meta     `json:"pagination"`
}

// WorkOrderSummaryResponse is returned by GET /work-orders/summary.
type WorkOrderSummaryResponse struct {
	ActiveWOs  int `json:"active_wos"`  // status In Progress
	Completed  int `json:"completed"`   // status Completed
	PendingWOs int `json:"pending_wos"` // status Pending (approved, not yet started)
	TotalUniqs int `json:"total_uniqs"` // distinct uniq_codes across all active items
}

// WorkOrderDetailItem is a single kanban row inside WO detail.
type WorkOrderDetailItem struct {
	ID           string  `json:"id"` // WO item UUID
	WoItemID     string  `json:"wo_item_id"`
	KanbanNumber string  `json:"kanban_number"`
	ItemUniqCode string  `json:"item_uniq_code"`
	Quantity     float64 `json:"quantity"`
	UOM          *string `json:"uom"`
	ProcessName  *string `json:"process_name"`
	Status       string  `json:"status"`
	QRDataURL    *string `json:"qr_data_url"`
}

// WorkOrderDetailResponse is returned by GET /work-orders/:id.
type WorkOrderDetailResponse struct {
	ID             string                `json:"id"` // WO UUID
	WoNumber       string                `json:"wo_number"`
	WoType         string                `json:"wo_type"`
	ReferenceWO    *string               `json:"reference_wo"`
	Status         string                `json:"status"`
	ApprovalStatus string                `json:"approval_status"`
	CreatedDate    string                `json:"created_date"` // YYYY-MM-DD
	TargetDate     *string               `json:"target_date"`  // YYYY-MM-DD
	CreatedByName  *string               `json:"created_by_name"`
	Notes          *string               `json:"notes"`
	QRDataURL      *string               `json:"qr_data_url"`
	Items          []WorkOrderDetailItem `json:"items"`
}

type WorkOrderApprovalResponse struct {
	ID             string `json:"id"` // WO UUID
	WoNumber       string `json:"wo_number"`
	ApprovalStatus string `json:"approval_status"`
}

type BulkApprovalResultItem struct {
	WoNumber       string `json:"wo_number"`
	ApprovalStatus string `json:"approval_status"`
}

type BulkApprovalFailedItem struct {
	WoNumber string `json:"wo_number"`
	Reason   string `json:"reason"`
}

type BulkWorkOrderApprovalResponse struct {
	Decision       string                   `json:"decision"`
	TotalRequested int                      `json:"total_requested"`
	TotalUpdated   int                      `json:"total_updated"`
	Updated        []BulkApprovalResultItem `json:"updated"`
	Failed         []BulkApprovalFailedItem `json:"failed"`
}

type WorkOrderQRResponse struct {
	WoNumber  string `json:"wo_number"`
	QRPayload string `json:"qr_payload"`
	DataURL   string `json:"data_url"`
}

type WorkOrderItemQRResponse struct {
	KanbanNumber string `json:"kanban_number"`
	QRPayload    string `json:"qr_payload"`
	DataURL      string `json:"data_url"`
}

type UniqOptionItem struct {
	UniqCode     string   `json:"uniq_code"`
	PartName     *string  `json:"part_name"`
	PartNumber   *string  `json:"part_number"`
	UOM          *string  `json:"uom"`
	AvailableQty *float64 `json:"available_qty"`
	KanbanQty    *int     `json:"kanban_qty"`
	KanbanNumber *string  `json:"kanban_number"`
	Sources      []string `json:"sources"`
}

type UniqOptionsResponse struct {
	Items []UniqOptionItem `json:"items"`
}
