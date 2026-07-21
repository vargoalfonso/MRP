package models

type PreviewDNRequest struct {
	PONumber string `json:"po_number" validate:"required"`
}
type CreateDNRequest struct {
	PONumber string `json:"po_number" validate:"required"`
	Period   string `json:"period" validate:"required"`
	Type     string `json:"type" validate:"required"`
	WONumber string `json:"wo_number"` // optional, diisi untuk DN Subcon

	Items []CreateDNItemRequest `json:"items" validate:"required,dive"`
}

type CreateDNItemRequest struct {
	ItemUniqCode string `json:"item_uniq_code" validate:"required"`
	Qty          int64  `json:"qty" validate:"required"`
	IncomingDate string `json:"incoming_date" validate:"required"`
}

type PreviewDNResponse struct {
	Period          string                  `json:"period"`
	PONumber        string                  `json:"po_number"`
	Supplier        string                  `json:"supplier"`
	TotalPO         int64                   `json:"total_po"`
	TotalIncoming   int64                   `json:"total_incoming"`
	TotalDNCreatd   int64                   `json:"total_dn_created"`
	TotalDNIncoming int64                   `json:"total_dn_incoming"`
	MaterialGrade   string                  `json:"material_grade"`
	Items           []PreviewDNItemResponse `json:"items"`
}

type PreviewDNItemResponse struct {
	ItemUniqCode  string `json:"item_uniq_code"`
	MaterialInfo  string `json:"material_info"`
	TotalQty      int64  `json:"total_qty"`
	RemainingQty  int64  `json:"remaining_qty"`
	UOM           string `json:"uom"`
	OrderQty      int64  `json:"order_qty"`
	PcsPerKanban  int64  `json:"pcs_per_kanban"`
	PackingNumber string `json:"packing_number"`
	DateIncoming  string `json:"date_incoming"`
}

type PreviewDNItem struct {
	Packing string `json:"packing" validate:"required"`
}

type PreviewDNItemRespons struct {
	DNNumber      string `json:"dn_number"`
	PackingNumber string `json:"packing_number"`
	PONumber      string `json:"po_number"`
	Supplier      string `json:"supplier"`
	ItemUniqCode  string `json:"item_uniq_code"`
	MaterialInfo  string `json:"material_info"`
	Weight        int64  `json:"weight"`
	TotalQty      int64  `json:"total_qty"`
	RemainingQty  int64  `json:"remaining_qty"`
	UOM           string `json:"uom"`
	OrderQty      int64  `json:"order_qty"`
	PcsPerKanban  int64  `json:"pcs_per_kanban"`
}

type QRPayload struct {
	Packing  string `json:"packing"`
	Quantity int    `json:"qty"`
}

type Pagination struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

type ScanDeliveryRequest struct {
	DNNumber     string  `json:"dn_number"`
	KanbanNumber string  `json:"kanban_number"`
	Qty          float64 `json:"qty"`
	ScannedBy    string  `json:"scanned_by"`
	WONumber     string  `json:"wo_number"` // optional, work order untuk DN Subcon OUT
}

// ScanDeliveryInRequest is used by action-ui "DN Subcon IN" to record goods
// returning from an external (subcon) process. Based on the work order's
// poka-yoke process flow, the scanned quantity is routed either to WIP (when a
// process still remains after subcon) or to Finished Goods (when subcon is the
// last process).
type ScanDeliveryInRequest struct {
	DNNumber     string  `json:"dn_number"`
	KanbanNumber string  `json:"kanban_number"`
	Qty          float64 `json:"qty"`
	ScannedBy    string  `json:"scanned_by"`
	WONumber     string  `json:"wo_number"`
}

// ScanDeliveryInResult reports where the returned goods were placed.
type ScanDeliveryInResult struct {
	Destination string `json:"destination"` // "WIP" | "Finished Goods"
	NextProcess string `json:"next_process"`
	ItemUniq    string `json:"item_uniq_code"`
	Qty         float64 `json:"qty"`
}

type SubmitDeliveryRequest struct {
	CustomerID int64  `json:"customer_id"`
	Cycle      string `json:"cycle"`
	Date       string `json:"date"`
	CreatedBy  string `json:"created_by"`
	Priority   string `json:"priority"`

	Items []DeliveryItemRequest `json:"items"`
}

type DeliveryItemRequest struct {
	ItemUniqCode string  `json:"item_uniq_code"`
	Qty          float64 `json:"qty"`
	UOM          string  `json:"uom"`
}
