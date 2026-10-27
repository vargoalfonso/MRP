package models

import (
	"encoding/json"
	"strings"
)

// EntrySupplier is one supplier allocation of a PO budget entry.
//
// An entry can be fulfilled by several suppliers (the entry-level supplier_name
// then reads "Multiple"). This list tells which suppliers and how much each one
// supplies, for every budget type (raw_material | indirect | subcon).
type EntrySupplier struct {
	SupplierID   *int64  `json:"supplier_id"`
	SupplierName string  `json:"supplier_name"`
	Quantity     float64 `json:"quantity"`
}

type entrySupplierDetail struct {
	Suppliers []detailSupplier `json:"suppliers"`
	Children  []detailChild    `json:"children"`
	Parents   []struct {
		Children []detailChild `json:"children"`
	} `json:"parents"`
}

type detailChild struct {
	Quantity  float64          `json:"quantity"`
	Suppliers []detailSupplier `json:"suppliers"`
}

type detailSupplier struct {
	SupplierID   *int64  `json:"supplier_id"`
	SupplierName string  `json:"supplier_name"`
	Quantity     float64 `json:"quantity"`
}

// BuildEntrySuppliers returns the supplier breakdown of an entry.
//
// Source order:
//  1. detail_jsonb parents[].children[].suppliers (or the flat children[] list)
//     aggregated per supplier, which is the same breakdown raw_material and
//     indirect entries have always carried;
//  2. detail_jsonb top-level suppliers[];
//  3. the entry-level supplier (single supplier, not "Multiple").
//
// The result is never nil so it serialises as [] instead of null.
func BuildEntrySuppliers(e POBudgetEntry) []EntrySupplier {
	out := make([]EntrySupplier, 0)
	index := map[string]int{}

	add := func(id *int64, name string, qty float64) {
		name = strings.TrimSpace(name)
		if name == "" && (id == nil || *id <= 0) {
			return
		}
		key := "name:" + strings.ToLower(name)
		if id != nil && *id > 0 {
			key = "id:" + int64Key(*id)
		}
		if i, ok := index[key]; ok {
			out[i].Quantity += qty
			if out[i].SupplierName == "" {
				out[i].SupplierName = name
			}
			return
		}
		index[key] = len(out)
		out = append(out, EntrySupplier{SupplierID: id, SupplierName: name, Quantity: qty})
	}

	raw := []byte(e.DetailJSON)
	if len(raw) > 0 && string(raw) != "{}" && string(raw) != "null" {
		var detail entrySupplierDetail
		if err := json.Unmarshal(raw, &detail); err == nil {
			children := make([]detailChild, 0)
			if len(detail.Parents) > 0 {
				for _, p := range detail.Parents {
					children = append(children, p.Children...)
				}
			} else {
				children = detail.Children
			}
			for _, c := range children {
				for _, s := range c.Suppliers {
					add(s.SupplierID, s.SupplierName, s.Quantity)
				}
			}
			if len(out) == 0 {
				for _, s := range detail.Suppliers {
					add(s.SupplierID, s.SupplierName, s.Quantity)
				}
			}
		}
	}

	if len(out) == 0 && e.SupplierName != nil {
		name := strings.TrimSpace(*e.SupplierName)
		if name != "" && !strings.EqualFold(name, "Multiple") {
			qty := e.PurchaseRequest
			if qty <= 0 {
				qty = e.Quantity
			}
			add(e.SupplierID, name, qty)
		}
	}
	return out
}

func int64Key(v int64) string {
	b, _ := json.Marshal(v)
	return string(b)
}
