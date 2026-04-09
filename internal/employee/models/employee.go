package models

import (
	"time"
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
