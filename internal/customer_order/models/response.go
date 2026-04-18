package models

import "time"

type DocumentResponse struct {
	ID             string     `json:"id"`
	DocumentType   string     `json:"document_type"`
	DocumentNumber string     `json:"document_number"`
	DocumentDate   time.Time  `json:"document_date"`
	PeriodSchedule string     `json:"period_schedule"`
	CustomerID     int64      `json:"customer_id"`
	CustomerName   string     `json:"customer_name"`
	ContactPerson  *string    `json:"contact_person"`
	DeliveryAddress *string   `json:"delivery_address"`
	Status         string     `json:"status"`
	Notes          *string    `json:"notes"`
	CreatedBy      string     `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	Items          []ItemResponse `json:"items,omitempty"`
}

type ItemResponse struct {
	ID           string     `json:"id"`
	LineNo       int        `json:"line_no"`
	ItemUniqCode string     `json:"item_uniq_code"`
	PartName     string     `json:"part_name"`
	PartNumber   string     `json:"part_number"`
	Model        *string    `json:"model"`
	Quantity     float64    `json:"quantity"`
	DeliveryDate *time.Time `json:"delivery_date"`
}

type ListResponse struct {
	Items      []DocumentResponse `json:"items"`
	Pagination PaginationMeta     `json:"pagination"`
}

type PaginationMeta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

func ToDocumentResponse(d *CustomerOrderDocument) DocumentResponse {
	resp := DocumentResponse{
		ID:              d.UUID,
		DocumentType:    d.DocumentType,
		DocumentNumber:  d.DocumentNumber,
		DocumentDate:    d.DocumentDate,
		PeriodSchedule:  d.PeriodSchedule,
		CustomerID:      d.CustomerID,
		CustomerName:    d.CustomerNameSnapshot,
		ContactPerson:   d.ContactPerson,
		DeliveryAddress: d.DeliveryAddress,
		Status:          d.Status,
		Notes:           d.Notes,
		CreatedBy:       d.CreatedBy,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}

	for _, item := range d.Items {
		resp.Items = append(resp.Items, ItemResponse{
			ID:           item.UUID,
			LineNo:       item.LineNo,
			ItemUniqCode: item.ItemUniqCode,
			PartName:     item.PartName,
			PartNumber:   item.PartNumber,
			Model:        item.Model,
			Quantity:     item.Quantity,
			DeliveryDate: item.DeliveryDate,
		})
	}

	return resp
}

func NewPaginationMeta(page, limit int, total int64) PaginationMeta {
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}
	return PaginationMeta{Total: total, Page: page, Limit: limit, TotalPages: totalPages}
}
