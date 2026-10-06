package models

type CreateSessionRequest struct {
	InventoryType     string               `json:"inventory_type" validate:"required"`
	Method            string               `json:"method" validate:"required"`
	PeriodMonth       int                  `json:"period_month" validate:"required"`
	PeriodYear        int                  `json:"period_year" validate:"required"`
	WarehouseLocation *string              `json:"warehouse_location"`
	ScheduleDate      *string              `json:"schedule_date"`
	CountedDate       *string              `json:"counted_date"`
	Remarks           *string              `json:"remarks"`
	Approver          *string              `json:"approver"`
	Items             []CreateEntryRequest `json:"items"`
}

type UpdateSessionRequest struct {
	Method            *string `json:"method"`
	PeriodMonth       *int    `json:"period_month"`
	PeriodYear        *int    `json:"period_year"`
	WarehouseLocation *string `json:"warehouse_location"`
	ScheduleDate      *string `json:"schedule_date"`
	CountedDate       *string `json:"counted_date"`
	Remarks           *string `json:"remarks"`
	Approver          *string `json:"approver"`
}

type CreateEntryRequest struct {
	// [so-packing] UniqCode jadi opsional kalau PackingNumber dikirim:
	// backend yang me-resolve uniq dari packing list / DN yang discan.
	UniqCode        string   `json:"uniq_code"`
	PackingNumber   string   `json:"packing_number"`
	CountedQty      float64  `json:"counted_qty" validate:"required"`
	WeightKg        *float64 `json:"weight_kg"`
	CyclePengiriman *string  `json:"cycle_pengiriman"`
	UserCounter     *string  `json:"user_counter"`
	Remarks         *string  `json:"remarks"`
}

type BulkCreateEntryRequest struct {
	Items []CreateEntryRequest `json:"items" validate:"required"`
}

type UpdateEntryRequest struct {
	UniqCode        *string  `json:"uniq_code"`
	CountedQty      *float64 `json:"counted_qty"`
	WeightKg        *float64 `json:"weight_kg"`
	CyclePengiriman *string  `json:"cycle_pengiriman"`
	UserCounter     *string  `json:"user_counter"`
	Remarks         *string  `json:"remarks"`
}

type ApproveRequest struct {
	Action  string  `json:"action" validate:"required"`
	Remarks *string `json:"remarks"`
}

type FormOptionsQuery struct {
	Type   string
	Method string
	Q      string
	Limit  int
}

type HistoryLogsQuery struct {
	Type     string
	UniqCode string
	From     string
	To       string
	Limit    int
	Offset   int
	Page     int
}

// CheckCountRequest asks the server whether counted quantities are below /
// above the CURRENT system stock. The system stock itself is never returned.
type CheckCountRequest struct {
	Type   string           `json:"type"`
	Method string           `json:"method"`
	Items  []CheckCountItem `json:"items"`
}

type CheckCountItem struct {
	// Key is echoed back so the client can map a result to its form row.
	Key        string  `json:"key"`
	UniqCode   string  `json:"uniq_code"`
	CountedQty float64 `json:"counted_qty"`
}
