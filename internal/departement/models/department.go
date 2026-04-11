package models

import (
	"time"
)

type Department struct {
	ID                 int64     `json:"id"`
	DepartmentCode     string    `json:"department_code"`
	DepartmentName     string    `json:"department_name"`
	Description        *string   `json:"description,omitempty"`
	ManagerID          *int64    `json:"manager_id,omitempty"`
	ParentDepartmentID *int64    `json:"parent_department_id,omitempty"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	Parent *Department `json:"parent,omitempty" gorm:"-"`
}
