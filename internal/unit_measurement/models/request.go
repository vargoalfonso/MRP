package models

type CreateUnitRequest struct {
	Name     string `json:"name" binding:"required"`
	Category string `json:"category" binding:"required"`
	Status   string `json:"status"`
}

type UpdateUnitRequest struct {
	Category string `json:"category"`
	Status   string `json:"status"`
}
