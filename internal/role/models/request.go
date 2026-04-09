package models

type CreateRoleRequest struct {
	Name        string                 `json:"name" validate:"required"`
	Description string                 `json:"description"`
	Permissions map[string]interface{} `json:"permissions"`
	Status      string                 `json:"status"`
}

type UpdateRoleRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Permissions map[string]interface{} `json:"permissions"`
	Status      string                 `json:"status"`
}
