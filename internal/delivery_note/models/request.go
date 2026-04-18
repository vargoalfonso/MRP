package models

type CreateDNRequest struct {
	PONumber string `json:"po_number" validate:"required"`
	Period   string `json:"period" validate:"required"`
	Type     string `json:"type" validate:"required"`

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
	Packing string `json:"packing"`
}

type Pagination struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}
