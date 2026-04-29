package models

type CreateMechinParameterRequest struct {
	MachineName    string `json:"machine_name" validate:"required,max=150"`
	MachineCount   int    `json:"machine_count" validate:"required,gt=0"`
	OperatingHours int    `json:"operating_hours" validate:"required,gt=0"`
	Status         string `json:"status" validate:"required,oneof=Active Inactive"`
}

type UpdateMechinParameterRequest struct {
	MachineName    string `json:"machine_name" validate:"omitempty,max=150"`
	MachineCount   *int   `json:"machine_count" validate:"omitempty,gt=0"`
	OperatingHours *int   `json:"operating_hours" validate:"omitempty,gt=0"`
	Status         string `json:"status" validate:"omitempty,oneof=Active Inactive"`
}
