package models

import (
	"time"

	"gorm.io/gorm"
)

type Employee struct {
	ID           int64      `json:"id" db:"id"`
	FullName     string     `json:"full_name" db:"full_name"`
	Email        string     `json:"email" db:"email"`
	PhoneNumber  string     `json:"phone_number" db:"phone_number"`
	JobTitle     string     `json:"job_title" db:"job_title"`
	UnitCost     float64    `json:"unit_cost" db:"unit_cost"`
	JoinDate     *time.Time `json:"join_date" db:"join_date"`
	RoleID       int64      `json:"role_id" db:"role_id"`
	DepartmentID int64      `json:"department_id" db:"department_id"`
	ReportsToID  *int64     `json:"reports_to_id" db:"reports_to_id"`
	Status       string     `json:"status" db:"status"`
	Notes        string     `json:"notes" db:"notes"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

type UserActivation struct {
	ID        int64
	UserID    int64
	Token     string
	ExpiredAt time.Time
	Used      bool
	CreatedAt time.Time
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
