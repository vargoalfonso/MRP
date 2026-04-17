package models

type CreateMachineRequest struct {
	MachineNumber   string `json:"machine_number" validate:"required"`
	MachineName     string `json:"machine_name" validate:"required"`
	ProductionLine  string `json:"production_line" validate:"required"`
	ProcessID       int64  `json:"process_id" validate:"required"`
	MachineCapacity int    `json:"machine_capacity" validate:"required"`
	Status          string `json:"status" validate:"required"`
}

type UpdateMachineRequest struct {
	MachineNumber   string `json:"machine_number" validate:"required"`
	MachineName     string `json:"machine_name" validate:"required"`
	ProductionLine  string `json:"production_line" validate:"required"`
	ProcessID       int64  `json:"process_id" validate:"required"`
	MachineCapacity int    `json:"machine_capacity" validate:"required"`
	Status          string `json:"status" validate:"required"`
}
