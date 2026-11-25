package models

type ImportDataRequest struct {
	CustomerName   string  `json:"customer_name"`
	UniqCode       string  `json:"uniq_code"`
	ProductModel   string  `json:"product_model"`
	PartName       string  `json:"part_name"`
	PartNumber     string  `json:"part_number"`
	ForecastPeriod string  `json:"forecast_period"`
	Quantity       float64 `json:"quantity"`
}
