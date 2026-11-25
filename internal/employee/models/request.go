package models

import "time"

type CreateEmployeeRequest struct {
	FullName     string     `json:"full_name" binding:"required,max=150"`
	Email        string     `json:"email" binding:"required,email,max=150"`
	PhoneNumber  string     `json:"phone_number" binding:"max=50"`
	JobTitle     string     `json:"job_title" binding:"max=100"`
	UnitCost     float64    `json:"unit_cost"`
	JoinDate     *time.Time `json:"join_date"`
	RoleID       int64      `json:"role_id" binding:"required"`
	DepartmentID *int64     `json:"department_id"`
	ReportsToID  *int64     `json:"reports_to_id"`
	Status       string     `json:"status" binding:"required,oneof=active inactive"`
	Notes        string     `json:"notes"`
}

type UpdateEmployeeRequest struct {
	FullName     *string    `json:"full_name,omitempty"`
	Email        *string    `json:"email,omitempty" binding:"omitempty,email,max=150"`
	PhoneNumber  *string    `json:"phone_number,omitempty"`
	JobTitle     *string    `json:"job_title,omitempty"`
	UnitCost     *float64   `json:"unit_cost,omitempty"`
	JoinDate     *time.Time `json:"join_date,omitempty"`
	RoleID       *int64     `json:"role_id,omitempty"`
	DepartmentID *int64     `json:"department_id,omitempty"`
	ReportsToID  *int64     `json:"reports_to_id,omitempty"`
	Status       *string    `json:"status,omitempty" binding:"omitempty,oneof=active inactive"`
	Notes        *string    `json:"notes,omitempty"`
}
