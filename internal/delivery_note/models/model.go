package models

import (
	"time"
)

type DeliveryNote struct {
	ID            int64     `json:"id" gorm:"primaryKey"`
	DNNumber      string    `json:"dn_number" gorm:"type:varchar(50)"`
	CustomerID    int64     `json:"customer_id"`
	ContactPerson string    `json:"contact_person"`
	Period        string    `json:"period"`
	PONumber      string    `json:"po_number"`
	Type          string    `json:"type"`
	Status        string    `json:"status"`
	IncomingDate  time.Time `json:"incoming_date"`
	CreatedBy     string    `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	Items []DeliveryNoteItem `json:"items" gorm:"foreignKey:DNID;references:ID"`
}

type DeliveryNoteItem struct {
	ID           int64           `json:"id" gorm:"primaryKey"`
	DNID         int64           `json:"dn_id" gorm:"index"`
	ItemUniqCode string          `json:"item_uniq_code"`
	Quantity     int             `json:"quantity"`
	UOM          string          `json:"uom"`
	Weight       int             `json:"weight"`
	KanbanID     int64           `json:"kanban_id"`
	Kanban       KanbanParameter `json:"kanban" gorm:"foreignKey:KanbanID"`
	QR           string          `json:"qr"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type KanbanParameter struct {
	ID           int64     `json:"id" gorm:"primaryKey"`
	KanbanNumber string    `json:"kanban_number"`
	ItemUniqCode string    `json:"item_uniq_code"`
	KanbanQty    int       `json:"kanban_qty"`
	MinStock     int       `json:"min_stock"`
	MaxStock     int       `json:"max_stock"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
