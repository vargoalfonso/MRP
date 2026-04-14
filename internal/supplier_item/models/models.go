package models

import (
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	SupplierItemTypeRawMaterial = "raw_material"
	SupplierItemTypeIndirect    = "indirect"
	SupplierItemTypeSubcon      = "subcon"

	SupplierItemStatusActive   = "active"
	SupplierItemStatusInactive = "inactive"
)

var AllowedSupplierItemTypes = []string{
	SupplierItemTypeRawMaterial,
	SupplierItemTypeIndirect,
	SupplierItemTypeSubcon,
}

var AllowedSupplierItemStatuses = []string{
	SupplierItemStatusActive,
	SupplierItemStatusInactive,
}

type SupplierItem struct {
	ID            int64          `gorm:"primaryKey;autoIncrement" json:"-"`
	UUID          string         `gorm:"uniqueIndex;not null" json:"id"`
	SupplierUUID  string         `gorm:"index;not null" json:"supplier_uuid"`
	SupplierName  string         `gorm:"not null" json:"supplier_name"`
	SebangoCode   string         `gorm:"size:100;not null" json:"sebango_code"`
	UniqCode      string         `gorm:"size:100;not null" json:"uniq_code"`
	Type          string         `gorm:"size:32;not null" json:"type"`
	Description   *string        `gorm:"type:text" json:"description,omitempty"`
	Quantity      int64          `gorm:"not null;default:0" json:"quantity"`
	UOM           *string        `gorm:"column:uom;size:32" json:"uom,omitempty"`
	Weight        *float64       `gorm:"column:weight;type:numeric(15,4)" json:"weight,omitempty"`
	PcsPerKanban  *int64         `gorm:"column:pcs_per_kanban" json:"pcs_per_kanban,omitempty"`
	CustomerCycle *string        `gorm:"column:customer_cycle;size:100" json:"customer_cycle,omitempty"`
	Status        string         `gorm:"size:20;not null;default:'active'" json:"status"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (SupplierItem) TableName() string {
	return "supplier_item"
}

type CreateSupplierItemRequest struct {
	SupplierUUID  string `json:"supplier_uuid" validate:"required,uuid4"`
	SebangoCode   string `json:"sebango_code" validate:"required,max=100"`
	UniqCode      string `json:"uniq_code" validate:"required,max=100"`
	Type          string `json:"type" validate:"required,oneof=raw_material indirect subcon"`
	Description   string `json:"description" validate:"omitempty,max=2000"`
	Quantity      string `json:"quantity" validate:"omitempty,max=50"`
	UOM           string `json:"uom" validate:"omitempty,max=32"`
	Weight        string `json:"weight" validate:"omitempty,max=50"`
	PcsPerKanban  string `json:"pcs_per_kanban" validate:"omitempty,max=50"`
	CustomerCycle string `json:"customer_cycle" validate:"omitempty,max=100"`
	Status        string `json:"status" validate:"required,oneof=active inactive"`
}

type UpdateSupplierItemRequest struct {
	SupplierUUID  string `json:"supplier_uuid" validate:"required,uuid4"`
	SebangoCode   string `json:"sebango_code" validate:"required,max=100"`
	UniqCode      string `json:"uniq_code" validate:"required,max=100"`
	Type          string `json:"type" validate:"required,oneof=raw_material indirect subcon"`
	Description   string `json:"description" validate:"omitempty,max=2000"`
	Quantity      string `json:"quantity" validate:"omitempty,max=50"`
	UOM           string `json:"uom" validate:"omitempty,max=32"`
	Weight        string `json:"weight" validate:"omitempty,max=50"`
	PcsPerKanban  string `json:"pcs_per_kanban" validate:"omitempty,max=50"`
	CustomerCycle string `json:"customer_cycle" validate:"omitempty,max=100"`
	Status        string `json:"status" validate:"required,oneof=active inactive"`
}

type ListSupplierItemQuery struct {
	Search       string `form:"search"`
	SupplierUUID string `form:"supplier_uuid"`
	Type         string `form:"type"`
	Status       string `form:"status"`
	Page         int    `form:"page"`
	Limit        int    `form:"limit"`
}

type SupplierItemListFilters struct {
	Search       string
	SupplierUUID *string
	Type         *string
	Status       *string
	Page         int
	Limit        int
	Offset       int
}

type SupplierItemListResult struct {
	Items      []SupplierItem `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func NewPaginationMeta(page, limit int, total int64) PaginationMeta {
	totalPages := 0
	if limit > 0 && total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	return PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}

func Trimmed(value string) string {
	return strings.TrimSpace(value)
}

func NormalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	cleaned := Trimmed(*value)
	if cleaned == "" {
		return nil
	}
	return &cleaned
}
