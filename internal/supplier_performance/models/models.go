package models

import "time"

// Snapshot is the read model for one supplier per evaluation period.
type Snapshot struct {
	SnapshotUUID            string    `gorm:"column:snapshot_uuid" json:"snapshot_uuid"`
	SupplierUUID            string    `gorm:"column:supplier_uuid" json:"supplier_id"`
	SupplierCode            string    `gorm:"column:supplier_code" json:"supplier_code"`
	SupplierName            string    `gorm:"column:supplier_name" json:"supplier_name"`
	EvaluationPeriodType    string    `gorm:"column:evaluation_period_type" json:"evaluation_period_type"`
	EvaluationPeriodValue   string    `gorm:"column:evaluation_period_value" json:"evaluation_period_value"`
	EvaluationDate          time.Time `gorm:"column:evaluation_date" json:"evaluation_date"`
	TotalDeliveries         int       `gorm:"column:total_deliveries" json:"total_deliveries"`
	OnTimeDeliveries        int       `gorm:"column:on_time_deliveries" json:"on_time_deliveries"`
	LateDeliveries          int       `gorm:"column:late_deliveries" json:"late_deliveries"`
	OTDPercentage           float64   `gorm:"column:otd_percentage" json:"otd_percentage"`
	AverageDelayDays        float64   `gorm:"column:average_delay_days" json:"average_delay_days"`
	QualityInspectionCount  int       `gorm:"column:quality_inspection_count" json:"quality_inspection_count"`
	AcceptedQuantity        float64   `gorm:"column:accepted_quantity" json:"accepted_quantity"`
	RejectedQuantity        float64   `gorm:"column:rejected_quantity" json:"rejected_quantity"`
	InspectedQuantity       float64   `gorm:"column:inspected_quantity" json:"inspected_quantity"`
	QualityPercentage       float64   `gorm:"column:quality_percentage" json:"quality_percentage"`
	TotalPurchaseValue      float64   `gorm:"column:total_purchase_value" json:"total_purchase_value"`
	ComputedScore           float64   `gorm:"column:computed_score" json:"computed_score"`
	PerformanceGrade        string    `gorm:"column:final_grade" json:"performance_grade"`
	StatusLabel             string    `gorm:"column:status_label" json:"status_label"`
	PoorDeliveryPerformance bool      `gorm:"column:poor_delivery_performance" json:"poor_delivery_performance"`
	QCAlert                 bool      `gorm:"column:qc_alert" json:"qc_alert"`
	SupplierReviewRequired  bool      `gorm:"column:supplier_review_required" json:"supplier_review_required"`
	IsGradeOverridden       bool      `gorm:"column:is_grade_overridden" json:"is_grade_overridden"`
	OverrideGrade           *string   `gorm:"column:override_grade" json:"override_grade,omitempty"`
	OverrideRemarks         *string   `gorm:"column:override_remarks" json:"override_remarks,omitempty"`
	OverrideBy              *string   `gorm:"column:override_by" json:"override_by,omitempty"`
	OverrideAt              *time.Time `gorm:"column:override_at" json:"override_at,omitempty"`
	ComputedAt              time.Time  `gorm:"column:computed_at" json:"computed_at"`
	DataThroughAt           time.Time  `gorm:"column:data_through_at" json:"data_through_at"`
	LogicVersion            string     `gorm:"column:logic_version" json:"logic_version"`
	FormulaOTD              string     `gorm:"column:formula_otd" json:"formula_otd"`
	FormulaQuality          string     `gorm:"column:formula_quality" json:"formula_quality"`
	FormulaGrade            string     `gorm:"column:formula_grade" json:"formula_grade"`
	FormulaNotes            *string    `gorm:"column:formula_notes" json:"formula_notes,omitempty"`
}

func (Snapshot) TableName() string { return "supplier_performance_snapshots" }

// Flags returns the list of active flag strings for the UI.
func (s Snapshot) Flags() []string {
	var flags []string
	if s.PoorDeliveryPerformance {
		flags = append(flags, "poor_delivery_performance")
	}
	if s.QCAlert {
		flags = append(flags, "qc_alert")
	}
	return flags
}

// SnapshotResponse is the JSON shape sent to frontend.
type SnapshotResponse struct {
	SupplierID              string    `json:"supplier_id"`
	SupplierCode            string    `json:"supplier_code"`
	SupplierName            string    `json:"supplier_name"`
	EvaluationPeriodType    string    `json:"evaluation_period_type"`
	EvaluationPeriodValue   string    `json:"evaluation_period_value"`
	TotalDeliveries         int       `json:"total_deliveries"`
	OnTimeDeliveries        int       `json:"on_time_deliveries"`
	LateDeliveries          int       `json:"late_deliveries"`
	OTDPercentage           float64   `json:"otd_percentage"`
	AverageDelayDays        float64   `json:"average_delay_days"`
	QualityInspectionCount  int       `json:"quality_inspection_count"`
	AcceptedQuantity        float64   `json:"accepted_quantity"`
	RejectedQuantity        float64   `json:"rejected_quantity"`
	InspectedQuantity       float64   `json:"inspected_quantity"`
	QualityPercentage       float64   `json:"quality_percentage"`
	TotalPurchaseValue      float64   `json:"total_purchase_value"`
	PerformanceGrade        string    `json:"performance_grade"`
	StatusLabel             string    `json:"status_label"`
	Flags                   []string  `json:"flags"`
	SupplierReviewRequired  bool      `json:"supplier_review_required"`
	IsGradeOverridden       bool      `json:"is_grade_overridden"`
	OverrideGrade           *string   `json:"override_grade,omitempty"`
	OverrideRemarks         *string   `json:"override_remarks,omitempty"`
	LogicVersion            string    `json:"logic_version"`
	FormulaOTD              string    `json:"formula_otd"`
	FormulaQuality          string    `json:"formula_quality"`
	FormulaGrade            string    `json:"formula_grade"`
	FormulaNotes            *string   `json:"formula_notes,omitempty"`
	EvaluationDate          time.Time `json:"evaluation_date"`
}

