package models

import (
	"time"

	"gorm.io/gorm"
)

type SupplierInfo struct {
	ID           int64          `gorm:"primaryKey;autoIncrement" json:"row_id"`
	UUID         string         `gorm:"uniqueIndex;not null" json:"id"`
	Uniq         string         `gorm:"not null" json:"uniq"`
	UniqZahir    *string        `gorm:"type:text" json:"uniq_zahir,omitempty"`
	SupplierName string         `gorm:"not null" json:"supplier_name"`
	Type         string         `gorm:"size:50;not null" json:"type"`
	Status       string         `gorm:"size:20;not null;default:'active'" json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (SupplierInfo) TableName() string {
	return "supplier_info"
}

type CreateSupplierInfoRequest struct {
	Uniq         string `json:"uniq" validate:"required"`
	UniqZahir    string `json:"uniq_zahir" validate:"omitempty"`
	SupplierName string `json:"supplier_name" validate:"required"`
	Type         string `json:"type" validate:"required"`
	Status       string `json:"status" validate:"required,oneof=active inactive"`
}

type UpdateSupplierInfoRequest struct {
	UniqZahir string `json:"uniq_zahir" validate:"omitempty"`
	Status    string `json:"status" validate:"required,oneof=active inactive"`
}

type ListSupplierInfoQuery struct {
	Search string `form:"search"`
	Status string `form:"status"`
	Page   int    `form:"page"`
	Limit  int    `form:"limit"`
}
