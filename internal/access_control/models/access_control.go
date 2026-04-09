package models

import (
	"time"
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
