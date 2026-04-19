package models

import "time"

type DocumentResponse struct {
	ID                 string         `json:"id"`
	DocumentType       string         `json:"document_type"`
	DocumentNumber     string         `json:"document_number"`
	DocumentDate       time.Time      `json:"document_date"`
	HeaderDeliveryDate *time.Time     `json:"header_delivery_date,omitempty"`
	PeriodSchedule     string         `json:"period_schedule"`
	CustomerID         int64          `json:"customer_id"`
	CustomerName       string         `json:"customer_name"`
	ContactPerson      *string        `json:"contact_person"`
	DeliveryAddress    *string        `json:"delivery_address"`
	Status             string         `json:"status"`
	Notes              *string        `json:"notes"`
	TotalQuantity      float64        `json:"total_quantity"`
	TotalUniq          int            `json:"total_uniq"`
	CreatedBy          string         `json:"created_by"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	Items              []ItemResponse `json:"items,omitempty"`
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

type SummaryMetric struct {
	TotalDocuments int64   `json:"total_documents"`
	TotalQuantity  float64 `json:"total_quantity"`
}

type SummaryResponse struct {
	DocumentType   string         `json:"document_type"`
	TotalDocuments int64          `json:"total_documents,omitempty"`
	TotalQuantity  float64        `json:"total_quantity"`
	DN             *SummaryMetric `json:"dn,omitempty"`
	PO             *SummaryMetric `json:"po,omitempty"`
	SO             *SummaryMetric `json:"so,omitempty"`
	Total          *SummaryMetric `json:"total,omitempty"`
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

	uniqCodes := make(map[string]struct{}, len(d.Items))
	if d.DocumentType == "DN" {
		resp.HeaderDeliveryDate = &d.DocumentDate
	}

	for _, item := range d.Items {
		resp.TotalQuantity += item.Quantity
		uniqCodes[item.ItemUniqCode] = struct{}{}
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
	resp.TotalUniq = len(uniqCodes)

	return resp
}

func NewPaginationMeta(page, limit int, total int64) PaginationMeta {
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}
	return PaginationMeta{Total: total, Page: page, Limit: limit, TotalPages: totalPages}
}
