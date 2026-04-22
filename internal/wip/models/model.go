package models

import "time"

type WIP struct {
	ID        int64     `gorm:"primaryKey"`
	WoID      int64     `gorm:"column:wo_id"`
	Status    string    `gorm:"column:status"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (WIP) TableName() string {
	return "wips"
}

type WIPItem struct {
	ID int64 `gorm:"primaryKey"`

	WipID int64 `gorm:"column:wip_id"`

	// 🔥 dari work_order_items
	Uniq string `gorm:"column:uniq"` // optional display

	PackingNumber string `gorm:"column:packing_number"`
	WipType       string `gorm:"column:wip_type"` // draft

	// 🔥 dari process_flow_json
	ProcessName string `gorm:"column:process_name"`
	MachineName string `gorm:"column:machine_name"`
	OpSeq       int    `gorm:"column:op_seq"`

	Seq int `gorm:"column:seq"`
	// 🔥 dari work_order_items
	UOM string `gorm:"column:uom"`

	// 🔥 input user
	Stock int `gorm:"column:stock"`

	// tracking
	QtyIn        int `gorm:"column:qty_in"`
	QtyOut       int `gorm:"column:qty_out"`
	QtyRemaining int `gorm:"column:qty_remaining"`

	Status string `gorm:"column:status"` // queue, process, done

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (WIPItem) TableName() string {
	return "wip_items"
}

type WIPLog struct {
	ID int64 `gorm:"primaryKey"`

	WipItemID int64  `gorm:"column:wip_item_id"`
	Action    string `gorm:"column:action"`
	Qty       int    `gorm:"column:qty"`

	CreatedAt time.Time `gorm:"column:created_at"`
}

func (WIPLog) TableName() string {
	return "wip_logs"
}

type WIPListResponse struct {
	Process               string `json:"process"`
	Uniq                  string `json:"uniq"`
	PartNumber            string `json:"part_number"`
	PartInfo              string `json:"part_info"`
	WONumber              string `json:"wo_number"`
	Stock                 int    `json:"stock"`
	KanbanNumber          string `json:"kanban_number"`
	Type                  string `json:"type"`
	StockToCompleteKanban int    `json:"stock_to_complete_kanban"`
	Kanban                int    `json:"kanban"`
}
