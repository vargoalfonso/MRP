package models

import (
	"time"
)

type ScrapType struct {
	ID          int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Code        string     `gorm:"size:32;uniqueIndex" json:"code"`
	Name        string     `gorm:"size:128;not null" json:"name"`
	Description *string    `gorm:"type:text" json:"description,omitempty"`
	Status      string     `gorm:"size:20;not null;default:Active" json:"status"`
	IsSystem    bool       `gorm:"not null;default:false" json:"is_system"`
	CreatedBy   *string    `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt   time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt   *time.Time `gorm:"index" json:"-"`
}

func (ScrapType) TableName() string { return "scrap_types" }

type ScrapTypeResponse struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Status      string    `json:"status"`
	IsSystem    bool      `json:"is_system"`
	CreatedBy   *string   `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ListScrapTypeResponse struct {
	Items      []ScrapTypeResponse `json:"items"`
	Pagination Pagination         `json:"pagination"`
}

type Pagination struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}
