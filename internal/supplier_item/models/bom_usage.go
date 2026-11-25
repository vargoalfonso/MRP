package models

// BomUsageRow menjelaskan di FG / parent mana sebuah uniq (child part) dipakai
// pada BOM current. Dipakai untuk traceability di halaman master supplier.
//
// Level 0 = uniq tersebut adalah root BOM (FG) itu sendiri.
type BomUsageRow struct {
	ChildUniqCode  string  `gorm:"column:child_uniq_code" json:"child_uniq_code"`
	ChildPartName  string  `gorm:"column:child_part_name" json:"child_part_name"`
	ParentUniqCode *string `gorm:"column:parent_uniq_code" json:"parent_uniq_code,omitempty"`
	ParentPartName *string `gorm:"column:parent_part_name" json:"parent_part_name,omitempty"`
	FGUniqCode     string  `gorm:"column:fg_uniq_code" json:"fg_uniq_code"`
	FGPartNumber   *string `gorm:"column:fg_part_number" json:"fg_part_number,omitempty"`
	FGPartName     string  `gorm:"column:fg_part_name" json:"fg_part_name"`
	FGModel        *string `gorm:"column:fg_model" json:"fg_model,omitempty"`
	Level          int     `gorm:"column:level" json:"level"`
	QtyPerUniq     float64 `gorm:"column:qty_per_uniq" json:"qty_per_uniq"`
	BomVersion     int     `gorm:"column:bom_version" json:"bom_version"`
	BomStatus      string  `gorm:"column:bom_status" json:"bom_status"`
}

// BomMaterialGrade adalah satu material code (item_material_specs.material_grade)
// yang ada di BOM, lengkap dengan info apakah sudah punya Raw Material Master
// (Active). HasMaster=false berarti material itu ada di BOM tapi tidak akan
// muncul di dropdown yang bersumber dari master.
type BomMaterialGrade struct {
	MaterialGrade string  `gorm:"column:material_grade" json:"material_grade"`
	UniqCount     int64   `gorm:"column:uniq_count" json:"uniq_count"`
	Form          *string `gorm:"column:form" json:"form,omitempty"`
	HasMaster     bool    `gorm:"column:has_master" json:"has_master"`
}

// BomUsageQuery: uniq_codes dipisah koma, mis. ?uniq_codes=BM4,BM6
type BomUsageQuery struct {
	UniqCodes string `form:"uniq_codes"`
}

// BomMaterialGradeQuery: type = raw_material | indirect.
type BomMaterialGradeQuery struct {
	Type   string `form:"type"`
	Search string `form:"search"`
}
