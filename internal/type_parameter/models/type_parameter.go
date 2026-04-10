package models

import (
	"time"
)

type TypeParameter struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	TypeCode  string    `json:"type_code"`
	TypeName  string    `json:"type_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
