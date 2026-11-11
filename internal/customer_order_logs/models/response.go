package models

import "time"

type LogResponse struct {
	ID                  string    `json:"id"`
	DocumentNumber      string    `json:"document_number"`
	RowNo               int       `json:"row_no"`
	ItemUniqCode        string    `json:"item_uniq_code"`
	PartName            string    `json:"part_name"`
	Description         string    `json:"description"`
	QtyActive           float64   `json:"qty_active"`
	FailureReason       string    `json:"failure_reason"`
	SpecialInstructions string    `json:"special_instructions"`
	Source              string    `json:"source"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
}

type PaginationMeta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

type ListLogResponse struct {
	Items      []LogResponse  `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
}

func ToLogResponse(m *CustomerOrderAutomationLog) LogResponse {
	return LogResponse{
		ID:                  m.UUID,
		DocumentNumber:      m.DocumentNumber,
		RowNo:               m.RowNo,
		ItemUniqCode:        m.ItemUniqCode,
		PartName:            m.PartName,
		Description:         m.Description,
		QtyActive:           m.QtyActive,
		FailureReason:       m.FailureReason,
		SpecialInstructions: m.SpecialInstructions,
		Source:              m.Source,
		Status:              m.Status,
		CreatedAt:           m.CreatedAt,
	}
}

func NewPaginationMeta(page, limit int, total int64) PaginationMeta {
	if limit <= 0 {
		limit = 20
	}
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}
	return PaginationMeta{Total: total, Page: page, Limit: limit, TotalPages: totalPages}
}
