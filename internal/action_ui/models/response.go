package models

import "time"

type IncomingScanDNItem struct {
	ID             string     `json:"id"`
	ItemUniqCode   string     `json:"item_uniq_code"`
	PackingNumber  *string    `json:"packing_number"`
	QtyReceived    int        `json:"qty_received"`
	WeightReceived *float64   `json:"weight_received"`
	QualityStatus  string     `json:"quality_status"`
	ReceivedAt     *time.Time `json:"received_at"`
	UOM            *string    `json:"uom"`
}

type IncomingScanResponse struct {
	QCTaskID int64              `json:"qc_task_id"`
	DNItem   IncomingScanDNItem `json:"dn_item"`
}
