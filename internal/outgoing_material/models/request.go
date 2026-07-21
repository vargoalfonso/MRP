package models

type CreateOutgoingRMRequest struct {
	PackingListRM       *string `json:"packing_list_rm"`
	Uniq                string  `json:"uniq" validate:"required"`
	Unit                *string `json:"unit"`
	QuantityOut         float64 `json:"quantity_out" validate:"required,gt=0"`
	Reason              string  `json:"reason" validate:"required"`
	Purpose             *string `json:"purpose"`
	WorkOrderNo         *string `json:"work_order_no"`
	DestinationLocation *string `json:"destination_location"`
	RequestedBy         *string `json:"requested_by"`
	Remarks             *string `json:"remarks"`
}

// UpdateOutgoingRMRequest is the body for PUT /api/v1/outgoing-raw-materials/:id.
// All fields are optional (pointer) so callers can send a partial update.
// When QuantityOut and/or Uniq change, stock in raw_materials is re-calculated
// automatically inside a single DB transaction.
type UpdateOutgoingRMRequest struct {
	Uniq                *string  `json:"uniq"`
	Unit                *string  `json:"unit"`
	PackingListRM       *string  `json:"packing_list_rm"`
	QuantityOut         *float64 `json:"quantity_out" validate:"omitempty,gt=0"`
	Reason              *string  `json:"reason"`
	Purpose             *string  `json:"purpose"`
	WorkOrderNo         *string  `json:"work_order_no"`
	DestinationLocation *string  `json:"destination_location"`
	RequestedBy         *string  `json:"requested_by"`
	Remarks             *string  `json:"remarks"`
}
