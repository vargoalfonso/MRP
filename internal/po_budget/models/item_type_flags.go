package models

// ItemTypeFlags adalah flag klasifikasi sebuah uniq dari item_material_specs
// (revisi BOM current). Satu uniq bisa sekaligus bertipe raw/indirect DAN
// is_subcon=true; dalam kasus itu ia masuk ke dua tab PO Budget.
type ItemTypeFlags struct {
	TypeMaterial string // nilai mentah type_material (lowercase, trimmed); "" bila kosong
	IsSubcon     bool
}
