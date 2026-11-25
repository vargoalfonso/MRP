package models

type CreateLogRequest struct {
	DocumentNumber      string  `json:"document_number" validate:"omitempty,max=128"`
	RowNo               int     `json:"row_no" validate:"omitempty"`
	ItemUniqCode        string  `json:"item_uniq_code" validate:"omitempty,max=100"`
	PartName            string  `json:"part_name" validate:"omitempty,max=255"`
	Description         string  `json:"description" validate:"omitempty"`
	QtyActive           float64 `json:"qty_active" validate:"omitempty"`
	FailureReason       string  `json:"failure_reason" validate:"required"`
	SpecialInstructions string  `json:"special_instructions" validate:"omitempty"`
	Source              string  `json:"source" validate:"omitempty,max=64"`
}

type ListLogQuery struct {
	Search         string `form:"search"`
	DocumentNumber string `form:"document_number"`
	Page           int    `form:"page"`
	Limit          int    `form:"limit"`
}
