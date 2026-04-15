package inventoryconst

// SourceFlag is a normalized origin marker for inventory_movement_logs.
//
// Keep these values stable once used in production.
type SourceFlag string

const (
	SourceManual           SourceFlag = "manual"
	SourceIncomingScan     SourceFlag = "incoming_scan"
	SourceQCApprove        SourceFlag = "qc_approve"
	SourceWOApprove        SourceFlag = "wo_approve"        // current implementation: deduct on WO approve
	SourceWOReserve        SourceFlag = "wo_reserve"        // future: reserve/issue to floor (optional)
	SourceWOConsumeActual  SourceFlag = "wo_consume_actual" // future: actual RM used from scan-out
	SourceProductionReject SourceFlag = "production_reject" // reserved for future use
	SourceStockOpname      SourceFlag = "stock_opname"

	// Legacy values (avoid introducing new usage; kept for backward compatibility)
	SourceWOScan     SourceFlag = "wo_scan"
	SourceProduction SourceFlag = "production"
)

type MovementType string

const (
	MovementIncoming           MovementType = "incoming"
	MovementOutgoing           MovementType = "outgoing"
	MovementStockOpname        MovementType = "stock_opname"
	MovementAdjustment         MovementType = "adjustment"
	MovementReceivedFromVendor MovementType = "received_from_vendor"
)

type MovementCategory string

const (
	CategoryRawMaterial      MovementCategory = "raw_material"
	CategoryIndirectMaterial MovementCategory = "indirect_raw_material"
	CategorySubcon           MovementCategory = "subcon"
)
