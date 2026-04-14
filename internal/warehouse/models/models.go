package models

import (
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	WarehouseTypeRawMaterial   = "raw_material"
	WarehouseTypeWIP           = "wip"
	WarehouseTypeFinishedGoods = "finished_goods"
	WarehouseTypeSubcon        = "subcon"
	WarehouseTypeGeneral       = "general"
)

var AllowedWarehouseTypes = []string{
	WarehouseTypeRawMaterial,
	WarehouseTypeWIP,
	WarehouseTypeFinishedGoods,
	WarehouseTypeSubcon,
	WarehouseTypeGeneral,
}

type Warehouse struct {
	ID            int64          `gorm:"primaryKey;autoIncrement" json:"-"`
	UUID          string         `gorm:"uniqueIndex;not null" json:"id"`
	WarehouseName string         `gorm:"size:255;not null" json:"warehouse_name"`
	TypeWarehouse string         `gorm:"column:type_warehouse;size:50;not null" json:"type_warehouse"`
	PlantID       string         `gorm:"column:plant_id;size:100;not null" json:"plant_id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Warehouse) TableName() string {
	return "warehouse"
}

type CreateWarehouseRequest struct {
	WarehouseName string `json:"warehouse_name" validate:"required,max=255"`
	TypeWarehouse string `json:"type_warehouse" validate:"required,oneof=raw_material wip finished_goods subcon general"`
	PlantID       string `json:"plant_id" validate:"required,max=100"`
}

type UpdateWarehouseRequest struct {
	WarehouseName string `json:"warehouse_name" validate:"required,max=255"`
	TypeWarehouse string `json:"type_warehouse" validate:"required,oneof=raw_material wip finished_goods subcon general"`
	PlantID       string `json:"plant_id" validate:"required,max=100"`
}

type ListWarehouseQuery struct {
	Search        string `form:"search"`
	TypeWarehouse string `form:"type_warehouse"`
	PlantID       string `form:"plant_id"`
	Page          int    `form:"page"`
	Limit         int    `form:"limit"`
}

type WarehouseListFilters struct {
	Search        string
	TypeWarehouse *string
	PlantID       *string
	Page          int
	Limit         int
	Offset        int
}

type WarehouseListResult struct {
	Items      []Warehouse    `json:"items"`
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
