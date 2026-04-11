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
	ApprovedQty int `json:"approved_qty" binding:"required"`
	NgQty       int `json:"ng_qty"`
	ScrapQty    int `json:"scrap_qty"`

	// notes and defects are stored as JSON in round_results for audit.
	Notes   *string       `json:"notes"`
	Defects []interface{} `json:"defects"`

	// Disposal: scrap | return_to_supplier | hold
	ScrapDisposition *string `json:"scrap_disposition"`
}

type RejectIncomingTaskRequest struct {
	RejectedQty int           `json:"rejected_qty" binding:"required"`
	Reason      string        `json:"reason" binding:"required"`
	Defects     []interface{} `json:"defects"`

	// return_to_supplier | scrap | hold
	Disposition *string `json:"disposition"`
}
