package models

type CreateUnitRequest struct {
	Code     string `json:"code" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Category string `json:"category" binding:"required"`
	Status   string `json:"status"`
}

type UpdateUnitRequest struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Status   string `json:"status"`
}
