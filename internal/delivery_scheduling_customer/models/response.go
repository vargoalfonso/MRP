package models

import "time"

// ─── Schedule Responses ───────────────────────────────────────────────────────

type CreateScheduleResponse struct {
	ScheduleID string `json:"schedule_id"`
	Status     string `json:"status"`
}

type ScheduleSummaryResponse struct {
	TotalDeliveries  int `json:"total_deliveries"`
	InTransit        int `json:"in_transit"`
	PendingApproval  int `json:"pending_approval"`
	DNCreated        int `json:"dn_created"`
}

type ScheduleListResponse struct {
	Groups     []ScheduleGroup `json:"groups"`
	Pagination Pagination      `json:"pagination"`
}

type ScheduleGroup struct {
	DeliveryDate string           `json:"delivery_date"`
	ItemCount    int              `json:"item_count"`
	Actions      GroupActions     `json:"actions"`
	Items        []ScheduleRow    `json:"items"`
}

type GroupActions struct {
	ApproveAllEnabled     bool `json:"approve_all_enabled"`
	ApprovePartialEnabled bool `json:"approve_partial_enabled"`
}

type ScheduleRow struct {
	ScheduleID     string  `json:"schedule_id"`
	CustomerName   string  `json:"customer_name"`
	PODNName       string  `json:"po_dn_name"`
	ItemUniqCode   string  `json:"item_uniq_code"`
	Model          string  `json:"model"`
	PartNo         string  `json:"part_no"`
	PartName       string  `json:"part_name"`
	Quantity       float64 `json:"quantity"`
	Cycle          string  `json:"cycle"`
	DNNumber       *string `json:"dn_number"`
	Status         string  `json:"status"`
	ApprovalStatus string  `json:"approval_status"`
}

type ScheduleDetailResponse struct {
	ScheduleID           string                `json:"schedule_id"`
	ScheduleDate         string                `json:"schedule_date"`
	DeliveryDate         string                `json:"delivery_date"`
	CustomerID           int64                 `json:"customer_id"`
	CustomerName         string                `json:"customer_name"`
	PONumber             string                `json:"po_number"`
	CustomerContactPerson string               `json:"customer_contact_person"`
	CustomerPhoneNumber  string                `json:"customer_phone_number"`
	DeliveryAddress      string                `json:"delivery_address"`
	TotalItems           int                   `json:"total_items"`
	TotalQuantity        float64               `json:"total_quantity"`
	Priority             string                `json:"priority"`
	Status               string                `json:"status"`
	ApprovalStatus       string                `json:"approval_status"`
	CreatedBy            string                `json:"created_by"`
	TransportCompany     string                `json:"transport_company"`
	VehicleNumber        string                `json:"vehicle_number"`
	DriverName           string                `json:"driver_name"`
	DriverContact        string                `json:"driver_contact"`
	DepartureAt          *time.Time            `json:"departure_at"`
	ArrivalAt            *time.Time            `json:"arrival_at"`
	DeliveryInstructions string                `json:"delivery_instructions"`
	Items                []ScheduleDetailItem  `json:"items"`
}

type ScheduleDetailItem struct {
	ItemUniqCode       string  `json:"item_uniq_code"`
	PartName           string  `json:"part_name"`
	DNNumber           string  `json:"dn_number"`
	Quantity           float64 `json:"quantity"`
	UOM                string  `json:"uom"`
	FGAvailable        float64 `json:"fg_available"`
	RemainingToPrepare float64 `json:"remaining_to_prepare"`
	Readiness          string  `json:"readiness"`
}

type ApproveScheduleResponse struct {
	ScheduleID string `json:"schedule_id"`
	DNID       string `json:"dn_id"`
	DNNumber   string `json:"dn_number"`
	Status     string `json:"status"`
}

type ApproveMultiResponse struct {
	ApprovedCount int    `json:"approved_count"`
	DNCreatedCount int   `json:"dn_created_count"`
	DeliveryDate  string `json:"delivery_date"`
}

// ─── Customer DN Responses ────────────────────────────────────────────────────

type CreateDNResponse struct {
	DNID           string              `json:"dn_id"`
	ScheduleID     string              `json:"schedule_id"`
	DNNumber       string              `json:"dn_number"`
	TotalItems     int                 `json:"total_items"`
	TotalQuantity  float64             `json:"total_quantity"`
	Status         string              `json:"status"`
	ApprovalStatus string              `json:"approval_status"`
	CreatedBy      string              `json:"created_by"`
	PrintedCount   int                 `json:"printed_count"`
	Items          []CreateDNItemResp  `json:"items"`
}

