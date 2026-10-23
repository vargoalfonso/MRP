package models

type CreateLogRequest struct {
	WoNumber            string  `json:"wo_number" validate:"omitempty,max=64"`
	AutomationJobID     string  `json:"automation_job_id" validate:"omitempty,max=128"`
	RobotName           string  `json:"robot_name" validate:"omitempty,max=255"`
	RowNo               int     `json:"row_no" validate:"omitempty"`
	ItemUniqCode        string  `json:"item_uniq_code" validate:"omitempty,max=100"`
	PartName            string  `json:"part_name" validate:"omitempty,max=255"`
	Description         string  `json:"description" validate:"omitempty"`
	Qty                 float64 `json:"qty" validate:"omitempty"`
	FailureReason       string  `json:"failure_reason" validate:"required"`
	SpecialInstructions string  `json:"special_instructions" validate:"omitempty"`
	Source              string  `json:"source" validate:"omitempty,max=64"`
}

type ListLogQuery struct {
	Search          string `form:"search"`
	WoNumber        string `form:"wo_number"`
	AutomationJobID string `form:"automation_job_id"`
	Page            int    `form:"page"`
	Limit           int    `form:"limit"`
}
