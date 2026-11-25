package models

import "time"

type IncomingScanDNItem struct {
	ID             int64      `json:"id"`
	ItemUniqCode   string     `json:"item_uniq_code"`
	PackingNumber  *string    `json:"packing_number"`
	QtyOrdered     int        `json:"qty_ordered"`
	QtyReceived    int        `json:"qty_received"`
	QtyRemaining   int        `json:"qty_remaining"`
	WeightKg       *int       `json:"weight_kg"`
	WeightReceived *float64   `json:"weight_received"`
	QualityStatus  string     `json:"quality_status"`
	ReceivedAt     *time.Time `json:"received_at"`
	UOM            *string    `json:"uom"`
	// Additional context for warehouse operator
	PoNumber        *string `json:"po_number"`
	SupplierName    *string `json:"supplier_name"`
	RawMaterialType *string `json:"raw_material_type"` // RM | INDIRECT | SUBCON
	DnNumber        *string `json:"dn_number"`
}

type IncomingScanResponse struct {
	QCTaskID int64              `json:"qc_task_id"`
	DNItem   IncomingScanDNItem `json:"dn_item"`
}
