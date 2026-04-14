package models

import (
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Customer struct {
	ID                    int64          `gorm:"primaryKey;autoIncrement" json:"-"`
	UUID                  string         `gorm:"uniqueIndex;not null" json:"id"`
	CustomerID            string         `gorm:"uniqueIndex;not null" json:"customer_id"`
	CustomerName          string         `gorm:"not null" json:"customer_name"`
	PhoneNumber           string         `gorm:"not null" json:"phone_number"`
	ShippingAddress       string         `gorm:"type:text;not null" json:"shipping_address"`
	BillingAddress        string         `gorm:"type:text;not null" json:"billing_address"`
	BillingSameAsShipping bool           `gorm:"not null;default:false" json:"billing_same_as_shipping"`
	BankAccount           *string        `gorm:"default:null" json:"bank_account,omitempty"`
	BankAccountNumber     *string        `gorm:"default:null" json:"bank_account_number,omitempty"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"-"`
}

type CreateCustomerRequest struct {
	CustomerName          string `json:"customer_name" validate:"required,max=255"`
	PhoneNumber           string `json:"phone_number" validate:"required,max=50"`
	ShippingAddress       string `json:"shipping_address" validate:"required,max=1000"`
	BillingAddress        string `json:"billing_address" validate:"max=1000"`
	BillingSameAsShipping bool   `json:"billing_same_as_shipping"`
	BankAccount           string `json:"bank_account" validate:"omitempty,max=150"`
	BankAccountNumber     string `json:"bank_account_number" validate:"omitempty,max=100"`
}

type UpdateCustomerRequest struct {
	CustomerName          string `json:"customer_name" validate:"required,max=255"`
	PhoneNumber           string `json:"phone_number" validate:"required,max=50"`
	ShippingAddress       string `json:"shipping_address" validate:"required,max=1000"`
	BillingAddress        string `json:"billing_address" validate:"max=1000"`
	BillingSameAsShipping bool   `json:"billing_same_as_shipping"`
	BankAccount           string `json:"bank_account" validate:"omitempty,max=150"`
	BankAccountNumber     string `json:"bank_account_number" validate:"omitempty,max=100"`
}

type ListCustomerQuery struct {
	Search string `form:"search"`
	Page   int    `form:"page"`
	Limit  int    `form:"limit"`
}

type CustomerListFilters struct {
	Search string
	Page   int
	Limit  int
	Offset int
}

type CustomerListResult struct {
	Items      []Customer     `json:"items"`
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
