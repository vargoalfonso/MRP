package models

import "time"

// ScanReturnRequest is the payload from the Product Return page when a QR /
// packing list is scanned.
type ScanReturnRequest struct {
	QRCodeValue string `json:"qr_code_value"`
}

// ScanReturnResponse auto-fills the Product Return form.
type ScanReturnResponse struct {
	Uniq          string `json:"uniq"`
	UniqID        string `json:"uniq_id"`
	PartNumber    string `json:"part_number"`
	PartName      string `json:"part_name"`
	Model         string `json:"model"`
	PackingNumber string `json:"packing_number"`
	ScrapType     string `json:"scrap_type"`
	SelectedUnit  string `json:"selected_unit"`
}

// SubmitReturnToQCRequest is submitted when the operator sends a return to QC.
type SubmitReturnToQCRequest struct {
	KanbanPackingList string   `json:"kanban_packing_list"`
	Uniq              string   `json:"uniq"`
	PartNumber        *string  `json:"part_number"`
	PartName          *string  `json:"part_name"`
	Model             *string  `json:"model"`
	PackingNumber     string   `json:"packing_number"`
	DNNumber          string   `json:"dn_number"`
	DateReceived      string   `json:"date_received"`
	ScrapType         string   `json:"scrap_type"`
	QuantityScrap     int      `json:"quantity_scrap"`
	Weight            *float64 `json:"weight"`
	UnitMeasurement   string   `json:"unit_measurement"`
	QuantityRework    int      `json:"quantity_rework"`
}

// ProductReturnRow maps to the product_returns table (BRD-extended).
type ProductReturnRow struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	Uniq           string     `gorm:"type:varchar(100);not null" json:"uniq"`
	DNNumber       string     `gorm:"type:varchar(100);not null" json:"dn_number"`
	QuantityScrap  int        `gorm:"default:0" json:"quantity_scrap"`
	QuantityRework int        `gorm:"default:0" json:"quantity_rework"`
	Weight         float64    `gorm:"column:weight;default:0" json:"weight"`
	Uom            string     `gorm:"column:uom;type:varchar(50)" json:"uom"`
	DateReceived   *time.Time `gorm:"column:date_received" json:"date_received"`
	ScrapType      string     `gorm:"column:scrap_type;type:varchar(50);default:'Product Return'" json:"scrap_type"`
	Status         string     `gorm:"type:varchar(50);default:'PENDING'" json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (ProductReturnRow) TableName() string { return "product_returns" }

// PendingReturnTask is one row in the QC "pending validation" list.
type PendingReturnTask struct {
	ReturnID          string  `json:"return_id"`
	KanbanPackingList string  `json:"kanban_packing_list"`
	PackingNumber     string  `json:"packing_number"`
	Uniq              string  `json:"uniq"`
	UniqID            string  `json:"uniq_id"`
	PartNumber        string  `json:"part_number"`
	PartName          string  `json:"part_name"`
	Model             string  `json:"model"`
	QuantityScrap     int     `json:"quantity_scrap"`
	Weight            float64 `json:"weight"`
	UnitMeasurement   string  `json:"unit_measurement"`
	ScrapType         string  `json:"scrap_type"`
}

// SubmitReturnValidationRequest is submitted by QC when validating a return.
type SubmitReturnValidationRequest struct {
	ReturnID  string `json:"return_id"`
	Action    string `json:"action"` // PASS | NOT_PASS
	Status    string `json:"status"` // approved | rejected
	MovedToFG bool   `json:"moved_to_fg"`
	Quantity  int    `json:"quantity"`
}
