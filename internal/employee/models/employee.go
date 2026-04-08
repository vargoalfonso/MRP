package models

import "gorm.io/gorm"

// Employee is the GORM model for the public.employee table.
// Schema matches the existing migration (employee_id uuid PK).
type Employee struct {
	EmployeeID  string         `gorm:"primaryKey;column:employee_id;type:uuid;default:gen_random_uuid()" json:"employee_id"`
	Name        string         `gorm:"column:name"                                                        json:"name"`
	CompanyName string         `gorm:"column:company_name;not null"                                       json:"company_name"`
	Email       string         `gorm:"column:email"                                                       json:"email"`
	Address     string         `gorm:"column:address"                                                     json:"address"`
	DeletedAt   gorm.DeletedAt `gorm:"index"                                                              json:"-"`
}

// TableName overrides GORM's default to match the existing schema.
func (Employee) TableName() string { return "public.employee" }

// EmployeeReq is the validated request body for create/update operations.
type EmployeeReq struct {
	EmployeeID  string `json:"employee_id"`
	Name        string `json:"name"         validate:"required"`
	CompanyName string `json:"company_name" validate:"required"`
	Email       string `json:"email"        validate:"required,email"`
	Address     string `json:"address"`
}

// EmployeeResp is the public response shape returned to clients.
type EmployeeResp struct {
	EmployeeID  string `json:"employee_id"`
	Name        string `json:"name"`
	CompanyName string `json:"company_name"`
	Email       string `json:"email"`
	Address     string `json:"address"`
}
