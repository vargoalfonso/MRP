package models

type CreateOrderRequest struct {
	DocumentType    string            `json:"document_type" validate:"required,oneof=PO DN SO"`
	CustomerID      int64             `json:"customer_id" validate:"required,gt=0"`
	ContactPerson   string            `json:"contact_person" validate:"omitempty,max=255"`
	DeliveryAddress string            `json:"delivery_address" validate:"omitempty"`
	DeliveryDate    string            `json:"delivery_date" validate:"omitempty"`
	Notes           string            `json:"notes" validate:"omitempty"`
	Items           []CreateItemInput `json:"items" validate:"required,min=1,dive"`
}

type CreateItemInput struct {
	ItemUniqCode string  `json:"item_uniq_code" validate:"required"`
	Quantity     float64 `json:"quantity" validate:"required,gt=0"`
	DeliveryDate string  `json:"delivery_date" validate:"omitempty"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=active completed cancelled"`
}

type ListOrderQuery struct {
	Search       string `form:"search"`
	DocumentType string `form:"document_type"`
	Status       string `form:"status"`
	CustomerID   int64  `form:"customer_id"`
	Period       string `form:"period"`
	Page         int    `form:"page"`
	Limit        int    `form:"limit"`
}

type ListFilters struct {
	Search       string
	DocumentType string
	Status       string
	CustomerID   int64
	Period       string
	Page         int
	Limit        int
	Offset       int
}
