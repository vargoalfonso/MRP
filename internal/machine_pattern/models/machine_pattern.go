package models

import (
	"time"
)

type MachinePattern struct {
	ID           int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UniqCode     string     `gorm:"size:64;not null;index" json:"uniq_code"`
	MachineID    int64      `gorm:"not null;index" json:"machine_id"`
	CycleTime    *float64   `gorm:"type:decimal(10,2)" json:"cycle_time,omitempty"`
	PatternValue float64    `gorm:"type:decimal(5,2);not null;default:1.0" json:"pattern_value"`
	WorkingDays  int        `gorm:"not null;default:26" json:"working_days"`
	MovingType   string     `gorm:"size:20;not null;default:Normal" json:"moving_type"`
	MinOutput    *float64   `gorm:"type:decimal(10,2)" json:"min_output,omitempty"`
	PRLReference *float64   `gorm:"type:decimal(15,4)" json:"prl_reference,omitempty"`
	Status       string     `gorm:"size:20;not null;default:Active" json:"status"`
	CreatedBy    *string    `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt    time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt    *time.Time `gorm:"index" json:"-"`
}

func (MachinePattern) TableName() string { return "machine_patterns" }

type MachinePatternResponse struct {
	ID           int64     `json:"id"`
	UniqCode     string    `json:"uniq_code"`
	MachineID    int64     `json:"machine_id"`
	MachineName  string    `json:"machine_name,omitempty"`
	CycleTime    *float64  `json:"cycle_time,omitempty"`
	PatternValue float64   `json:"pattern_value"`
	WorkingDays  int       `json:"working_days"`
	MovingType   string    `json:"moving_type"`
	MinOutput    *float64  `json:"min_output,omitempty"`
	PRLReference *float64   `json:"prl_reference,omitempty"`
	Status       string    `json:"status"`
	CreatedBy    *string   `json:"created_by,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ListMachinePatternResponse struct {
	Items      []MachinePatternResponse `json:"items"`
	Pagination Pagination              `json:"pagination"`
}

type Pagination struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

type MachinePatternSummary struct {
	TotalPattern int     `json:"total_pattern"`
	FastMoving   int     `json:"fast_moving"`
	SlowMoving   int     `json:"slow_moving"`
	Normal       int     `json:"normal"`
	AvgPattern   float64 `json:"avg_pattern"`
}
