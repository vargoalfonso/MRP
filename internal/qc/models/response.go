package models

import "time"

type TaskListItem struct {
	ID        int64     `json:"id"`
	TaskType  string    `json:"task_type"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`

	// Incoming context
	PackingNumber *string  `json:"packing_number"`
	DnNumber      *string  `json:"dn_number"`
	PoNumber      *string  `json:"po_number"`
	SupplierName  *string  `json:"supplier_name"`
	ItemUniqCode  *string  `json:"item_uniq_code"`
	QtyReceived   *float64 `json:"qty_received"`
	UOM           *string  `json:"uom"`
}

type TaskListResponse struct {
	Items      []TaskListItem `json:"items"`
	Pagination struct {
		Total      int64 `json:"total"`
		Page       int   `json:"page"`
		Limit      int   `json:"limit"`
		TotalPages int64 `json:"total_pages"`
	} `json:"pagination"`
}

type ApproveIncomingTaskResponse struct {
	TaskID int64  `json:"qc_task_id"`
	Status string `json:"status"`

	DNItemID      string `json:"dn_item_id"`
	QualityStatus string `json:"quality_status"`
	ApprovedQty   int    `json:"approved_qty"`
	NgQty         int    `json:"ng_qty"`
	ScrapQty      int    `json:"scrap_qty"`
	InventoryType string `json:"inventory_type"`
	ItemUniqCode  string `json:"item_uniq_code"`
	DeltaQty      int    `json:"delta_qty"`
}

type RejectIncomingTaskResponse struct {
	TaskID int64  `json:"qc_task_id"`
	Status string `json:"status"`

	DNItemID      string `json:"dn_item_id"`
	QualityStatus string `json:"quality_status"`
	RejectedQty   int    `json:"rejected_qty"`
	Disposition   string `json:"disposition"`
}
