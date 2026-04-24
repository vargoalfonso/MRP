package models

type ListTasksRequest struct {
	TaskType string `json:"task_type"`
	Status   string `json:"status"`
}

type StartTaskResponse struct {
	TaskID int64  `json:"task_id"`
	Status string `json:"status"`
}

type ApproveIncomingTaskRequest struct {
	NumberOfDefects int           `json:"number_of_defects"`
	DateChecked     string        `json:"date_checked" binding:"required"` // format: YYYY-MM-DD
	Defects         []DefectInput `json:"defects"`
}

type RejectIncomingTaskRequest struct {
	NumberOfDefects int           `json:"number_of_defects" binding:"required"`
	DateChecked     string        `json:"date_checked" binding:"required"` // format: YYYY-MM-DD
	Defects         []DefectInput `json:"defects"`
}

type DefectInput struct {
	ReasonCode   string  `json:"reason_code"`
	ReasonText   string  `json:"reason_text"`
	QtyDefect    float64 `json:"qty_defect"`
	QtyScrap     float64 `json:"qty_scrap"`
	IsRepairable bool    `json:"is_repairable"`
}
