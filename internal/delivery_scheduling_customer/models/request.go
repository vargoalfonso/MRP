package models

// ─── Schedule Requests ────────────────────────────────────────────────────────

type CreateScheduleRequest struct {
	CustomerOrderDocumentUUID string               `json:"customer_order_document_uuid"`
	CustomerOrderReference    string               `json:"customer_order_reference"`
	CustomerID              int64                `json:"customer_id" validate:"required,gt=0"`
	CustomerName            string               `json:"customer_name" validate:"required"`
	DeliveryDate            string               `json:"delivery_date" validate:"required"`
	Cycle                   string               `json:"cycle"`
	Priority                string               `json:"priority"`
	TransportCompany        string               `json:"transport_company"`
	VehicleNumber           string               `json:"vehicle_number"`
	DriverName              string               `json:"driver_name"`
	DriverContact           string               `json:"driver_contact"`
	DepartureAt             string               `json:"departure_at"`
	ArrivalAt               string               `json:"arrival_at"`
	DeliveryInstructions    string               `json:"delivery_instructions"`
	Remarks                 string               `json:"remarks"`
	Items                   []CreateScheduleItem `json:"items" validate:"required,min=1,dive"`
}

type CreateScheduleItem struct {
	// CustomerOrderDocumentItemUUID is the UUID of the source item line from customer_order_document_items.
	// When provided, service auto-resolves part_no, part_name, model, uom, total_order from DB.
	CustomerOrderDocumentItemUUID string  `json:"customer_order_document_item_uuid"`
	ItemUniqCode                  string  `json:"item_uniq_code" validate:"required"`
	PartNo                        string  `json:"part_no"`
	PartName                      string  `json:"part_name"`
	Model                         string  `json:"model"`
	TotalOrder                    float64 `json:"total_order"`
	TotalDelivery                 float64 `json:"total_delivery" validate:"required,gt=0"`
	UOM                           string  `json:"uom"`
}

type ApproveScheduleRequest struct {
	Notes        string `json:"notes"`
	ForcePartial bool   `json:"force_partial"`
}

type ApproveAllRequest struct {
	DeliveryDate string `json:"delivery_date" validate:"required"`
	CustomerID   int64  `json:"customer_id"`
	Notes        string `json:"notes"`
	ForcePartial bool   `json:"force_partial"`
}

type ApprovePartialRequest struct {
	DeliveryDate string   `json:"delivery_date" validate:"required"`
	ScheduleIDs  []string `json:"schedule_ids" validate:"required,min=1"`
	Notes        string   `json:"notes"`
	ForcePartial bool     `json:"force_partial"`
}

// ─── Customer DN Requests ─────────────────────────────────────────────────────

type CreateCustomerDNRequest struct {
	ScheduleID             string                   `json:"schedule_id"`
	ScheduleDate           string                   `json:"schedule_date"`
	CustomerID             int64                    `json:"customer_id" validate:"required,gt=0"`
	CustomerName           string                   `json:"customer_name" validate:"required"`
	PONumber               string                   `json:"po_number"`
	CustomerContactPerson  string                   `json:"customer_contact_person"`
	CustomerPhoneNumber    string                   `json:"customer_phone_number"`
	DeliveryAddress        string                   `json:"delivery_address"`
	DeliveryDate           string                   `json:"delivery_date" validate:"required"`
	Priority               string                   `json:"priority"`
	TransportCompany       string                   `json:"transport_company"`
	VehicleNumber          string                   `json:"vehicle_number"`
	DriverName             string                   `json:"driver_name"`
	DriverContact          string                   `json:"driver_contact"`
	DepartureAt            string                   `json:"departure_at"`
	ArrivalAt              string                   `json:"arrival_at"`
	Status                 string                   `json:"status"`
	ApprovalStatus         string                   `json:"approval_status"`
	DeliveryInstructions   string                   `json:"delivery_instructions"`
	Remarks                string                   `json:"remarks"`
	Items                  []CreateCustomerDNItem   `json:"items" validate:"required,min=1,dive"`
}

type CreateCustomerDNItem struct {
	ItemUniqCode string  `json:"item_uniq_code" validate:"required"`
	ProductName  string  `json:"product_name" validate:"required"`
	PartNumber   string  `json:"part_number" validate:"required"`
	Model        string  `json:"model"`
	FGLocation   string  `json:"fg_location"`
	Quantity     float64 `json:"quantity" validate:"required,gt=0"`
	UOM          string  `json:"uom" validate:"required"`
}

type ConfirmDNRequest struct {
	Notes string `json:"notes"`
}

// ─── Delivery Scan Requests ───────────────────────────────────────────────────

type SubmitScanRequest struct {
	ClientEventID string         `json:"client_event_id" validate:"required"`
	DNNumber      string         `json:"dn_number" validate:"required"`
	ItemUniqCode  string         `json:"item_uniq_code" validate:"required"`
	ScannedKanbans []ScannedKanban `json:"scanned_kanbans" validate:"required,min=1,dive"`
	DeliveryCycle string         `json:"delivery_cycle"`
}

type ScannedKanban struct {
	ProductionKanban string  `json:"production_kanban" validate:"required"`
	Qty              float64 `json:"qty" validate:"required,gt=0"`
}

// ─── List Filters ─────────────────────────────────────────────────────────────

type ScheduleListFilter struct {
	DeliveryDate string
	CustomerID   int64
	Status       string
	Page         int
	Limit        int
}

type DNListFilter struct {
	DeliveryDate string
	CustomerID   int64
	Status       string
	Page         int
	Limit        int
}
