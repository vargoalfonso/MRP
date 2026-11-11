package models

type CreateWIPRequest struct {
	WoID     int64           `json:"wo_id"`
	WONumber string          `json:"wo_number"` // optional (UI only)
	Items    []CreateWIPItem `json:"items"`
}

type CreateWIPItem struct {
	Uniq         string        `json:"uniq"`
	KanbanNumber string        `json:"kanban_number"`
	WipType      string        `json:"wip_type"` // default: draft
	UOM          string        `json:"uom"`
	Stock        int           `json:"stock"`
	StockKanban  int           `json:"stock_kanban"` // optional (UI only)
	ProcessFlow  []ProcessFlow `json:"process_flow"`
}

type ProcessFlow struct {
	OpSeq       int    `json:"op_seq"`
	MachineName string `json:"machine_name"`
	ProcessName string `json:"process_name"`
}

type UpdateWIPRequest struct {
	Status string `json:"status"`
}

type ScanRequest struct {
	WipItemID int64  `json:"wip_item_id"`
	Action    string `json:"action"` // scan_in | scan_out
	Qty       int    `json:"qty"`
}

type UpdateWIPItemRequest struct {
	Stock  *int   `json:"stock,omitempty"`
	Status string `json:"status,omitempty"`
}

type UpdateWIPItemScan struct {
	QtyIn        int
	QtyOut       int
	QtyRemaining int
	Status       string
}

type CreateWIPLogRequest struct {
	WipItemID int64  `json:"wip_item_id"`
	Action    string `json:"action"`
	Qty       int    `json:"qty"`
}
