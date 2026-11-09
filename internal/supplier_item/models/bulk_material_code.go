package models

// MaterialCodeUniq adalah satu uniq (item BOM) yang material code-nya cocok.
// "Material code" = item_material_specs.material_grade, sama persis dengan
// yang tampil di halaman detail BOM dan di kolom Material Code master-supplier.
type MaterialCodeUniq struct {
	UniqCode      string   `gorm:"column:uniq_code" json:"uniq_code"`
	PartNumber    *string  `gorm:"column:part_number" json:"part_number,omitempty"`
	PartName      string   `gorm:"column:part_name" json:"part_name"`
	Model         *string  `gorm:"column:model" json:"model,omitempty"`
	UOM           string   `gorm:"column:uom" json:"uom,omitempty"`
	MaterialCode  string   `gorm:"column:material_code" json:"material_code"`
	Grade         *string  `gorm:"column:grade" json:"grade,omitempty"`
	Form          *string  `gorm:"column:form" json:"form,omitempty"`
	WidthMm       *float64 `gorm:"column:width_mm" json:"width_mm,omitempty"`
	DiameterMm    *float64 `gorm:"column:diameter_mm" json:"diameter_mm,omitempty"`
	ThicknessMm   *float64 `gorm:"column:thickness_mm" json:"thickness_mm,omitempty"`
	LengthMm      *float64 `gorm:"column:length_mm" json:"length_mm,omitempty"`
	WeightKg      *float64 `gorm:"column:weight_kg" json:"weight_kg,omitempty"`
	AlreadyListed bool     `gorm:"-" json:"already_registered"`
}

// MaterialCodePreviewQuery dipakai untuk melihat uniq mana saja yang akan ikut.
type MaterialCodePreviewQuery struct {
	SupplierUUID string `form:"supplier_uuid"`
	MaterialCode string `form:"material_code"`
	Type         string `form:"type"` // raw_material | indirect
}

// BulkByMaterialCodeRequest mendaftarkan SEMUA uniq ber-material-code tertentu
// ke satu supplier sekaligus.
type BulkByMaterialCodeRequest struct {
	SupplierUUID  string `json:"supplier_uuid" validate:"required,uuid4"`
	MaterialCode  string `json:"material_code" validate:"required,max=64"`
	Type          string `json:"type" validate:"required,oneof=raw_material indirect"`
	WarehouseUUID string `json:"warehouse_uuid" validate:"omitempty,uuid"`
	WarehouseName string `json:"warehouse_name" validate:"omitempty,max=255"`
	Quantity      string `json:"quantity" validate:"omitempty,max=50"`
	UOM           string `json:"uom" validate:"omitempty,max=32"`
	Status        string `json:"status" validate:"required,oneof=active inactive"`
	// Opsional: batasi ke subset uniq (mis. user menghapus centang beberapa uniq).
	// Kosong = semua uniq dengan material code tsb.
	SelectedUniqCodes []string `json:"selected_uniq_codes"`
}

type BulkByMaterialCodeResult struct {
	MaterialCode     string   `json:"material_code"`
	Matched          int      `json:"matched"`
	Created          int      `json:"created"`
	Skipped          int      `json:"skipped"`
	CreatedUniqCodes []string `json:"created_uniq_codes"`
	SkippedUniqCodes []string `json:"skipped_uniq_codes"`
}
