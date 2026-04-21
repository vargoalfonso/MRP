package models

import "time"

type OutgoingRMItem struct {
	ID              int64     `json:"id"`
	TransactionID   string    `json:"transaction_id"`
	TransactionDate time.Time `json:"transaction_date"`
	Uniq            string    `json:"uniq"`
	RMName          *string   `json:"rm_name"`
	PackingListRM   *string   `json:"packing_list_rm"`
	Unit            *string   `json:"unit"`
	QuantityOut     float64   `json:"quantity_out"`
	StockBefore     float64   `json:"stock_before"`
	StockAfter      float64   `json:"stock_after"`
	Reason          string    `json:"reason"`
	Purpose         *string   `json:"purpose"`
	WorkOrderNo         *string   `json:"work_order_no"`
	DestinationLocation *string   `json:"destination_location"`
	RequestedBy     *string   `json:"requested_by"`
	Remarks         *string   `json:"remarks"`
	CreatedBy       *string   `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
}

type Pagination struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

type OutgoingRMListResponse struct {
	Items      []OutgoingRMItem `json:"items"`
	Pagination Pagination       `json:"pagination"`
}

// FormOptionItem is returned by GET /form-options to pre-fill the create modal.
type FormOptionItem struct {
	ID                int64    `json:"id"`
	UniqCode          string   `json:"uniq_code"`
	PartNumber        *string  `json:"part_number"`
	PartName          *string  `json:"part_name"`
	UOM               *string  `json:"uom"`
	StockQty          float64  `json:"stock_qty"`
	WarehouseLocation *string  `json:"warehouse_location"`
}
