package models

type CreateACMRequest struct {
	FullName     string `json:"full_name" validate:"required,max=150"`
	EmployeeID   *int64 `json:"employee_id"`
	RoleID       *int64 `json:"role_id"`
	DepartmentID *int64 `json:"department_id"`
	Status       string `json:"status" validate:"required"`
}

type UpdateACMRequest struct {
	FullName     string `json:"full_name" validate:"required,max=150"`
	EmployeeID   *int64 `json:"employee_id"`
	RoleID       *int64 `json:"role_id"`
	DepartmentID *int64 `json:"department_id"`
	Status       string `json:"status" validate:"required"`
}
