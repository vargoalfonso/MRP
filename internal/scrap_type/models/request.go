package models

type CreateScrapTypeRequest struct {
	Name        string  `json:"name" validate:"required,max=128"`
	Description *string `json:"description"`
	Status      string  `json:"status" validate:"omitempty,oneof=Active Inactive"`
}

type UpdateScrapTypeRequest struct {
	Name        *string `json:"name" validate:"omitempty,max=128"`
	Description *string `json:"description"`
	Status      *string `json:"status" validate:"omitempty,oneof=Active Inactive"`
}
