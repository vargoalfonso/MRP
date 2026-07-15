package models

type CreateProductReturnRequest struct {
	Uniq           string  `json:"uniq" validate:"required"`
	DNNumber       string  `json:"dn_number" validate:"required"`
	QuantityScrap  int     `json:"quantity_scrap"`
	QuantityRework int     `json:"quantity_rework"`
	Status         string  `json:"status"`
	Weight         float64 `json:"weight"`
	UOM            string  `json:"uom"`
	DateReceived   string  `json:"date_received"`
	ScrapType      string  `json:"scrap_type"`
}

type UpdateProductReturnRequest struct {
	Uniq           string  `json:"uniq"`
	DNNumber       string  `json:"dn_number"`
	QuantityScrap  int     `json:"quantity_scrap"`
	QuantityRework int     `json:"quantity_rework"`
	Status         string  `json:"status"`
	Weight         float64 `json:"weight"`
	UOM            string  `json:"uom"`
	DateReceived   string  `json:"date_received"`
	ScrapType      string  `json:"scrap_type"`
}
