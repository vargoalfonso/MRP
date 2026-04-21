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
