package models

import "time"

// RawMaterialMaster is the canonical material specification reused by many BOM items.
type RawMaterialMaster struct {
	ID            int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	MaterialCode  string     `json:"material_code" gorm:"size:64;not null"`
	MaterialName  string     `json:"material_name" gorm:"size:255;not null"`
	MaterialGrade *string    `json:"material_grade" gorm:"size:100"`
	Form          *string    `json:"form" gorm:"size:32"`
	TypeMaterial  string     `json:"type_material" gorm:"size:50;not null;default:raw"`
	WidthMM       *float64   `json:"width_mm" gorm:"column:width_mm;type:numeric(18,4)"`
	DiameterMM    *float64   `json:"diameter_mm" gorm:"column:diameter_mm;type:numeric(18,4)"`
	ThicknessMM   *float64   `json:"thickness_mm" gorm:"column:thickness_mm;type:numeric(18,4)"`
	LengthMM      *float64   `json:"length_mm" gorm:"column:length_mm;type:numeric(18,4)"`
	WeightKG      *float64   `json:"weight_kg" gorm:"column:weight_kg;type:numeric(18,6)"`
	UOM           *string    `json:"uom" gorm:"column:uom;size:32"`
	SpecKey       string     `json:"spec_key" gorm:"size:512;not null"`
	Status        string     `json:"status" gorm:"size:20;not null;default:Active"`
	CreatedBy     *string    `json:"created_by" gorm:"size:255"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedBy     *string    `json:"updated_by" gorm:"size:255"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"-" gorm:"index"`
	BOMUsageCount int64      `json:"bom_usage_count" gorm:"-"`
	InventoryQty  float64    `json:"inventory_qty" gorm:"-"`
}

func (RawMaterialMaster) TableName() string { return "raw_material_masters" }

type CreateRawMaterialMasterRequest struct {
	MaterialCode  string   `json:"material_code" validate:"required,max=64"`
	MaterialName  string   `json:"material_name" validate:"required,max=255"`
	MaterialGrade *string  `json:"material_grade"`
	Form          *string  `json:"form"`
	TypeMaterial  string   `json:"type_material" validate:"omitempty,oneof=raw indirect subcon"`
	WidthMM       *float64 `json:"width_mm"`
	DiameterMM    *float64 `json:"diameter_mm"`
	ThicknessMM   *float64 `json:"thickness_mm"`
	LengthMM      *float64 `json:"length_mm"`
	WeightKG      *float64 `json:"weight_kg"`
	UOM           *string  `json:"uom"`
	Status        string   `json:"status" validate:"omitempty,oneof=Active Inactive"`
}

type UpdateRawMaterialMasterRequest struct {
	MaterialCode  *string  `json:"material_code"`
	MaterialName  *string  `json:"material_name"`
	MaterialGrade *string  `json:"material_grade"`
	Form          *string  `json:"form"`
	TypeMaterial  *string  `json:"type_material"`
	WidthMM       *float64 `json:"width_mm"`
	DiameterMM    *float64 `json:"diameter_mm"`
	ThicknessMM   *float64 `json:"thickness_mm"`
	LengthMM      *float64 `json:"length_mm"`
	WeightKG      *float64 `json:"weight_kg"`
	UOM           *string  `json:"uom"`
	Status        *string  `json:"status"`
}

type RawMaterialMasterList struct {
	Items []RawMaterialMaster `json:"items"`
	Total int64               `json:"total"`
}

type RawMaterialPlanningItem struct {
	MasterID          int64   `json:"master_id" gorm:"column:master_id"`
	MaterialCode      string  `json:"material_code" gorm:"column:material_code"`
	MaterialName      string  `json:"material_name" gorm:"column:material_name"`
	MaterialGrade     *string `json:"material_grade" gorm:"column:material_grade"`
	UOM               *string `json:"uom" gorm:"column:uom"`
	DailyDemand       float64 `json:"daily_demand" gorm:"column:daily_demand"`
	BeginningStock    float64 `json:"beginning_stock" gorm:"column:beginning_stock"`
	EndingStock       float64 `json:"ending_stock" gorm:"column:ending_stock"`
	SafetyStock       float64 `json:"safety_stock" gorm:"column:safety_stock"`
	TargetStockDays   float64 `json:"target_stock_days" gorm:"column:target_stock_days"`
	ActualStockDays   float64 `json:"actual_stock_days" gorm:"column:actual_stock_days"`
	MinimumStockLevel float64 `json:"minimum_stock_level" gorm:"column:minimum_stock_level"`
	RecommendedBuyQty float64 `json:"recommended_buy_qty" gorm:"column:recommended_buy_qty"`
	Decision          string  `json:"decision" gorm:"column:decision"`
	DemandSourceCount int64   `json:"demand_source_count" gorm:"column:demand_source_count"`
}
