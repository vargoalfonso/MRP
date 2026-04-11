package models

type CreateDNRequest struct {
	PONumber      string `json:"po_number" validate:"required"`
	CustomerID    int64  `json:"customer_id"`
	ContactPerson string `json:"contact_person"`
	Period        string `json:"period"`
	IncomingDate  string `json:"incoming_date"`
	Type          string `json:"type"`
}

type CreateDNItemRequest struct {
	ItemUniqCode string `json:"item_uniq_code" validate:"required"`
	Quantity     int    `json:"quantity" validate:"required"`
	KanbanID     int64  `json:"kanban_id"`
}
