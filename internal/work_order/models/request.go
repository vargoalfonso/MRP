package models

// CreateWorkOrderRequest is the body for POST /api/v1/working-order/work-orders.
type CreateWorkOrderRequest struct {
	// Optional: allow FE to reuse preview wo_id.
	WOID        *string               `json:"wo_id"`
	WOType      string                `json:"wo_type" validate:"required,oneof=New Assembly Rework Addendum"`
	ReferenceWO *string               `json:"reference_wo"`
	CreatedDate *string               `json:"created_date"` // YYYY-MM-DD (optional)
	TargetDate  *string               `json:"target_date"`  // YYYY-MM-DD
	Items       []CreateWorkOrderItem `json:"items" validate:"required,min=1,dive"`
	Notes       *string               `json:"notes"`
}

type CreateWorkOrderItem struct {
	ItemUniqCode string  `json:"item_uniq_code" validate:"required"`
	Quantity     float64 `json:"quantity" validate:"required,gt=0"`
	UOM          *string `json:"uom"`
	ProcessName  *string `json:"process_name"`
	KanbanQty    int     `json:"kanban_qty"` // pcs per kanban/batch; if 0, auto from kanban_parameters
}

// WorkOrderApprovalRequest is the body for POST /api/v1/working-order/work-orders/:id/approval.
type WorkOrderApprovalRequest struct {
	Decision string  `json:"decision" validate:"required,oneof=approve reject"`
	Notes    *string `json:"notes"`
}

// BulkWorkOrderApprovalRequest is the body for POST /api/v1/working-order/work-orders/bulk-approval.
type BulkWorkOrderApprovalRequest struct {
	Decision  string   `json:"decision" validate:"required,oneof=approve reject"`
	WONumbers []string `json:"wo_numbers" validate:"required,min=1"`
	Notes     *string  `json:"notes"`
}

// CreateRMProcessingWorkOrderRequest is the body for POST /api/v1/working-order/rm-processing/work-orders.
type CreateRMProcessingWorkOrderRequest struct {
	SourceMaterialUniq string  `json:"source_material_uniq" validate:"required"`
	TargetMaterialUniq string  `json:"target_material_uniq" validate:"required"`
	Model              *string `json:"model"`
	GradeSize          *string `json:"grade_size"`
	InputQty           float64 `json:"input_qty" validate:"required,gt=0"`
	InputUOM           string  `json:"input_uom" validate:"required"`
	OutputQty          float64 `json:"output_qty" validate:"required,gt=0"`
	OutputUOM          string  `json:"output_uom" validate:"required"`
	DateIssued         *string `json:"date_issued"`
	Remarks            *string `json:"remarks"`
}
