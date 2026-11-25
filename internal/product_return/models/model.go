package models

import (
	"time"
)

type ProductReturn struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Uniq           string    `gorm:"type:varchar(100);not null" json:"uniq"`
	DNNumber       string    `gorm:"type:varchar(100);not null" json:"dn_number"`
	QuantityScrap  int       `gorm:"default:0" json:"quantity_scrap"`
	QuantityRework int       `gorm:"default:0" json:"quantity_rework"`
	Status         string    `gorm:"type:varchar(50);default:'PENDING'" json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// BRD fields
	Weight        float64    `json:"weight"`
	UOM           string     `gorm:"column:uom" json:"uom"`
	DateReceived  *time.Time `json:"date_received"`
	ScrapType     string     `json:"scrap_type"`
	PackingNumber string     `gorm:"->" json:"packing_number"`
	PartName      string     `gorm:"->" json:"part_name"`
	PartNumber    string     `gorm:"->" json:"part_number"`
	Model         string     `gorm:"->" json:"model"`
}
