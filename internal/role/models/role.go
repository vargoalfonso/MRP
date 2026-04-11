package models

import (
	"time"

	"gorm.io/datatypes"
)

type Role struct {
	ID          int64             `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string            `json:"name" db:"name"`
	Description string            `json:"description" db:"description"`
	Permissions datatypes.JSONMap `gorm:"type:jsonb"`
	Status      string            `json:"status" db:"status"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
}
