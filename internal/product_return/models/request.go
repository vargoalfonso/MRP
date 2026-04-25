package models

type CreateProductReturnRequest struct {
	Uniq           string `json:"uniq" validate:"required"`
	DNNumber       string `json:"dn_number" validate:"required"`
	QuantityScrap  int    `json:"quantity_scrap"`
	QuantityRework int    `json:"quantity_rework"`
	Status         string `json:"status"`
}

type UpdateProductReturnRequest struct {
	Uniq           string `json:"uniq"`
	DNNumber       string `json:"dn_number"`
	QuantityScrap  int    `json:"quantity_scrap"`
	QuantityRework int    `json:"quantity_rework"`
	Status         string `json:"status"`
}
