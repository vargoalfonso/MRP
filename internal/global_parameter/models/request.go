package models

type CreateGlobalParameterRequest struct {
	ParameterGroup string `json:"parameter_group" validate:"required"`
	Period         string `json:"period" validate:"required"` // contoh: 2026-04
	WorkingDays    int    `json:"working_days" validate:"required,gte=0"`
	Status         string `json:"status" validate:"required,oneof=active inactive"`
}

type UpdateGlobalParameterRequest struct {
	ParameterGroup string `json:"parameter_group"`
	Period         string `json:"period"`
	WorkingDays    int    `json:"working_days" validate:"omitempty,gte=0"`
	Status         string `json:"status" validate:"omitempty,oneof=active inactive"`
}
