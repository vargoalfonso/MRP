package models

import (
	"time"

	"gorm.io/gorm"
)

type Customer struct {
	ID           uint   `gorm:"primaryKey"`
	UUID         string `gorm:"type:varchar(36);uniqueIndex"`
	CustomerID   string `gorm:"type:varchar(50);index"` // customer_code
	CustomerName string `gorm:"type:varchar(255);index"`
	PhoneNumber  string `gorm:"type:varchar(20)"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"`
}

func (Customer) TableName() string {
	return "customers"
}

type PRL struct {
	ID uint `gorm:"primaryKey"`

	UUID  string `gorm:"type:varchar(36);uniqueIndex"`
	PRLID string `gorm:"type:varchar(50);uniqueIndex"`

	CustomerUUID string `gorm:"type:varchar(36);index"`
	CustomerCode string `gorm:"type:varchar(50)"`
	CustomerName string `gorm:"type:varchar(255)"`

	UniqBomUUID *string `gorm:"type:uuid"`

	UniqCode     string `gorm:"type:varchar(100);index"`
	ProductModel string `gorm:"type:varchar(100)"`
	PartName     string `gorm:"type:varchar(255)"`
	PartNumber   string `gorm:"type:varchar(100)"`

	ForecastPeriod string  `gorm:"type:varchar(20);index"`
	Quantity       float64 `gorm:"type:decimal(20,2)"`

	Status string `gorm:"type:varchar(20);default:'pending'"`

	ApprovedAt *time.Time
	RejectedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"`
}

func (PRL) TableName() string {
	return "prls"
}

type Item struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	UUID      string `gorm:"type:uuid;not null;uniqueIndex"`
	UniqCode  string `gorm:"type:varchar(100);not null;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type BulkInsertResponse struct {
	Success    int    `json:"success"`
	Failed     int    `json:"failed"`
	FailedFile string `json:"failed_file"`
}

type FailedImport struct {
	RowNumber      int    `json:"row_number"` // Penting untuk mapping balik ke Excel
	CustomerName   string `json:"customer_name"`
	UniqCode       string `json:"uniq_code"`
	ProductModel   string `json:"product_model"`
	PartName       string `json:"part_name"`
	PartNumber     string `json:"part_number"`
	ForecastPeriod string `json:"forecast_period"`
	Quantity       int    `json:"quantity"`
	ErrorMessage   string `json:"error_message"`
}