type CreateDNItemResp struct {
	DNItemID      string  `json:"dn_item_id"`
	DNNumber      string  `json:"dn_number"`
	ItemUniqCode  string  `json:"item_uniq_code"`
	PartNumber    string  `json:"part_number"`
	Model         string  `json:"model"`
	Quantity      float64 `json:"quantity"`
	FGLocation    string  `json:"fg_location"`
	PackingNumber string  `json:"packing_number"`
	QR            string  `json:"qr"`
}

type DNListResponse struct {
	Summary    ScheduleSummaryResponse `json:"summary"`
	Items      []DNListRow             `json:"items"`
	Pagination Pagination              `json:"pagination"`
}

type DNListRow struct {
	DNID              string  `json:"dn_id"`
	DNNumber          string  `json:"dn_number"`
	DeliveryDate      string  `json:"delivery_date"`
	CustomerName      string  `json:"customer_name"`
	PONumber          string  `json:"po_number"`
	PartName          string  `json:"part_name"`
	ItemUniqCode      string  `json:"item_uniq_code"`
	PartNumber        string  `json:"part_number"`
	Quantity          float64 `json:"quantity"`
	FGLocation        string  `json:"fg_location"`
	QRCode            string  `json:"qr_code"`
	PackingListNumber string  `json:"packing_list_number"`
	Status            string  `json:"status"`
	PrintedCount      int     `json:"printed_count"`
}

type DNDetailResponse struct {
	DNID                  string          `json:"dn_id"`
	DNNumber              string          `json:"dn_number"`
	CustomerID            int64           `json:"customer_id"`
	CustomerName          string          `json:"customer_name"`
	PONumber              string          `json:"po_number"`
	CustomerContactPerson string          `json:"customer_contact_person"`
	CustomerPhoneNumber   string          `json:"customer_phone_number"`
	DeliveryAddress       string          `json:"delivery_address"`
	DeliveryDate          string          `json:"delivery_date"`
	Priority              string          `json:"priority"`
	Status                string          `json:"status"`
	ApprovalStatus        string          `json:"approval_status"`
	TransportCompany      string          `json:"transport_company"`
	VehicleNumber         string          `json:"vehicle_number"`
	DriverName            string          `json:"driver_name"`
	DriverContact         string          `json:"driver_contact"`
	DepartureAt           *time.Time      `json:"departure_at"`
	ArrivalAt             *time.Time      `json:"arrival_at"`
	DeliveryInstructions  string          `json:"delivery_instructions"`
	TotalItems            int             `json:"total_items"`
	TotalQuantity         float64         `json:"total_quantity"`
	PrintedCount          int             `json:"printed_count"`
	CreatedBy             string          `json:"created_by"`
	Items                 []DNDetailItem  `json:"items"`
}

type DNDetailItem struct {
	DNItemID      string  `json:"dn_item_id"`
	ItemUniqCode  string  `json:"item_uniq_code"`
	PartName      string  `json:"part_name"`
	Quantity      float64 `json:"quantity"`
	UOM           string  `json:"uom"`
	RemainingQty  float64 `json:"remaining_qty"`
	PackingNumber string  `json:"packing_number"`
	QR            string  `json:"qr"`
}

type ConfirmDNResponse struct {
	DNID   string `json:"dn_id"`
	Status string `json:"status"`
}

// ─── Delivery Scan Responses ──────────────────────────────────────────────────

type DeliveryLookupResponse struct {
	DNID            string  `json:"dn_id"`
	DNItemID        string  `json:"dn_item_id"`
	DNNumber        string  `json:"dn_number"`
	PODNReference   string  `json:"po_dn_reference"`
	ItemUniqCode    string  `json:"item_uniq_code"`
	PartName        string  `json:"part_name"`
	Model           string  `json:"model"`
	PartNo          string  `json:"part_no"`
	PackingNumber   string  `json:"packing_number"`
	QuantityOrder   float64 `json:"quantity_order"`
	RemainingQty    float64 `json:"remaining_qty"`
	UOM             string  `json:"uom"`
	DeliveryDate    string  `json:"delivery_date"`
	DeliveryCycle   string  `json:"delivery_cycle"`
}

type SubmitScanResponse struct {
	DNNumber     string  `json:"dn_number"`
	ItemUniqCode string  `json:"item_uniq_code"`
	DeliveredQty float64 `json:"delivered_qty"`
	RemainingQty float64 `json:"remaining_qty"`
	DNStatus     string  `json:"dn_status"`
}
