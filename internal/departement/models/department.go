package models

import (
	"time"
)

type Department struct {
	ID                 int64     `json:"id" db:"id"`
	DepartmentCode     string    `json:"department_code" db:"department_code"`
	DepartmentName     string    `json:"department_name" db:"department_name"`
	Description        *string   `json:"description,omitempty" db:"description"`
	ManagerID          *int64    `json:"manager_id,omitempty" db:"manager_id"`
	ParentDepartmentID *string   `json:"parent_department_id,omitempty" db:"parent_department_id"`
	Status             string    `json:"status" db:"status"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}