// ListResponse wraps items + pagination meta.
type ListResponse struct {
	Items      []SnapshotResponse `json:"items"`
	Pagination PaginationMeta     `json:"pagination"`
}

// PaginationMeta mirrors other modules.
type PaginationMeta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

// SummaryResponse is returned by GET /summary.
type SummaryResponse struct {
	ExcellentSuppliers      int     `json:"excellent_suppliers"`
	GoodSuppliers           int     `json:"good_suppliers"`
	ReviewRequiredSuppliers int     `json:"review_required_suppliers"`
	TotalSuppliersEvaluated int     `json:"total_suppliers_evaluated"`
	TotalPurchaseValue      float64 `json:"total_purchase_value"`
	LogicVersion            string  `json:"logic_version"`
	FormulaGrade            string  `json:"formula_grade"`
	ComputedAt              time.Time `json:"computed_at"`
}

// AuditLog is one entry in GET /:supplier_id/audit-logs.
type AuditLog struct {
	Action                string    `gorm:"column:action" json:"action"`
	OldGrade              *string   `gorm:"column:old_grade" json:"old_grade,omitempty"`
	NewGrade              *string   `gorm:"column:new_grade" json:"new_grade,omitempty"`
	Remarks               *string   `gorm:"column:remarks" json:"remarks,omitempty"`
	LogicVersion          *string   `gorm:"column:logic_version" json:"logic_version,omitempty"`
	Actor                 *string   `gorm:"column:actor" json:"actor,omitempty"`
	OccurredAt            time.Time `gorm:"column:occurred_at" json:"occurred_at"`
}

// OverrideRequest is the POST /:supplier_id/override body.
type OverrideRequest struct {
	SupplierUUID  string `json:"-"`
	PeriodType    string `json:"period_type" validate:"required,oneof=monthly quarterly yearly"`
	PeriodValue   string `json:"period_value" validate:"required"`
	OverrideGrade string `json:"override_grade" validate:"required,oneof=A B C"`
	Remarks       string `json:"override_remarks" validate:"required"`
}

// ChartsResponse is returned by GET /charts.
type ChartsResponse struct {
	Trend   []ChartTrendPoint   `json:"trend"`
	Scatter []ChartScatterPoint `json:"scatter"`
	Top5    []ChartRankPoint    `json:"top_5"`
	Bottom5 []ChartRankPoint    `json:"bottom_5"`
}

// ChartTrendPoint is one period in the trend line.
type ChartTrendPoint struct {
	Period               string  `json:"period" gorm:"column:period"`
	AvgOTDPercentage     float64 `json:"avg_otd_percentage" gorm:"column:avg_otd_percentage"`
	AvgQualityPercentage float64 `json:"avg_quality_percentage" gorm:"column:avg_quality_percentage"`
}

// ChartScatterPoint is one supplier dot in the scatter chart.
type ChartScatterPoint struct {
	SupplierID        string  `json:"supplier_id" gorm:"column:supplier_uuid"`
	SupplierName      string  `json:"supplier_name" gorm:"column:supplier_name"`
	OTDPercentage     float64 `json:"otd_percentage" gorm:"column:otd_percentage"`
	QualityPercentage float64 `json:"quality_percentage" gorm:"column:quality_percentage"`
	StatusLabel       string  `json:"status_label" gorm:"column:status_label"`
}

// ChartRankPoint is one supplier in top/bottom ranking.
type ChartRankPoint struct {
	SupplierID       string  `json:"supplier_id" gorm:"column:supplier_uuid"`
	SupplierName     string  `json:"supplier_name" gorm:"column:supplier_name"`
	PerformanceGrade string  `json:"performance_grade" gorm:"column:final_grade"`
	StatusLabel      string  `json:"status_label" gorm:"column:status_label"`
	Score            float64 `json:"score" gorm:"column:computed_score"`
}

// ExportResponse is returned by GET /export.
type ExportResponse struct {
	PeriodType     string             `json:"period_type"`
	PeriodValue    string             `json:"period_value"`
	LogicVersion   string             `json:"logic_version"`
	FormulaOTD     string             `json:"formula_otd"`
	FormulaQuality string             `json:"formula_quality"`
	FormulaGrade   string             `json:"formula_grade"`
	ExportedAt     time.Time          `json:"exported_at"`
	Items          []SnapshotResponse `json:"items"`
}
