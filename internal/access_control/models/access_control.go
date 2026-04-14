package models

import (
	"time"

	"gorm.io/gorm"
)

type AccessControlMatrix struct {
	ID           int64     `json:"id" db:"id"`
	FullName     string    `json:"full_name" db:"full_name"`
	EmployeeID   *int64    `json:"employee_id,omitempty" db:"employee_id"`
	RoleID       *int64    `json:"role_id,omitempty" db:"role_id"`
	DepartmentID *int64    `json:"department_id,omitempty" db:"department_id"`
	Status       string    `json:"status" db:"status"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type User struct {
	ID         int64   `gorm:"primaryKey;autoIncrement"                    json:"-"`
	UUID       string  `gorm:"uniqueIndex;not null;default:uuid_generate_v4()" json:"id"`
	Username   string  `gorm:"uniqueIndex;not null"                        json:"username"`
	Email      string  `gorm:"uniqueIndex;not null"                        json:"email"`
	Password   string  `gorm:"default:''"                                  json:"-"`
	Roles      string  `gorm:"default:'user'"                              json:"roles"`
	EmployeeID *string `gorm:"index;default:null"                          json:"employee_id,omitempty"`

	// OAuth / SSO support
	Provider   string  `gorm:"not null;default:'local'"    json:"provider"`
	ProviderID *string `gorm:"index;default:null"          json:"provider_id,omitempty"`
	AvatarURL  *string `gorm:"default:null"                json:"avatar_url,omitempty"`

	IsVerified bool `gorm:"default:false" json:"is_verified"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
