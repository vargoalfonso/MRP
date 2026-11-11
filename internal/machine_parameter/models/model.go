package models

import "time"

// MechinParameter stores machine-level configuration values for system setting.
type MechinParameter struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	MachineName    string    `gorm:"column:machine_name" json:"machine_name"`
	MachineCount   int       `gorm:"column:machine_count" json:"machine_count"`
	OperatingHours int       `gorm:"column:operating_hours" json:"operating_hours"`
	Status         string    `gorm:"column:status" json:"status"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (MechinParameter) TableName() string { return "mechin_parameters" }

type ListMechinParameterResponse struct {
	Items      []MechinParameter `json:"items"`
	Pagination Pagination        `json:"pagination"`
}

type Pagination struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}
