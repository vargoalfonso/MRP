package models

import (
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	MaterialCategoryRawMaterial         = "Raw Material"
	MaterialCategoryIndirectRawMaterial = "Indirect Raw Material"
	MaterialCategorySubcon              = "Subcon"

	SupplierStatusActive   = "Active"
	SupplierStatusInactive = "Inactive"
)

var AllowedMaterialCategories = []string{
	MaterialCategoryRawMaterial,
	MaterialCategoryIndirectRawMaterial,
	MaterialCategorySubcon,
}

var AllowedSupplierStatuses = []string{
	SupplierStatusActive,
	SupplierStatusInactive,
}

type Supplier struct {
	ID                   int64          `gorm:"primaryKey;autoIncrement" json:"row_id"`
	UUID                 string         `gorm:"uniqueIndex;not null" json:"id"`
	SupplierCode         string         `gorm:"uniqueIndex;not null" json:"supplier_code"`
	SupplierName         string         `gorm:"not null" json:"supplier_name"`
	ContactPerson        string         `gorm:"not null" json:"contact_person"`
	ContactNumber        string         `gorm:"not null" json:"contact_number"`
	EmailAddress         string         `gorm:"not null" json:"email_address"`
	MaterialCategory     *string        `gorm:"default:null" json:"material_category,omitempty"`
	FullAddress          string         `gorm:"type:text;not null" json:"full_address"`
	City                 string         `gorm:"not null" json:"city"`
	Province             string         `gorm:"not null" json:"province"`
	Country              string         `gorm:"not null" json:"country"`
	TaxIDNPWP            string         `gorm:"not null" json:"tax_id_npwp"`
	BankName             string         `gorm:"not null" json:"bank_name"`
	BankAccountNumber    string         `gorm:"not null" json:"bank_account_number"`
	BankAccountName      string         `gorm:"not null" json:"bank_account_name"`
	PaymentTerms         string         `gorm:"not null" json:"payment_terms"`
	DeliveryLeadTimeDays int            `gorm:"not null" json:"delivery_lead_time_days"`
	Status               string         `gorm:"not null;default:'Active'" json:"status"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`
}

type CreateSupplierRequest struct {
	SupplierName         string `json:"supplier_name" validate:"required,max=255"`
	ContactPerson        string `json:"contact_person" validate:"required,max=255"`
	ContactNumber        string `json:"contact_number" validate:"required,max=50"`
	EmailAddress         string `json:"email_address" validate:"required,email,max=255"`
	MaterialCategory     string `json:"material_category" validate:"omitempty,max=50"`
	FullAddress          string `json:"full_address" validate:"required,max=1000"`
	City                 string `json:"city" validate:"required,max=150"`
	Province             string `json:"province" validate:"required,max=150"`
	Country              string `json:"country" validate:"required,max=150"`
	TaxIDNPWP            string `json:"tax_id_npwp" validate:"required,max=50"`
	BankName             string `json:"bank_name" validate:"required,max=150"`
	BankAccountNumber    string `json:"bank_account_number" validate:"required,max=100"`
	BankAccountName      string `json:"bank_account_name" validate:"required,max=255"`
	PaymentTerms         string `json:"payment_terms" validate:"required,max=150"`
	DeliveryLeadTimeDays int    `json:"delivery_lead_time_days" validate:"gte=0,lte=3650"`
	Status               string `json:"status" validate:"required,max=20"`
}

type UpdateSupplierRequest struct {
	SupplierName         string `json:"supplier_name" validate:"required,max=255"`
	ContactPerson        string `json:"contact_person" validate:"required,max=255"`
	ContactNumber        string `json:"contact_number" validate:"required,max=50"`
	EmailAddress         string `json:"email_address" validate:"required,email,max=255"`
	MaterialCategory     string `json:"material_category" validate:"omitempty,max=50"`
	FullAddress          string `json:"full_address" validate:"required,max=1000"`
	City                 string `json:"city" validate:"required,max=150"`
	Province             string `json:"province" validate:"required,max=150"`
	Country              string `json:"country" validate:"required,max=150"`
	TaxIDNPWP            string `json:"tax_id_npwp" validate:"required,max=50"`
	BankName             string `json:"bank_name" validate:"required,max=150"`
	BankAccountNumber    string `json:"bank_account_number" validate:"required,max=100"`
	BankAccountName      string `json:"bank_account_name" validate:"required,max=255"`
	PaymentTerms         string `json:"payment_terms" validate:"required,max=150"`
	DeliveryLeadTimeDays int    `json:"delivery_lead_time_days" validate:"gte=0,lte=3650"`
	Status               string `json:"status" validate:"required,max=20"`
}

type ListSupplierQuery struct {
	Search           string `form:"search"`
	Status           string `form:"status"`
	MaterialCategory string `form:"material_category"`
	Page             int    `form:"page"`
	Limit            int    `form:"limit"`
}

type SupplierListFilters struct {
	Search           string
	Status           *string
	MaterialCategory *string
	Page             int
	Limit            int
	Offset           int
}

type SupplierListResult struct {
	Items      []Supplier     `json:"items"`
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
