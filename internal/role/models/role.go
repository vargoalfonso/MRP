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
	UserCount   int64             `gorm:"-" json:"user_count"`
}

type RoleUser struct {
	ID           int64      `json:"id"`
	FullName     string     `json:"full_name"`
	Email        string     `json:"email"`
	JobTitle     string     `json:"job_title"`
	Status       string     `json:"status"`
	DepartmentID *int64     `json:"department_id"`
	JoinDate     *time.Time `json:"join_date"`
}
