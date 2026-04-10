package models

type CreateTypeRequest struct {
	TypeCode string `json:"type_code" binding:"required"`
	TypeName string `json:"type_name" binding:"required"`
	Status   string `json:"status"`
}

type UpdateTypeRequest struct {
	TypeName string `json:"type_name"`
	Status   string `json:"status"`
}
