package models

type CreateDepartmentRequest struct {
	DepartmentCode     string  `json:"department_code" validate:"required,max=50"`
	DepartmentName     string  `json:"department_name" validate:"required,max=100"`
	Description        *string `json:"description"`
	ManagerID          *int64  `json:"manager_id"`
	ParentDepartmentID *int64  `json:"parent_department_id"`
	Status             string  `json:"status" validate:"required"`
}

type UpdateDepartmentRequest struct {
	DepartmentCode     string  `json:"department_code" validate:"required,max=50"`
	DepartmentName     string  `json:"department_name" validate:"required,max=100"`
	Description        *string `json:"description"`
	ManagerID          *int64  `json:"manager_id"`
	ParentDepartmentID *int64  `json:"parent_department_id"`
	Status             string  `json:"status" validate:"required"`
}
