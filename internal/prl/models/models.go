package models

import (
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	PRLStatusPending  = "pending"
	PRLStatusApproved = "approved"
	PRLStatusRejected = "rejected"
)

var AllowedPRLStatuses = []string{
	PRLStatusPending,
	PRLStatusApproved,
	PRLStatusRejected,
}

type UniqBillOfMaterial struct {
	ID           int64          `gorm:"primaryKey;autoIncrement" json:"-"`
	UUID         string         `gorm:"uniqueIndex;not null" json:"id"`
	UniqCode     string         `gorm:"uniqueIndex;not null" json:"uniq_code"`
	ProductModel string         `gorm:"not null" json:"product_model"`
	PartName     string         `gorm:"not null" json:"part_name"`
	PartNumber   string         `gorm:"not null" json:"part_number"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type PRL struct {
	ID             int64          `gorm:"primaryKey;autoIncrement" json:"-"`
	UUID           string         `gorm:"uniqueIndex;not null" json:"id"`
	PRLID          string         `gorm:"uniqueIndex;not null" json:"prl_id"`
	CustomerUUID   string         `gorm:"index;not null" json:"customer_uuid"`
	CustomerCode   string         `gorm:"not null" json:"customer_code"`
	CustomerName   string         `gorm:"not null" json:"customer_name"`
	UniqBOMUUID    string         `gorm:"index;not null" json:"uniq_bom_uuid"`
	UniqCode       string         `gorm:"not null" json:"uniq_code"`
	ProductModel   string         `gorm:"not null" json:"product_model"`
	PartName       string         `gorm:"not null" json:"part_name"`
	PartNumber     string         `gorm:"not null" json:"part_number"`
	ForecastPeriod string         `gorm:"not null" json:"forecast_period"`
	Quantity       int64          `gorm:"not null" json:"quantity"`
	Status         string         `gorm:"not null;default:'pending'" json:"status"`
	ApprovedAt     *time.Time     `gorm:"default:null" json:"approved_at,omitempty"`
	RejectedAt     *time.Time     `gorm:"default:null" json:"rejected_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

type CreateUniqBOMRequest struct {
	UniqCode     string `json:"uniq_code" validate:"required,max=100"`
	ProductModel string `json:"product_model" validate:"required,max=255"`
	PartName     string `json:"part_name" validate:"required,max=255"`
	PartNumber   string `json:"part_number" validate:"required,max=150"`
}

type UpdateUniqBOMRequest struct {
	UniqCode     string `json:"uniq_code" validate:"required,max=100"`
	ProductModel string `json:"product_model" validate:"required,max=255"`
	PartName     string `json:"part_name" validate:"required,max=255"`
	PartNumber   string `json:"part_number" validate:"required,max=150"`
}

type ListUniqBOMQuery struct {
	Search string `form:"search"`
	Page   int    `form:"page"`
	Limit  int    `form:"limit"`
}

type CreatePRLEntryRequest struct {
	CustomerUUID   string `json:"customer_uuid" validate:"omitempty,uuid4"`
	CustomerCode   string `json:"customer_code" validate:"omitempty,max=32"`
	UniqCode       string `json:"uniq_code" validate:"required,max=100"`
	ForecastPeriod string `json:"forecast_period" validate:"required,max=7"`
	Quantity       int64  `json:"quantity" validate:"required,gte=1"`
}

type BulkCreatePRLRequest struct {
	Entries []CreatePRLEntryRequest `json:"entries"`
}

type UpdatePRLRequest struct {
	ForecastPeriod string `json:"forecast_period" validate:"required,max=7"`
	Quantity       int64  `json:"quantity" validate:"required,gte=1"`
}

type BulkStatusActionRequest struct {
	IDs []string `json:"ids"`
}

type ListPRLQuery struct {
	Search         string `form:"search"`
	Status         string `form:"status"`
	ForecastPeriod string `form:"forecast_period"`
	CustomerUUID   string `form:"customer_uuid"`
	UniqCode       string `form:"uniq_code"`
	Page           int    `form:"page"`
	Limit          int    `form:"limit"`
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type UniqBOMListResult struct {
	Items      []UniqBillOfMaterial `json:"items"`
	Pagination PaginationMeta       `json:"pagination"`
}

type PRLListResult struct {
	Items      []PRL          `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
}

type UniqBOMListFilters struct {
	Search string
	Page   int
	Limit  int
	Offset int
}

type PRLListFilters struct {
	Search         string
	Status         *string
	ForecastPeriod *string
	CustomerUUID   *string
	UniqCode       *string
	Page           int
	Limit          int
	Offset         int
}

type CustomerLookup struct {
	ID           string `json:"id"`
	CustomerID   string `json:"customer_id"`
	CustomerName string `json:"customer_name"`
}

type ForecastPeriodOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type BulkCreatePRLResponse struct {
	CreatedCount int   `json:"created_count"`
	Items        []PRL `json:"items"`
}

type BulkStatusActionResponse struct {
	UpdatedCount int64  `json:"updated_count"`
	Status       string `json:"status"`
}

type ImportPRLResponse struct {
	ImportedCount int   `json:"imported_count"`
	Items         []PRL `json:"items"`
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
