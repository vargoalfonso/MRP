package models

type CreateProcessRequest struct {
	ProcessCode string `json:"process_code" validate:"required"`
	ProcessName string `json:"process_name" validate:"required"`
	Category    string `json:"category" validate:"required"`
	Sequence    int    `json:"sequence" validate:"required"`
	Status      string `json:"status" validate:"required"`
}

type UpdateProcessRequest struct {
	ProcessName string `json:"process_name"`
	Category    string `json:"category"`
	Sequence    int    `json:"sequence"`
	Status      string `json:"status"`
}
