package models

type CreateACMRequest struct {
	FullName     string `json:"full_name" validate:"required,max=150"`
	EmployeeID   string `json:"employee_id"`
	RoleID       string `json:"role_id"`
	DepartmentID string `json:"department_id"`
	Status       string `json:"status" validate:"required"`
}

type UpdateACMRequest struct {
	FullName     string `json:"full_name" validate:"required,max=150"`
	EmployeeID   string `json:"employee_id"`
	RoleID       string `json:"role_id"`
	DepartmentID string `json:"department_id"`
	Status       string `json:"status" validate:"required"`
}
