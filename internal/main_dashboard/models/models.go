package models

import "time"

type DateRange struct {
	StartDate time.Time
	EndDate   time.Time
}

type PeriodMeta struct {
	Type      string `json:"type"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type Summary struct {
	AsOf                  time.Time             `json:"as_of"`
	Period                PeriodMeta            `json:"period"`
	KPIs                  KPIBundle             `json:"kpis"`
	DeliveryPerformance   DeliveryPerformance   `json:"delivery_performance"`
	ProductionPerformance ProductionPerformance `json:"production_performance"`
	TopCustomers          []TopCustomer         `json:"top_customers"`
	CurrentUniqProgress   []UniqProgress        `json:"current_uniq_progress"`
}

type KPIBundle struct {
	TotalDeliveries   DeliveryKPI          `json:"total_deliveries"`
	CurrentProduction CurrentProductionKPI `json:"current_production"`
	TotalProduction   TotalProductionKPI   `json:"total_production"`
	PORawMaterial     PORawMaterialKPI     `json:"po_raw_material"`
}

type DeliveryKPI struct {
	Value        int64   `json:"value"`
	Subtitle     string  `json:"subtitle"`
	DeltaPercent float64 `json:"delta_percent"`
	DeltaLabel   string  `json:"delta_label"`
	Trend        string  `json:"trend"`
}

type CurrentProductionKPI struct {
	Value        int64   `json:"value"`
	Subtitle     string  `json:"subtitle"`
	DeltaPercent float64 `json:"delta_percent"`
	DeltaLabel   string  `json:"delta_label"`
	Trend        string  `json:"trend"`
}

type TotalProductionKPI struct {
	Value        float64 `json:"value"`
	Subtitle     string  `json:"subtitle"`
	DeltaPercent float64 `json:"delta_percent"`
	DeltaLabel   string  `json:"delta_label"`
	Trend        string  `json:"trend"`
}

type PORawMaterialKPI struct {
	Value      int64  `json:"value"`
	Subtitle   string `json:"subtitle"`
	DeltaValue int64  `json:"delta_value"`
	DeltaLabel string `json:"delta_label"`
	Trend      string `json:"trend"`
}

type DeliveryPerformance struct {
	TotalDeliveries   int64                `json:"total_deliveries"`
	TotalValue        float64              `json:"total_value"`
	OnTimeRatePercent float64              `json:"on_time_rate_percent"`
	OnTimeCount       int64                `json:"on_time_count"`
	Trend             []DeliveryTrendPoint `json:"trend"`
}

type DeliveryTrendPoint struct {
	Label  string `json:"label"`
	Actual int64  `json:"actual"`
	Target int64  `json:"target"`
}

type ProductionPerformance struct {
	CurrentProduction int64                  `json:"current_production"`
	CapacityPercent   float64                `json:"capacity_percent"`
	TotalProduction   float64                `json:"total_production"`
	QualityPercent    float64                `json:"quality_percent"`
	Trend             []ProductionTrendPoint `json:"trend"`
}

type ProductionTrendPoint struct {
	Label    string  `json:"label"`
	Produced float64 `json:"produced"`
	Target   float64 `json:"target"`
}

type TopCustomer struct {
	CustomerID    int64   `json:"customer_id"`
	CustomerName  string  `json:"customer_name"`
	DeliveryCount int64   `json:"delivery_count"`
	SharePercent  float64 `json:"share_percent"`
	Status        string  `json:"status"`
}

type UniqProgress struct {
	WONumber        string  `json:"wo_number"`
	UniqCode        string  `json:"uniq_code"`
	ProducedQty     float64 `json:"produced_qty"`
	TargetQty       float64 `json:"target_qty"`
	ProgressPercent float64 `json:"progress_percent"`
	Status          string  `json:"status"`
}

type RawMaterialSummary struct {
	AsOf                 time.Time              `json:"as_of"`
	POSummary            POSummary              `json:"po_summary"`
	CategoryDistribution []CategoryDistribution `json:"category_distribution"`
	TopSuppliers         []TopSupplier          `json:"top_suppliers"`
}

type POSummary struct {
	TotalPOs       int64                 `json:"total_pos"`
	TotalValue     float64               `json:"total_value"`
	LowStockAlerts int64                 `json:"low_stock_alerts"`
	CriticalAlerts int64                 `json:"critical_alerts"`
	MonthlyTrend   []POMonthlyTrendPoint `json:"monthly_trend"`
}

type POMonthlyTrendPoint struct {
	Label    string  `json:"label"`
	Ordered  float64 `json:"ordered"`
	Received float64 `json:"received"`
}

type CategoryDistribution struct {
	Category     string  `json:"category"`
	SharePercent float64 `json:"share_percent"`
}

type TopSupplier struct {
	SupplierUUID   string  `json:"supplier_uuid"`
	SupplierCode   string  `json:"supplier_code"`
	SupplierName   string  `json:"supplier_name"`
	OnTimePercent  float64 `json:"on_time_percent"`
	QualityPercent float64 `json:"quality_percent"`
	Grade          string  `json:"grade"`
}

type ListTablesResponse struct {
	Schema string      `json:"schema"`
	Count  int64       `json:"count"`
	Tables []TableInfo `json:"tables"`
}

type TableInfo struct {
	Name        string  `json:"name"`
	RowEstimate float64 `json:"row_estimate"`
}
