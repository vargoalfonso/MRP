package models

type CreateApprovalWorkflowRequest struct {
	ActionName string `json:"action_name" validate:"required"`
	Level1Role string `json:"level_1_role" validate:"required"`
	Level2Role string `json:"level_2_role"`
	Level3Role string `json:"level_3_role"`
	Level4Role string `json:"level_4_role"`
	Status     string `json:"status" validate:"required,oneof=active inactive"`
	CreatedBy  string `json:"created_by" validate:"required"`
}

type UpdateApprovalWorkflowRequest struct {
	ActionName string `json:"action_name"`
	Level1Role string `json:"level_1_role"`
	Level2Role string `json:"level_2_role"`
	Level3Role string `json:"level_3_role"`
	Level4Role string `json:"level_4_role"`
	Status     string `json:"status" validate:"omitempty,oneof=active inactive"`
}
