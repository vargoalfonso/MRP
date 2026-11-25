package models

import (
	"time"
)

type GlobalParameter struct {
	ID             int64     `gorm:"primaryKey"`
	ParameterGroup string    `gorm:"column:parameter_group"`
	Period         string    `gorm:"column:period"`
	WorkingDays    int       `gorm:"column:working_days"`
	Status         string    `gorm:"column:status"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (GlobalParameter) TableName() string {
	return "global_parameters"
}
