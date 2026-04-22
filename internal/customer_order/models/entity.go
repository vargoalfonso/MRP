package models

import "time"

type CustomerOrderDocument struct {
	ID                   int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID                 string     `gorm:"uniqueIndex;not null" json:"uuid"`
	DocumentType         string     `gorm:"size:16;not null" json:"document_type"`
	DocumentNumber       string     `gorm:"size:128;not null" json:"document_number"`
	DocumentDate         time.Time  `gorm:"not null" json:"document_date"`
	PeriodSchedule       string     `gorm:"size:64" json:"period_schedule"`
	CustomerID           int64      `gorm:"not null" json:"customer_id"`
	CustomerNameSnapshot string     `gorm:"size:255" json:"customer_name"`
	ContactPerson        *string    `gorm:"size:255" json:"contact_person"`
	DeliveryAddress      *string    `gorm:"type:text" json:"delivery_address"`
	Status               string     `gorm:"size:32;not null;default:draft" json:"status"`
	Notes                *string    `gorm:"type:text" json:"notes"`
	CreatedBy            string     `gorm:"size:255" json:"created_by"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	DeletedAt            *time.Time `gorm:"index" json:"-"`

	Items []CustomerOrderDocumentItem `gorm:"foreignKey:DocumentID" json:"items,omitempty"`
}

func (CustomerOrderDocument) TableName() string { return "customer_order_documents" }

type CustomerOrderDocumentItem struct {
	ID           int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID         string     `gorm:"uniqueIndex;not null" json:"uuid"`
	DocumentID   int64      `gorm:"not null;index" json:"-"`
	LineNo       int        `gorm:"not null" json:"line_no"`
	ItemUniqCode string     `gorm:"size:100;not null" json:"item_uniq_code"`
	PartName     string     `gorm:"size:255;not null" json:"part_name"`
	PartNumber   string     `gorm:"size:128;not null" json:"part_number"`
	Model        *string    `gorm:"size:255" json:"model"`
	UOM          string     `gorm:"size:64" json:"uom"`
	Quantity     float64    `gorm:"type:numeric(15,4);not null" json:"quantity"`
	DeliveryDate *time.Time `json:"delivery_date"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (CustomerOrderDocumentItem) TableName() string { return "customer_order_document_items" }

type ItemSnapshot struct {
	PartName   string  `gorm:"column:part_name"`
	PartNumber *string `gorm:"column:part_number"`
	Model      *string `gorm:"column:model"`
	UOM        *string `gorm:"column:uom"`
}
