package service

import (
	"strings"

	"github.com/ganasa18/go-template/internal/po_budget/models"
)

// budgetTypesFromFlags menentukan SEMUA tab/budget_type tempat sebuah uniq harus
// muncul, berdasarkan type_material dan is_subcon:
//
//   - type_material raw      -> raw_material
//   - type_material indirect -> indirect
//   - is_subcon = true       -> subcon (ditambahkan, tidak menggantikan di atas)
//
// Contoh: raw + is_subcon=true menghasilkan [raw_material, subcon] sehingga item
// muncul di dua tab. Bila tidak ada informasi sama sekali, default subcon
// (perilaku lama).
func budgetTypesFromFlags(f models.ItemTypeFlags) []string {
	out := make([]string, 0, 2)
	add := func(t string) {
		for _, e := range out {
			if e == t {
				return
			}
		}
		out = append(out, t)
	}
	tm := strings.ToLower(strings.TrimSpace(f.TypeMaterial))
	switch tm {
	case "raw", "raw_material":
		add(budgetTypeRawMaterial)
	case "indirect":
		add(budgetTypeIndirect)
	}
	if f.IsSubcon || tm == "subcon" {
		add(budgetTypeSubcon)
	}
	if len(out) == 0 {
		add(budgetTypeSubcon)
	}
	return out
}
